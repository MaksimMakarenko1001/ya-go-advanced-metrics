package v0

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/subnetService/service"
	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg"
)

var _ service.SubnetService = (*Service)(nil)

var (
	errNotFound      = errors.New("not found")
	errTrustedSubnet = errors.New("trusted subnet error")
)

type Service struct {
	cfg       Config
	netIP     *net.IPNet
	netIPFunc func(string) (*net.IPNet, error)
}

func New(config Config) *Service {
	s := &Service{cfg: config}

	s.netIPFunc = func(trustedSubnet string) (*net.IPNet, error) {
		_, ipNet, err := net.ParseCIDR(config.TrustedSubnet)
		if err != nil {
			return nil, fmt.Errorf("parse cidr error: %w", err)
		}
		if ipNet == nil {
			return nil, errNotFound
		}
		return ipNet, nil
	}

	return s
}

func (s *Service) Validate(ctx context.Context, ip net.IP) error {
	if !s.cfg.ValidateEnabled {
		return nil
	}
	if s.netIP == nil {
		netIP, err := s.netIPFunc(s.cfg.TrustedSubnet)
		if err != nil {
			return fmt.Errorf("%w:%s", errTrustedSubnet, err.Error())
		}

		s.netIP = netIP
	}
	if ok := s.netIP.Contains(ip); !ok {
		return pkg.ErrForbidden
	}

	return nil
}
