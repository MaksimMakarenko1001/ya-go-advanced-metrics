package remote

import (
	"context"
	"time"

	pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/models"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/service"
	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg"
	"google.golang.org/grpc/metadata"
)

var _ service.UpdateBatchRemoteService = (*Service)(nil)

type Service struct {
	updateBatchService service.UpdateBatchService
}

func New(
	updateBatchService service.UpdateBatchService,
) *Service {
	return &Service{
		updateBatchService: updateBatchService,
	}
}

func (srv *Service) Call(ctx context.Context, req *pb.UpdateMetricsRequest) (resp *pb.UpdateMetricsResponse, err error) {
	request := models.Request{
		Metrics: make([]models.Metric, 0, len(req.GetMetrics())),
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("x-real-ip")
		if len(values) > 0 {
			request.IPAddress = values[0]
		}
	}

	for _, m := range req.GetMetrics() {
		request.Metrics = append(request.Metrics, models.Metric{
			ID:    m.GetId(),
			MType: pkg.ProtoMetricTypeMap[m.GetType()],
			Delta: pkg.ToPtr(m.GetDelta()),
			Value: pkg.ToPtr(m.GetValue()),
		})
	}

	if err := srv.updateBatchService.Do(ctx, time.Now(), request); err != nil {
		return nil, err
	}

	return &pb.UpdateMetricsResponse{}, nil
}
