package api

import (
	"context"
	"net"

	pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"
	subnetService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/subnetService/service"
	updateBatchService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/updateBatchService/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type API struct {
	pb.UnimplementedMetricsServer

	updateBatchService updateBatchService.UpdateBatchRemoteService
	subnetService      subnetService.SubnetService
}

func New(
	updateBatchService updateBatchService.UpdateBatchRemoteService,
	subnetService subnetService.SubnetService,

) *API {
	return &API{
		updateBatchService: updateBatchService,
		subnetService:      subnetService,
	}
}

func (api *API) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	return api.updateBatchService.Call(ctx, req)
}

func (api *API) WithTrustedSubnet() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		var realIP string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get("x-real-ip")
			if len(values) > 0 {
				realIP = values[0]
			}
		}

		if err := api.subnetService.Validate(ctx, net.ParseIP(realIP)); err != nil {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip")
		}

		return handler(ctx, req)
	}
}
