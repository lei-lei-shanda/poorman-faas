// Package main is what gets deployed to the cloud platform.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"poorman-faas/pkg/helm"
	"poorman-faas/pkg/proxy"
	"poorman-faas/pkg/pruner"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/httprate"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type UploadOption struct {
	User    string `json:"user"`
	Replica int    `json:"replica"`
}

type UploadRequest struct {
	Script string       `json:"script"`
	Option UploadOption `json:"option"`
}

type UploadResponse struct {
	URL     string `json:"url"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func getUploadHandler(k8sNamespace string, reaper *pruner.Pruner, client *kubernetes.Clientset) http.HandlerFunc {
	writeErrorResponse := func(w http.ResponseWriter, statusCode int, err error) {
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(UploadResponse{
			Code:    statusCode,
			Message: err.Error(),
		})
	}

	hanlder := func(w http.ResponseWriter, r *http.Request) {
		var req UploadRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// create a helm chart
		chart, err := helm.NewChart(k8sNamespace, req.Script)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// deploy the chart
		err = chart.Deploy(r.Context(), client)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, err)
			// TODO: running `defer chart.Teardown(r.Context(), client)` in the background
			return
		}

		// update the reaper
		reaper.MustRegister(r.Context(), chart.Service().Name, &chart)

		// TODO: wait until service is ready
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(UploadResponse{
			URL:     fmt.Sprintf("http://%s.%s.svc.cluster.local", chart.Service().Name, chart.Namespace),
			Code:    http.StatusOK,
			Message: "success",
		})
	}
	return hanlder
}

func run(ctx context.Context, logger *slog.Logger, port int, client *kubernetes.Clientset) error {
	// initialize the reaper
	reaper := pruner.NewPruner(ctx, client, 10*time.Minute, logger)

	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(logger, nil))
	// admin routes: this creates faas service.
	{
		admin := chi.NewRouter()

		// because this creates k8s resource, we are extra careful.
		// for example, see e2b create sandbox rate limit at 5/second.
		admin.Use(httprate.LimitByIP(10, time.Minute))
		admin.Post("/python", getUploadHandler("faas", reaper, client))
		r.Mount("/admin", admin)
	}
	// gateway routes: this proxies to the faas service.
	{
		gateway := chi.NewRouter()
		namespace := "faas"
		getServiceName := func(r *http.Request) string {
			// return r.PathValue("svcName")
			return chi.URLParam(r, "svcName")
		}
		rp, err := proxy.New(
			proxy.WithTransport(proxy.ProxyTransport()),
			proxy.WithRewrites(
				proxy.RewriteURL("gateway", namespace, getServiceName),
				proxy.DebugRequest(logger),
			),
			proxy.WithModifyResponse(func(r *http.Response) error {
				svcName := getServiceName(r.Request)
				reaper.MustUpdate(r.Request.Context(), svcName)
				return nil
			}),
		)
		if err != nil {
			return fmt.Errorf("proxy.New(): %w", err)
		}
		gateway.Handle("/{svcName}/*", rp)
		r.Mount("/gateway", gateway)
	}
	// add health check route
	{
		r.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
	}

	// Single server listening on port 8080
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	// Start the single server
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed", "error", err)
			return
		}
	}()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Received shutdown signal, shutting down server...")

	// Create a context with a timeout for graceful shutdown
	shutdownCtx := context.Background()
	shutdownCtx, shutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down server", "error", err)
		return err
	}
	return nil
}

func main() {
	port, error := strconv.Atoi(os.Getenv("PORT"))
	if error != nil {
		panic(error)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Kubeconfig: %#v\n", config)
	// fmt.Printf("Kubernetes Cluster Host: %s\n", config.Host)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	err = run(ctx, logger, port, clientset)
	if err != nil {
		panic(err)
	}
}
