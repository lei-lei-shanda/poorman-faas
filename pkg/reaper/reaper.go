// Package reaper culls k8s resources that are not needed anymore.
//
// This is similiar to Knative serving, that scales down to zero pods.
package reaper

import (
	"context"
	"fmt"
	"log/slog"
	"poorman-faas/pkg/util"
	"sync"
	"time"
)

type Charter interface {
	Teardown(ctx context.Context) error
}

// Reaper culls resources that have expired by monitoring the last accessed time.
type Reaper struct {
	expirer Expirer
	logger  *slog.Logger
	// mapping of UUID to Helm Chart
	mu      sync.RWMutex
	mapping map[string]Charter
}

// New creates a new Reaper with the given clientset and time to live.
func New(ctx context.Context, pollEvery time.Duration, timeToLive time.Duration, logger *slog.Logger) *Reaper {
	// TODO: iterate over all service in the namespace
	p := Reaper{
		expirer: NewPQExpirer(timeToLive),
		mapping: make(map[string]Charter),
		logger:  logger,
	}
	util.MustGo(func() {
		p.Watch(ctx, pollEvery)
	})
	return &p
}

// Watch starts a background goroutine that culls k8s resources that have expired.
func (p *Reaper) Watch(ctx context.Context, pollEvery time.Duration) {
	// TODO: iterate over all services in the namespace to initialize the mapping
	ticker := time.Tick(pollEvery)
	for {
		select {
		case <-ctx.Done():
			// clean up on exit
			services := p.expirer.Expire(ctx)
			p.MustCull(ctx, services)
			return
		// case <-time.After(5 * time.Minute):
		case now := <-ticker:
			p.logger.Debug("Reaper.Watch", "now", now)
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
		p.logger.Debug("Reaper.MustRegister", "service", service)
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

		err := chart.Teardown(ctx)
		if err != nil {
			p.logger.Error("chart.Teardown()", "error", err, "service", service)
			continue
		}
		delete(p.mapping, service)
		p.logger.Debug("Reaper.MustCull", "service", service)
	}
}
