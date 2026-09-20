package service

import (
	"context"
	"net"
)

type SubnetService interface {
	Validate(ctx context.Context, ip net.IP) error
}
