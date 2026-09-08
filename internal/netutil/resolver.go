package netutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	setupOnce   sync.Once
	fallbackDNS = []string{
		"8.8.8.8:53",
		"1.1.1.1:53",
		"8.8.4.4:53",
		"1.0.0.1:53",
	}
)

// SetupNetworkEnvironment initializes DNS resolvers and TLS certificates for Android/embedded environments.
func SetupNetworkEnvironment() {
	setupOnce.Do(func() {
		setupDNSResolver()
		setupRootCerts()
	})
}

// IsAndroid returns true if running on an Android device or OS.
func IsAndroid() bool {
	if runtime.GOOS == "android" {
		return true
	}
	if os.Getenv("ANDROID_ROOT") != "" || os.Getenv("ANDROID_DATA") != "" {
		return true
	}
	if _, err := os.Stat("/system/bin/getprop"); err == nil {
		return true
	}
	return false
}

func setupDNSResolver() {
	systemDNS := getAndroidSystemDNS()
	var allServers []string
	allServers = append(allServers, systemDNS...)
	allServers = append(allServers, fallbackDNS...)

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 4 * time.Second,
			}

			// If address is loopback port 53 (127.0.0.1:53 or [::1]:53), Android lacks /etc/resolv.conf
			isLoopbackDNS := strings.HasPrefix(address, "127.0.0.1:53") ||
				strings.HasPrefix(address, "[::1]:53") ||
				strings.HasPrefix(address, "localhost:53")

			if isLoopbackDNS || IsAndroid() {
				// Try discovered or fallback DNS servers in priority order
				var lastErr error
				for _, dnsServer := range allServers {
					dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
					conn, err := d.DialContext(dialCtx, "udp", dnsServer)
					cancel()
					if err == nil {
						return conn, nil
					}
					lastErr = err
				}
				if lastErr != nil {
					return nil, lastErr
				}
			}

			// Try default address first
			conn, err := d.DialContext(ctx, network, address)
			if err == nil {
				return conn, nil
			}

			// If default fails, fallback to public DNS
			for _, dnsServer := range allServers {
				dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				conn, fallbackErr := d.DialContext(dialCtx, "udp", dnsServer)
				cancel()
				if fallbackErr == nil {
					return conn, nil
				}
			}

			return nil, err
		},
	}
	slog.Info("Network DNS resolver initialized", "android", IsAndroid(), "servers", len(allServers))
}

func getAndroidSystemDNS() []string {
	if !IsAndroid() {
		return nil
	}

	var servers []string
	// Check Android system properties
	props := []string{"net.dns1", "net.dns2", "net.dns3", "net.dns4"}
	for _, prop := range props {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		cmd := exec.CommandContext(ctx, "/system/bin/getprop", prop)
		out, err := cmd.Output()
		cancel()
		if err == nil {
			ip := strings.TrimSpace(string(out))
			if ip != "" && net.ParseIP(ip) != nil {
				servers = append(servers, ip+":53")
			}
		}
	}
	return servers
}

func setupRootCerts() {
	if os.Getenv("SSL_CERT_DIR") == "" {
		_ = os.Setenv("SSL_CERT_DIR", "/system/etc/security/cacerts:/apex/com.android.conscrypt/cacerts")
	}

	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}

	// Scan Android CA directories
	caDirs := []string{
		"/system/etc/security/cacerts",
		"/apex/com.android.conscrypt/cacerts",
		"/data/misc/user/0/cacerts-added",
	}

	loaded := 0
	for _, dir := range caDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err == nil && len(data) > 0 {
				if pool.AppendCertsFromPEM(data) {
					loaded++
				}
			}
		}
	}

	if loaded > 0 {
		tlsConfig := &tls.Config{
			RootCAs: pool,
		}
		if http.DefaultTransport != nil {
			if transport, ok := http.DefaultTransport.(*http.Transport); ok {
				transport.TLSClientConfig = tlsConfig
			}
		}
		slog.Info("Loaded Android system CA certificates into TLS pool", "certs", loaded)
	}
}
