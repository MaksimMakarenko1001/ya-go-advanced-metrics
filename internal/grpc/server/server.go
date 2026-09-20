package server

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Server struct {
	config  Config
	handler pb.MetricsServer
	server  *grpc.Server
}

func New(
	cfg Config,
	handler pb.MetricsServer,
	interceptors ...grpc.UnaryServerInterceptor,
) *Server {
	opts := make([]grpc.ServerOption, 0, 2)

	opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
	}))

	opts = append(opts, grpc.ChainUnaryInterceptor(interceptors...))

	return &Server{
		config:  cfg,
		handler: handler,
		server:  grpc.NewServer(opts...),
	}
}

func (s *Server) Start(errCh chan<- error) {
	listen, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		errCh <- fmt.Errorf("listen error: %v", err)
		return
	}

	pb.RegisterMetricsServer(s.server, s.handler)

	log.Println("gRPC server is running on" + s.config.Address)

	go func() {
		if err := s.server.Serve(listen); err != nil {
			log.Println("grpc error:", err.Error())
			errCh <- err
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) {
	log.Println("gRPC server is trying to stop")
	stopCh := make(chan struct{})

	go func() {
		defer close(stopCh)

		if s.server != nil {
			s.server.GracefulStop()
		}
	}()

	select {
	case <-stopCh:
	case <-ctx.Done():
		if s.server != nil {
			s.server.Stop()
		}
	}
}
