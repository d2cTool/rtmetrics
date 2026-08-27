package grpctls

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestGenerateRoundTrip(t *testing.T) {
	t.Parallel()

	_, certPEM, keyPEM, err := Generate([]string{"127.0.0.1"})
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "grpc.crt")
	keyFile := filepath.Join(dir, "grpc.key")
	require.NoError(t, Write(certFile, keyFile, certPEM, keyPEM))

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srvCreds, err := ServerCredentials(certFile, keyFile)
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.Creds(srvCreds))
	healthpb.RegisterHealthServer(srv, health.NewServer())
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	cliCreds, err := ClientCredentials(certFile)
	require.NoError(t, err)
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(cliCreds))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = healthpb.NewHealthClient(conn).Check(t.Context(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
}

func TestServerCredentialsEphemeral(t *testing.T) {
	t.Parallel()

	creds, err := ServerCredentials("", "")
	require.NoError(t, err)
	require.Equal(t, "tls", creds.Info().SecurityProtocol)
}

func TestServerCredentialsRequiresBothFiles(t *testing.T) {
	t.Parallel()

	_, err := ServerCredentials("only.crt", "")
	require.Error(t, err)
}

func TestClientCredentialsWithoutCAUsesTLS(t *testing.T) {
	t.Parallel()

	creds, err := ClientCredentials("")
	require.NoError(t, err)
	require.Equal(t, "tls", creds.Info().SecurityProtocol)
}
