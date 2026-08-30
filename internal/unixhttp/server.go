// Package unixhttp runs private HTTP surfaces — listeners that carry
// server-side rendering traffic and are never reachable from outside the host.
//
// Production uses a Unix domain socket, which is private by construction: it
// has no port and no interface. Development uses a loopback port instead,
// because `nuxt dev` runs under Node, whose fetch has no way to dial a socket.
package unixhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxHeaderValueCount = 64

// Server is an HTTP server on one private listener.
type Server struct {
	http *http.Server
	// socket is the filesystem path to unlink on Close. Empty for a loopback
	// listener, which leaves nothing behind to clean up.
	socket string
	addr   string
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

	return serve(listener, socket, socket, handler), nil
}

// ListenLoopback binds a loopback TCP address and starts serving handler.
//
// This is the development counterpart to Listen. It refuses any address that
// is not loopback: this listener carries the same authenticated surface as the
// public one but without the edge in front of it, so binding it to a routable
// interface would publish an unguarded copy of the whole site.
func ListenLoopback(address string, handler http.Handler) (*Server, error) {
	if handler == nil {
		return nil, errors.New("private HTTP handler is required")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("private HTTP address must be host:port: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf(
			"private HTTP address must be a loopback IP, got %q", host,
		)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}
	return serve(listener, "", listener.Addr().String(), handler), nil
}

func serve(
	listener net.Listener,
	socket string,
	addr string,
	handler http.Handler,
) *Server {
	server := &Server{
		http: &http.Server{
			Handler:             privateDiagnostics(handler),
			ReadHeaderTimeout:   10 * time.Second,
			MaxHeaderValueCount: maxHeaderValueCount,
		},
		socket: socket,
		addr:   addr,
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
	return server
}

// privateDiagnostics adds the Go 1.27 goroutine leak profile to the private
// listener. It deliberately exposes no other pprof surface.
func privateDiagnostics(next http.Handler) http.Handler {
	profile := pprof.Handler("goroutineleak")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/debug/pprof/goroutineleak" {
			profile.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Socket returns the absolute socket path, or the empty string for a loopback
// listener.
func (s *Server) Socket() string {
	return s.socket
}

// Addr returns the address the server is reachable at: a socket path, or a
// resolved host:port.
func (s *Server) Addr() string {
	return s.addr
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
	if s.socket == "" {
		return shutdownErr
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
