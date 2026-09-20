package service

import (
	"context"
	"time"

	pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/models"
)

type UpdateBatchService interface {
	Do(ctx context.Context, ts time.Time, request models.Request) (err error)
}

type UpdateBatchRemoteService interface {
	Call(ctx context.Context, req *pb.UpdateMetricsRequest) (resp *pb.UpdateMetricsResponse, err error)
}
