// Package proxy rewrites request before sending, and records states after receiving.
package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"
)

type Option func(rp *httputil.ReverseProxy) error
type Rewrite func(req *httputil.ProxyRequest)

// New creates a new ReverseProxy with the given options.
func New(opts ...Option) (*httputil.ReverseProxy, error) {
	rp := &httputil.ReverseProxy{}
	for _, opt := range opts {
		if err := opt(rp); err != nil {
			return nil, err
		}
	}
	return rp, nil
}

// WithTransport sets the transport for the ReverseProxy.
func WithTransport(transport http.RoundTripper) Option {
	return func(rp *httputil.ReverseProxy) error {
		rp.Transport = transport
		return nil
	}
}

// WithModifyResponse sets the modify response function for the ReverseProxy.
func WithModifyResponse(modifyResponse func(*http.Response) error) Option {
	return func(rp *httputil.ReverseProxy) error {
		rp.ModifyResponse = modifyResponse
		return nil
	}
}

// WithRewrites sets the rewrites for the ReverseProxy.
func WithRewrites(rewrites ...func(*httputil.ProxyRequest)) Option {
	final := func(req *httputil.ProxyRequest) {
		for _, rewrite := range rewrites {
			rewrite(req)
		}
	}
	return func(rp *httputil.ReverseProxy) error {
		rp.Rewrite = final
		return nil
	}
}

// WithErrorHandler sets the error handler for the ReverseProxy.
func WithErrorHandler(logger *slog.Logger) Option {
	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("proxy error occurred",
			"error", err,
			"url", r.URL.String(),
			"method", r.Method,
			"user_agent", r.UserAgent(),
			"remote_addr", r.RemoteAddr,
		)

		// Check if it's a context cancellation
		if errors.Is(r.Context().Err(), context.Canceled) {
			logger.Warn("request was canceled by client", "url", r.URL.String())
			http.Error(w, "request canceled", http.StatusRequestTimeout)
		} else {
			http.Error(w, "service temporarily unavailable", http.StatusBadGateway)
		}
	}
	return func(rp *httputil.ReverseProxy) error {
		rp.ErrorHandler = errorHandler
		return nil
	}
}

// DebugRequest dumps the request and response for debugging.
func DebugRequest(logger *slog.Logger) func(req *httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		dump, err := httputil.DumpRequest(req.In, false)
		if err != nil {
			logger.Debug("failed to dump request", "error", err)
			return
		}
		logger.Debug("incoming request", "dump", string(dump))
		dump, err = httputil.DumpRequest(req.Out, false)
		if err != nil {
			logger.Debug("failed to dump request", "error", err)
			return
		}
		logger.Debug("outgoing request", "dump", string(dump))
	}
}

// ProxyTransport creates a new transport for the ReverseProxy with extended timeouts and retry capabilities.
func ProxyTransport() *http.Transport {
	transport := &http.Transport{
		// Extended timeouts for better reliability
		// DialContext: (&net.Dialer{
		// 	Timeout:   60 * time.Second, // Increased from default 30s
		// 	KeepAlive: 60 * time.Second, // Increased keep-alive time
		// }).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       120 * time.Second, // Increased idle timeout
		TLSHandshakeTimeout:   15 * time.Second,  // Increased TLS handshake timeout
		ExpectContinueTimeout: 5 * time.Second,   // Increased expect continue timeout
		ResponseHeaderTimeout: 60 * time.Second,  // Added response header timeout
		DisableKeepAlives:     false,
		DisableCompression:    false,
	}
	return transport
}
