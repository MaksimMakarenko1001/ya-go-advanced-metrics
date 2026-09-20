package v0

import (
	"context"
	"time"

	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/entities"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/models"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/service"
	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg"
)

var _ service.UpdateBatchService = (*Service)(nil)

var (
	errInvalidMetricValue *pkg.Error = pkg.ErrBadRequest.SetInfo("invalid metric value")
	errInvalidMetricType  *pkg.Error = pkg.ErrBadRequest.SetInfo("invalid metric type")
)

type Service struct {
	metricRepository MetricRepository
}

func New(
	metricRepository MetricRepository,
) *Service {
	return &Service{
		metricRepository: metricRepository,
	}
}

func (srv *Service) Do(ctx context.Context, ts time.Time, request models.Request) (err error) {
	if len(request.Metrics) == 0 {
		return nil
	}

	counters := make(map[string]entities.CounterItem, len(request.Metrics))
	gauges := make(map[string]entities.GaugeItem, len(request.Metrics))
	metrics := make([]string, 0, len(request.Metrics))

	for _, metric := range request.Metrics {
		switch metric.MType {
		case pkg.MetricTypeCounter:
			if metric.Delta == nil {
				return errInvalidMetricValue
			}

			delta := *metric.Delta + counters[metric.ID].MetricValue
			counters[metric.ID] = entities.CounterItem{
				MetricType:  metric.MType,
				MetricName:  metric.ID,
				MetricValue: delta,
				CreatedAt:   ts,
				UpdatedAt:   ts,
			}

		case pkg.MetricTypeGauge:
			if metric.Value == nil {
				return errInvalidMetricValue
			}
			gauges[metric.ID] = entities.GaugeItem{
				MetricType:  metric.MType,
				MetricName:  metric.ID,
				MetricValue: *metric.Value,
				CreatedAt:   ts,
				UpdatedAt:   ts,
			}

		default:
			return errInvalidMetricType
		}

		metrics = append(metrics, metric.ID)
	}

	outboxes := []entities.Outbox{
		{
			Destination: string(models.FileOutboxDestination),
			Payload: pkg.MustJSON(models.FileEvent{
				TS:        ts,
				Metrics:   metrics,
				IPAddress: request.IPAddress,
			}),
		},
		{
			Destination: string(models.RemoteOutboxDestination),
			Payload: pkg.MustJSON(models.RemoteEvent{
				TS:        ts,
				Metrics:   metrics,
				IPAddress: request.IPAddress,
			}),
		},
	}

	ok, err := srv.metricRepository.AddUpdateBatch(ctx, pkg.ValuesToList(counters), pkg.ValuesToList(gauges), outboxes, "")
	if err != nil {
		return pkg.ErrInternalServer.SetInfo(err.Error())
	}
	if !ok {
		return pkg.ErrBadRequest
	}

	return nil
}
