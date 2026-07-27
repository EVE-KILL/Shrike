// Package unixhttp runs private HTTP surfaces over Unix domain sockets.
package unixhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Server is an HTTP server bound to one filesystem Unix socket.
type Server struct {
	http   *http.Server
	socket string
	done   chan struct{}

	mu  sync.RWMutex
	err error
}

// Listen binds socket and starts serving handler. An existing socket is
// treated as stale and replaced; an ordinary file is never removed.
func Listen(socket string, handler http.Handler) (*Server, error) {
	if err := validateSocketPath(socket); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("Unix HTTP handler is required")
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		return nil, fmt.Errorf("create Unix socket directory: %w", err)
	}
	if err := removeStaleSocket(socket); err != nil {
		return nil, err
	}

	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("listen on Unix socket %s: %w", socket, err)
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(socket)
		return nil, fmt.Errorf("secure Unix socket %s: %w", socket, err)
	}

	server := &Server{
		http: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
		socket: socket,
		done:   make(chan struct{}),
	}
	go func() {
		err := server.http.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		server.mu.Lock()
		server.err = err
		server.mu.Unlock()
		close(server.done)
	}()
	return server, nil
}

// Socket returns the absolute socket path used by the server.
func (s *Server) Socket() string {
	return s.socket
}

// Done closes if the private HTTP server stops.
func (s *Server) Done() <-chan struct{} {
	return s.done
}

// Err reports why the server stopped. It is intended to be read after Done.
func (s *Server) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// Close gracefully drains the server and removes its socket.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownErr := s.http.Shutdown(ctx)
	if errors.Is(shutdownErr, http.ErrServerClosed) {
		shutdownErr = nil
	}
	if removeErr := os.Remove(s.socket); removeErr != nil && !os.IsNotExist(removeErr) {
		if shutdownErr != nil {
			return errors.Join(shutdownErr, removeErr)
		}
		return fmt.Errorf("remove Unix socket %s: %w", s.socket, removeErr)
	}
	return shutdownErr
}

func validateSocketPath(socket string) error {
	if socket == "" {
		return errors.New("Unix socket path is required")
	}
	if !filepath.IsAbs(socket) {
		return fmt.Errorf("Unix socket path must be absolute: %s", socket)
	}
	if filepath.Clean(socket) == string(filepath.Separator) {
		return errors.New("Unix socket path cannot be the filesystem root")
	}
	return nil
}

func removeStaleSocket(socket string) error {
	info, err := os.Lstat(socket)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect Unix socket %s: %w", socket, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to replace non-socket path %s", socket)
	}
	if err := os.Remove(socket); err != nil {
		return fmt.Errorf("remove stale Unix socket %s: %w", socket, err)
	}
	return nil
}
