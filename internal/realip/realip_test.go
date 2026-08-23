package realip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCIDR(t *testing.T) {
	t.Parallel()

	network, err := ParseCIDR("")
	require.NoError(t, err)
	assert.Nil(t, network)

	network, err = ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	require.NotNil(t, network)
	assert.True(t, network.Contains(net.ParseIP("192.168.1.10")))
	assert.False(t, network.Contains(net.ParseIP("10.0.0.1")))

	_, err = ParseCIDR("not-a-cidr")
	require.Error(t, err)
}

func TestAllowed(t *testing.T) {
	t.Parallel()

	assert.True(t, Allowed(nil, ""))
	assert.True(t, Allowed(nil, "8.8.8.8"))

	_, network, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)

	assert.True(t, Allowed(network, "10.1.2.3"))
	assert.False(t, Allowed(network, "192.168.0.1"))
	assert.False(t, Allowed(network, ""))
	assert.False(t, Allowed(network, "not-an-ip"))
}

func TestHostReturnsIPv4(t *testing.T) {
	t.Parallel()

	ip := net.ParseIP(Host())
	require.NotNil(t, ip)
	assert.NotNil(t, ip.To4())
}
