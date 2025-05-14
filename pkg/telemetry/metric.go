package telemetry

import (
	"context"
	"errors"
	"time"
)

// MetricsContext holds all metrics collected during command execution
type MetricsContext struct {
	StartTime   time.Time         `json:"start_time"`
	RootCommand string            `json:"root_command"`
	Subcommand  string            `json:"subcommand"`
	Metrics     []Metric          `json:"metrics"`
	Properties  map[string]string `json:"properties"`
}

// Metric represents a single metric with its value and dimensions
type Metric struct {
	Value      float64           `json:"value"`
	Name       string            `json:"name"`
	Dimensions map[string]string `json:"dimensions"`
}

// contextKey is used to store the metrics context
type metricsContextKey struct{}

// WithMetricsContext returns a new context with the metrics context
func WithMetricsContext(ctx context.Context, metrics *MetricsContext) context.Context {
	return context.WithValue(ctx, metricsContextKey{}, metrics)
}

// MetricsFromContext retrieves the metrics context
func MetricsFromContext(ctx context.Context) (*MetricsContext, error) {
	metrics, ok := ctx.Value(metricsContextKey{}).(*MetricsContext)
	if !ok {
		return &MetricsContext{}, errors.New("no metrics context")
	}
	return metrics, nil
}

// NewMetricsContext creates a new metrics context
func NewMetricsContext(rootCommand, subcommand string) *MetricsContext {
	return &MetricsContext{
		StartTime:   time.Now(),
		RootCommand: rootCommand,
		Subcommand:  subcommand,
		Metrics:     make([]Metric, 0),
	}
}

// AddMetric adds a new metric to the context without dimensions
func (m *MetricsContext) AddMetric(name string, value float64) {
	m.AddMetricWithDimensions(name, value, make(map[string]string))
}

// AddMetricWithDimensions adds a new metric to the context with dimensions
func (m *MetricsContext) AddMetricWithDimensions(name string, value float64, dimensions map[string]string) {
	m.Metrics = append(m.Metrics, Metric{
		Name:       name,
		Value:      value,
		Dimensions: dimensions,
	})
}
