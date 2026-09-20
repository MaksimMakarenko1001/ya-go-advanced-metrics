package v0

import (
	"net"
	"testing"

	"github.com/MaksimMakarenko1001/ya-go-advanced/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ValidateEnabledService(t *testing.T) {
	tests := []struct {
		name      string
		ipToCheck net.IP
		wantErr   bool
	}{
		{
			name:      "ip within subnet",
			ipToCheck: net.ParseIP("192.168.1.100"),
			wantErr:   false,
		},
		{
			name:      "ip at subnet boundary (first)",
			ipToCheck: net.ParseIP("192.168.1.0"),
			wantErr:   false,
		},
		{
			name:      "ip at subnet boundary (last)",
			ipToCheck: net.ParseIP("192.168.1.255"),
			wantErr:   false,
		},
		{
			name:      "ip outside subnet",
			ipToCheck: net.ParseIP("10.0.0.1"),
			wantErr:   true,
		},
		{
			name:      "ip just outside subnet",
			ipToCheck: net.ParseIP("192.168.2.1"),
			wantErr:   true,
		},
		{
			name:      "nil ip",
			ipToCheck: nil,
			wantErr:   true,
		},
	}

	service := New(Config{
		ValidateEnabled: true,
		TrustedSubnet:   "192.168.1.0/24",
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Validate(t.Context(), tt.ipToCheck)

			if tt.wantErr {
				require.ErrorIs(t, err, pkg.ErrForbidden)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestService_ValidateDisabledService(t *testing.T) {
	tests := []struct {
		name      string
		ipToCheck net.IP
	}{
		{
			name:      "any ip when validation disabled",
			ipToCheck: net.ParseIP("192.168.1.100"),
		},
		{
			name:      "localhost when validation disabled",
			ipToCheck: net.ParseIP("127.0.0.1"),
		},
		{
			name:      "ipv6 when validation disabled",
			ipToCheck: net.ParseIP("::1"),
		},
	}

	disabledService := New(Config{
		ValidateEnabled: false,
		TrustedSubnet:   "192.168.1.0/24", // This should be ignored
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := disabledService.Validate(t.Context(), tt.ipToCheck)

			assert.NoError(t, err)
		})
	}
}

func TestService_ValidateInvalidSubnet(t *testing.T) {
	tests := []struct {
		name       string
		subnetCIDR string
		ipToCheck  net.IP
	}{
		{
			name:       "invalid cidr format",
			subnetCIDR: "invalid-cidr",
			ipToCheck:  net.ParseIP("192.168.1.100"),
		},
		{
			name:       "empty subnet",
			subnetCIDR: "",
			ipToCheck:  net.ParseIP("192.168.1.100"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New(Config{
				ValidateEnabled: true,
				TrustedSubnet:   tt.subnetCIDR,
			})

			err := service.Validate(t.Context(), tt.ipToCheck)
			require.ErrorIs(t, err, errTrustedSubnet)
		})
	}
}
