package metrics

import (
	"net/http"

	"github.com/patbaumgartner/watchtower/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler is an HTTP handle for serving metric data
type Handler struct {
	Path   string
	Handle http.HandlerFunc
}

// New is a factory function creating a new Metrics instance
func New() *Handler {
	metrics.Default()
	handler := promhttp.Handler()

	return &Handler{
		Path:   "/v1/metrics",
		Handle: handler.ServeHTTP,
	}
}
