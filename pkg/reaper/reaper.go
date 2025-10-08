// Package reaper culls k8s resources that are not needed anymore.
//
// This is similiar to Knative serving, that scales down to zero pods.
// https://knative.dev/docs/serving/autoscaling/pruning/
package reaper

import (
	"context"
	"fmt"
	"log/slog"
	"poorman-faas/pkg/util"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

type Charter interface {
	Teardown(ctx context.Context, clientset *kubernetes.Clientset) error
}

// Reaper is a service that prunes resources that have expired.
type Reaper struct {
	clientset *kubernetes.Clientset
	expirer   Expirer
	logger    *slog.Logger
	// mapping of UUID to Helm Chart
	mu      sync.RWMutex
	mapping map[string]Charter
}

// NewReaper creates a new Pruner with the given clientset and time to live.
func NewReaper(ctx context.Context, clientset *kubernetes.Clientset, timeToLive time.Duration, logger *slog.Logger) *Reaper {
	// TODO: iterate over all service in the namespace
	p := Reaper{
		clientset: clientset,
		expirer:   NewPQExpirer(timeToLive),
		mapping:   make(map[string]Charter),
		logger:    logger,
	}
	util.MustGo(func() {
		p.Watch(ctx)
	})
	return &p
}

// Watch starts a background goroutine that prunes resources that have expired.
func (p *Reaper) Watch(ctx context.Context) {
	// TODO: iterate over all services in the namespace to initialize the mapping
	for {
		select {
		case <-ctx.Done():
			// clean up on exit
			services := p.expirer.Expire(ctx)
			p.MustCull(ctx, services)
			return
		case <-time.After(5 * time.Minute):
			services := p.expirer.Expire(ctx)
			p.MustCull(ctx, services)
		}
	}
}

func (p *Reaper) MustRegister(ctx context.Context, service string, chart Charter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.mapping[service]; !exists {
		p.mapping[service] = chart
	}

	err := p.expirer.Update(ctx, service)
	if err != nil {
		p.logger.Error("Reaper.MustRegister", "error", err, "service", service)
		return
	}
}

func (p *Reaper) MustUpdate(ctx context.Context, service string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.mapping[service]; !exists {
		p.logger.Error("Reaper.mapping[service]", "error", fmt.Errorf("service %s not found", service), "service", service)
		return
	}
	err := p.expirer.Update(ctx, service)
	if err != nil {
		p.logger.Error("Reaper.expirer.Update()", "error", err, "service", service)
		return
	}
}

func (p *Reaper) MustCull(ctx context.Context, services []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, service := range services {
		chart, exists := p.mapping[service]
		if !exists {
			p.logger.Error("Reaper.mapping[service]", "error", fmt.Errorf("service %s not found", service), "service", service)
			continue
		}

		err := chart.Teardown(ctx, p.clientset)
		if err != nil {
			p.logger.Error("chart.Teardown()", "error", err, "service", service)
			continue
		}
		delete(p.mapping, service)
	}
}
