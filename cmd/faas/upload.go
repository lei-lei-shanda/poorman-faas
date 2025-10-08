package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"poorman-faas/pkg/helm"
	pkg_reaper "poorman-faas/pkg/reaper"

	"k8s.io/client-go/kubernetes"
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

type Charter struct {
	chart  *helm.Chart
	client *kubernetes.Clientset
}

func (c *Charter) Teardown(ctx context.Context) error {
	return c.chart.Teardown(ctx, c.client)
}

func getUploadHandler(k8sNamespace string, reaper *pkg_reaper.Reaper, client *kubernetes.Clientset) http.HandlerFunc {
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

		// wrap client with chart
		charter := Charter{
			chart:  &chart,
			client: client,
		}

		// update the reaper
		reaper.MustRegister(r.Context(), chart.Service().Name, &charter)

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
