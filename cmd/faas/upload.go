package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"poorman-faas/pkg"
	"poorman-faas/pkg/helm"
	pkg_reaper "poorman-faas/pkg/reaper"
	"poorman-faas/pkg/util"

	"k8s.io/client-go/kubernetes"
)

type UploadOption struct {
	User    string `json:"user"`
	Replica int    `json:"replica"`
}

type UploadRequest struct {
	Script  string       `json:"script"`
	DotFile string       `json:"dotFile"`
	Option  UploadOption `json:"option"`
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

func getUploadHandler(config pkg.Config, reaper *pkg_reaper.Reaper) http.HandlerFunc {
	k8sNamespace := config.K8sNamespace
	client := config.K8SClientset

	writeErrorResponse := func(w http.ResponseWriter, statusCode int, err error) {
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(UploadResponse{
			Code:    statusCode,
			Message: err.Error(),
		})
	}

	hanlder := func(w http.ResponseWriter, r *http.Request) {
		var req UploadRequest

		// validate user request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("json.NewDecoder().Decode(): %w", err))
			return
		}

		// create a helm chart
		chart, err := helm.NewChart(k8sNamespace, req.Script, req.DotFile)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("helm.NewChart(): %w", err))
			return
		}

		// deploy the chart
		err = chart.Deploy(r.Context(), client)
		if err != nil {
			// TODO: check error status of Teardown
			newErr := chart.Teardown(r.Context(), client)
			if newErr != nil {
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("chart.Deploy(): %w, chart.Teardown(): %w", err, newErr))
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("chart.Deploy(): %w", err))
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
		ip, err := util.K8sExternalDomainName(r.Context(), client, config.K8sLoadBalancerPort, config.GatewayServiceName, config.GatewayPathPrefix, config.K8sNamespace, chart.Service().Name)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("util.K8sExternalDomainName(): %w", err))
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(UploadResponse{
			URL:     ip,
			Code:    http.StatusOK,
			Message: "success",
		})
	}
	return hanlder
}
