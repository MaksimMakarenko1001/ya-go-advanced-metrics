package client

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type (
	DialerFunc func(context.Context, string) (net.Conn, error)
	ConnFunc   func([]grpc.DialOption, ...grpc.UnaryClientInterceptor) (*grpc.ClientConn, error)
)

type Client struct {
	opts         []grpc.DialOption
	interceptors []grpc.UnaryClientInterceptor
	connFunc     ConnFunc
}

func New(
	cfg Config,
	interceptors ...grpc.UnaryClientInterceptor,
) *Client {
	cli := &Client{
		opts:         make([]grpc.DialOption, 0, len(interceptors)+3),
		interceptors: interceptors,
		connFunc:     connector(cfg.Address),
	}

	cli.opts = append(cli.opts,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(cfg.MsgSizeMaxMB*1024*1024)),
		grpc.WithContextDialer(dialer(cfg.Address, cfg.Timeout)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	return cli
}

func (c *Client) WithRealIP(ip string) *Client {
	interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", ip)

		return invoker(ctx, method, req, reply, cc, opts...)
	}

	c.interceptors = append(c.interceptors, interceptor)

	return c
}

func (c *Client) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest, opts ...grpc.CallOption) (*pb.UpdateMetricsResponse, error) {
	conn, err := c.connFunc(c.opts, c.interceptors...)
	if err != nil {
		return nil, fmt.Errorf("update metrics error: %w", err)
	}

	return pb.NewMetricsClient(conn).UpdateMetrics(ctx, in, opts...)
}

func dialer(address string, timeout time.Duration) DialerFunc {
	return func(ctx context.Context, s string) (net.Conn, error) {
		dialCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var dialer net.Dialer

		return dialer.DialContext(dialCtx, "tcp", address)
	}
}

func connector(address string) ConnFunc {
	return func(opts []grpc.DialOption, interceptors ...grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
		opts = append(opts, grpc.WithChainUnaryInterceptor(interceptors...))

		conn, err := grpc.NewClient(address, opts...)
		if err != nil {
			return nil, fmt.Errorf("connect error: %w", err)
		}

		return conn, nil
	}
}
