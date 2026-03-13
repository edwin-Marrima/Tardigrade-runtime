package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		expectedIP      net.IP
		expectedMask    net.IPMask
		expectedGateway net.IP
		wantErr     bool
	}{
		{
			name:            "valid IPv4 /24",
			cidr:            "192.168.1.5/24",
			expectedIP:      net.IP{192, 168, 1, 5},
			expectedMask:    net.CIDRMask(24, 32),
			expectedGateway: net.IP{192, 168, 1, 1},
		},
		{
			name:            "valid IPv4 /16",
			cidr:            "10.0.5.20/16",
			expectedIP:      net.IP{10, 0, 5, 20},
			expectedMask:    net.CIDRMask(16, 32),
			expectedGateway: net.IP{10, 0, 0, 1},
		},
		{
			name:            "valid IPv4 /30 — smallest routable subnet",
			cidr:            "172.16.0.2/30",
			expectedIP:      net.IP{172, 16, 0, 2},
			expectedMask:    net.CIDRMask(30, 32),
			expectedGateway: net.IP{172, 16, 0, 1},
		},
		{
			name:            "host is the first usable address",
			cidr:            "10.0.0.1/24",
			expectedIP:      net.IP{10, 0, 0, 1},
			expectedMask:    net.CIDRMask(24, 32),
			expectedGateway: net.IP{10, 0, 0, 1},
		},
		{
			name:    "invalid CIDR — missing prefix length",
			cidr:    "192.168.1.1",
			wantErr: true,
		},
		{
			name:    "invalid CIDR — bad IP",
			cidr:    "999.168.1.1/24",
			wantErr: true,
		},
		{
			name:    "empty string",
			cidr:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseCIDR(tt.cidr)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedIP, info.IP)
			assert.Equal(t, tt.expectedMask, info.Mask)
			assert.Equal(t, tt.expectedGateway, info.Gateway)
		})
	}
}
