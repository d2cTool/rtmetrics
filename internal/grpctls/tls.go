// Package grpctls собирает TLS-credentials для gRPC на самоподписанных или файловых сертификатах.
package grpctls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"google.golang.org/grpc/credentials"
)

// ServerCredentials загружает сертификат и ключ или выпускает эфемерный self-signed.
func ServerCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	if certFile == "" && keyFile == "" {
		cert, _, _, err := Generate(nil)
		if err != nil {
			return nil, err
		}
		return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}), nil
	}
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("grpc tls: both cert and key files are required")
	}
	return credentials.NewServerTLSFromFile(certFile, keyFile)
}

// ClientCredentials доверяет CA из caFile. Пустой caFile — TLS без проверки имени сервера.
func ClientCredentials(caFile string) (credentials.TransportCredentials, error) {
	if caFile == "" {
		return credentials.NewTLS(&tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}), nil
	}
	return credentials.NewClientTLSFromFile(caFile, "")
}

// Generate выпускает self-signed сертификат. hosts пустой — localhost, 127.0.0.1, ::1.
func Generate(hosts []string) (tls.Certificate, []byte, []byte, error) {
	if len(hosts) == 0 {
		hosts = []string{"localhost", "127.0.0.1", "::1"}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, nil, fmt.Errorf("generate grpc tls key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, nil, nil, fmt.Errorf("generate grpc tls serial: %w", err)
	}

	dns, ips := splitHosts(hosts)
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"rtmetrics"},
			CommonName:   "rtmetrics-grpc",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dns,
		IPAddresses:           ips,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, nil, fmt.Errorf("create grpc tls cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return tls.Certificate{}, nil, nil, fmt.Errorf("marshal grpc tls key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, nil, nil, fmt.Errorf("load grpc tls pair: %w", err)
	}
	return cert, certPEM, keyPEM, nil
}

// Write пишет PEM сертификата и ключа на диск.
func Write(certFile, keyFile string, certPEM, keyPEM []byte) error {
	if err := os.WriteFile(certFile, certPEM, 0o644); err != nil {
		return fmt.Errorf("write grpc cert: %w", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write grpc key: %w", err)
	}
	return nil
}

func splitHosts(hosts []string) (dns []string, ips []net.IP) {
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			ips = append(ips, ip)
			continue
		}
		dns = append(dns, h)
	}
	return dns, ips
}
