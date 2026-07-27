// Package renderer supervises the Bun process that serves the Nuxt frontend.
package renderer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultStartupTimeout = 30 * time.Second

// Options configures the supervised Nuxt renderer.
type Options struct {
	Entrypoint string
	Socket     string
	APISocket  string
	Stdout     io.Writer
	Stderr     io.Writer
}

// Process is a running Bun renderer.
type Process struct {
	command *exec.Cmd
	socket  string
	done    chan struct{}

	mu  sync.RWMutex
	err error
}

// ResolveEntrypoint resolves an explicit entrypoint, or finds the nearest
// web/.output/server/index.mjs while walking upward from cwd. The second return
// value is false outside a source checkout when no renderer was configured.
func ResolveEntrypoint(configured string) (string, bool, error) {
	if strings.TrimSpace(configured) != "" {
		entrypoint, err := filepath.Abs(configured)
		if err != nil {
			return "", false, fmt.Errorf("resolve Nuxt entrypoint: %w", err)
		}
		if err := requireRegularFile(entrypoint); err != nil {
			return "", false, err
		}
		return entrypoint, true, nil
	}

	start, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("find current directory: %w", err)
	}
	foundWebSource := false
	for dir := start; ; dir = filepath.Dir(dir) {
		webDir := filepath.Join(dir, "web")
		if info, statErr := os.Stat(filepath.Join(webDir, "package.json")); statErr == nil && !info.IsDir() {
			foundWebSource = true
		}
		candidate := filepath.Join(webDir, ".output", "server", "index.mjs")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, true, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	if foundWebSource {
		return "", false, errors.New(
			"Nuxt renderer is not built; run `cd web && bun install --frozen-lockfile && bun run build`",
		)
	}
	return "", false, nil
}

// Start launches Bun, waits for the renderer to accept connections on its
// socket, and then returns ownership of the process to the caller.
func Start(options Options) (*Process, error) {
	if err := validateSocketPath(options.Socket); err != nil {
		return nil, fmt.Errorf("Nuxt socket: %w", err)
	}
	if err := validateSocketPath(options.APISocket); err != nil {
		return nil, fmt.Errorf("Shrike API socket: %w", err)
	}
	if err := requireRegularFile(options.Entrypoint); err != nil {
		return nil, err
	}
	if err := removeStaleSocket(options.Socket); err != nil {
		return nil, err
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		return nil, errors.New("Bun is required to run the Nuxt renderer: executable not found in PATH")
	}

	command := exec.Command(bun, options.Entrypoint)
	command.Dir = filepath.Dir(options.Entrypoint)
	command.Stdout = options.Stdout
	command.Stderr = options.Stderr
	if command.Stdout == nil {
		command.Stdout = os.Stdout
	}
	if command.Stderr == nil {
		command.Stderr = os.Stderr
	}
	command.Env = replaceEnv(os.Environ(), map[string]string{
		"NITRO_UNIX_SOCKET": options.Socket,
		"NUXT_API_SOCKET":   options.APISocket,
		"NODE_ENV":          "production",
	})

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start Bun renderer: %w", err)
	}
	process := &Process{
		command: command,
		socket:  options.Socket,
		done:    make(chan struct{}),
	}
	go func() {
		err := command.Wait()
		process.mu.Lock()
		process.err = err
		process.mu.Unlock()
		close(process.done)
	}()

	if err := process.waitReady(defaultStartupTimeout); err != nil {
		_ = process.Close()
		return nil, err
	}
	return process, nil
}

// Socket returns the renderer socket path.
func (p *Process) Socket() string {
	return p.socket
}

// Done closes when Bun exits.
func (p *Process) Done() <-chan struct{} {
	return p.done
}

// Err reports the process exit status. It is intended to be read after Done.
func (p *Process) Err() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.err
}

// Close asks Bun to shut down gracefully, waits, and kills it only if it does
// not exit within ten seconds. The exact renderer socket is then removed.
func (p *Process) Close() error {
	select {
	case <-p.done:
	default:
		if err := p.command.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("signal Bun renderer: %w", err)
		}
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-p.done:
			timer.Stop()
		case <-timer.C:
			if err := p.command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				return fmt.Errorf("kill Bun renderer: %w", err)
			}
			<-p.done
		}
	}
	if err := os.Remove(p.socket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove Nuxt socket %s: %w", p.socket, err)
	}
	return nil
}

func (p *Process) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-p.done:
			if err := p.Err(); err != nil {
				return fmt.Errorf("Bun renderer exited before binding %s: %w", p.socket, err)
			}
			return fmt.Errorf("Bun renderer exited before binding %s", p.socket)
		default:
		}

		connection, err := net.DialTimeout("unix", p.socket, 100*time.Millisecond)
		if err == nil {
			return connection.Close()
		}
		lastErr = err
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf(
		"Bun renderer did not accept connections on %s within %s (last dial: %v)",
		p.socket,
		timeout,
		lastErr,
	)
}

func requireRegularFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("find Nuxt renderer entrypoint %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Nuxt renderer entrypoint is not a regular file: %s", path)
	}
	return nil
}

func validateSocketPath(socket string) error {
	if socket == "" {
		return errors.New("path is required")
	}
	if !filepath.IsAbs(socket) {
		return fmt.Errorf("path must be absolute: %s", socket)
	}
	if filepath.Clean(socket) == string(filepath.Separator) {
		return errors.New("path cannot be the filesystem root")
	}
	return nil
}

func removeStaleSocket(socket string) error {
	info, err := os.Lstat(socket)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect Nuxt socket %s: %w", socket, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to replace non-socket path %s", socket)
	}
	if err := os.Remove(socket); err != nil {
		return fmt.Errorf("remove stale Nuxt socket %s: %w", socket, err)
	}
	return nil
}

func replaceEnv(current []string, replacements map[string]string) []string {
	result := make([]string, 0, len(current)+len(replacements))
	for _, item := range current {
		key, _, found := strings.Cut(item, "=")
		if _, replace := replacements[key]; found && replace {
			continue
		}
		result = append(result, item)
	}
	for key, value := range replacements {
		result = append(result, key+"="+value)
	}
	return result
}
