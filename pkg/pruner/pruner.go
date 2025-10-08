// Package pruner prunes k8s resources that are not needed anymore.
//
// This is similiar to Knative serving, that scales down to zero pods.
// https://knative.dev/docs/serving/autoscaling/pruning/
package pruner

import (
	"context"
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
	// mapping of UUID to Helm Chart
	mu      sync.RWMutex
	mapping map[string]Charter
}

// NewPruner creates a new Pruner with the given clientset and time to live.
func NewPruner(ctx context.Context, clientset *kubernetes.Clientset, timeToLive time.Duration) *Pruner {
	// TODO: iterate over all service in the namespace
	p := Pruner{
		clientset: clientset,
		expirer:   NewPQExpirer(timeToLive),
		mapping:   make(map[string]Charter),
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
			p.prune(ctx, services)
			return
		case <-time.After(5 * time.Minute):
			services := p.expirer.Expire(ctx)
			p.prune(ctx, services)
		}
	}
}

func (p *Pruner) MustRegister(ctx context.Context, service string, chart Charter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.mapping[service]; !exists {
		p.mapping[service] = chart
	}

	p.expirer.Update(ctx, service)
}

func (p *Pruner) MustUpdate(ctx context.Context, service string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.mapping[service]; !exists {
		return
	}
	p.expirer.Update(ctx, service)
}

func (p *Pruner) prune(ctx context.Context, services []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, service := range services {
		chart, exists := p.mapping[service]
		if !exists {
			continue
		}

		err := chart.Teardown(ctx, p.clientset)
		if err != nil {
			continue
		}
		delete(p.mapping, service)
	}
}
