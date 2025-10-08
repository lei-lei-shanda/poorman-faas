// Package pruner prunes k8s resources that are not needed anymore.
//
// This is similiar to Knative serving, that scales down to zero pods.
// https://knative.dev/docs/serving/autoscaling/pruning/
package pruner

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

// Pruner is a service that prunes resources that have expired.
type Pruner struct {
	clientset *kubernetes.Clientset
	expirer   Expirer
	logger    *slog.Logger
	// mapping of UUID to Helm Chart
	mu      sync.RWMutex
	mapping map[string]Charter
}

// NewPruner creates a new Pruner with the given clientset and time to live.
func NewPruner(ctx context.Context, clientset *kubernetes.Clientset, timeToLive time.Duration, logger *slog.Logger) *Pruner {
	// TODO: iterate over all service in the namespace
	p := Pruner{
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
func (p *Pruner) Watch(ctx context.Context) {
	// TODO: iterate over all services in the namespace to initialize the mapping
	for {
		select {
		case <-ctx.Done():
			// clean up on exit
			services := p.expirer.Expire(ctx)
			p.MustPrune(ctx, services)
			return
		case <-time.After(5 * time.Minute):
			services := p.expirer.Expire(ctx)
			p.MustPrune(ctx, services)
		}
	}
}

func (p *Pruner) MustRegister(ctx context.Context, service string, chart Charter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.mapping[service]; !exists {
		p.mapping[service] = chart
	}

	err := p.expirer.Update(ctx, service)
	if err != nil {
		p.logger.Error("pruner.MustRegister", "error", err, "service", service)
		return
	}
}

func (p *Pruner) MustUpdate(ctx context.Context, service string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.mapping[service]; !exists {
		return
	}
	err := p.expirer.Update(ctx, service)
	if err != nil {
		p.logger.Error("pruner.MustUpdate", "error", err, "service", service)
		return
	}
}

func (p *Pruner) MustPrune(ctx context.Context, services []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, service := range services {
		chart, exists := p.mapping[service]
		if !exists {
			p.logger.Error("pruner.prune", "error", fmt.Errorf("service %s not found", service), "service", service)
			continue
		}

		err := chart.Teardown(ctx, p.clientset)
		if err != nil {
			p.logger.Error("pruner.prune", "error", err, "service", service)
			continue
		}
		delete(p.mapping, service)
	}
}
