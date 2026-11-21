// Package ingress serves Shrike's HTTP handlers on a shared listener.
package ingress

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type Config struct {
	Address      string
	LegacyDomain string
	TLSCertFile  string
	TLSKeyFile   string
}

func NewServer(cfg Config, surfaces map[string]http.Handler, log zerolog.Logger) (*http.Server, error) {
	if _, _, err := net.SplitHostPort(cfg.Address); err != nil {
		return nil, fmt.Errorf("invalid HTTP listener address: %w", err)
	}
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return nil, errors.New("configure both --tls-cert and --tls-key to enable HTTPS")
	}
	handler, err := newHandler(cfg.LegacyDomain, surfaces)
	if err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           accessLog(log, handler),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		IdleTimeout:       5 * time.Minute,
		MaxHeaderBytes:    64 << 10,
		WriteTimeout:      0, // The feed and reload streams manage their own lifetimes.
	}
	if cfg.TLSCertFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load HTTP TLS certificate: %w", err)
		}
		server.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
	}
	return server, nil
}

// Serve runs the bound listener and drains ordinary requests on cancellation.
func Serve(ctx context.Context, server *http.Server, listener net.Listener, log zerolog.Logger) error {
	defer listener.Close() //nolint:errcheck // owns the listener even if TLS setup fails
	defer server.Close()   //nolint:errcheck // serve and shutdown errors are returned separately
	done := make(chan error, 1)
	go func() {
		if server.TLSConfig != nil {
			done <- server.ServeTLS(listener, "", "")
			return
		}
		done <- server.Serve(listener)
	}()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		drain, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(drain); err != nil {
			log.Warn().Err(err).Msg("HTTP shutdown deadline reached, closing remaining connections")
			return server.Close()
		}
		return nil
	}
}
