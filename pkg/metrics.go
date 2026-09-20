package pkg

import pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"

// MetricType represents the type of a metric.
type MetricType = string

const (
	// MetricTypeGauge represents a gauge metric type.
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeCounter represents a counter metric type.
	MetricTypeCounter MetricType = "counter"
)

var ProtoMetricTypeMap = map[pb.Metric_MType]MetricType{
	pb.Metric_GAUGE:   MetricTypeGauge,
	pb.Metric_COUNTER: MetricTypeCounter,
}
