package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	DefaultMaxInputBytes   int64 = 32 << 20
	DefaultMaxOutputBytes  int64 = 128 << 20
	DefaultMaxCells              = 2_000_000
	DefaultMaxSheets             = 100
	DefaultMaxPreviewBytes       = 256 << 10
)

// Options は MCP サーバの設定．
type Options struct {
	Version         string
	AllowDirs       []string
	CreateAllowDirs bool
	MaxInputBytes   int64
	MaxOutputBytes  int64
	MaxCells        int
	MaxSheets       int
	MaxPreviewBytes int
	Logger          *slog.Logger
}

type server struct {
	opts  Options
	roots []string
	sem   chan struct{}
}

// Run は stdio で MCP サーバを起動する．stdout はプロトコル専用になる．
func Run(ctx context.Context, opts Options) error {
	s, err := newServer(opts)
	if err != nil {
		return err
	}
	srv := s.newMCPServer()
	if s.opts.Logger != nil {
		s.opts.Logger.Info("json2xlsx MCP server starting",
			"version", s.opts.Version,
			"roots", s.roots,
		)
	}
	return srv.Run(ctx, &mcp.StdioTransport{MaxLineLength: 64 << 20})
}

// NewServer はテストや埋め込み向けに MCP サーバを構築する．
func NewServer(opts Options) (*mcp.Server, error) {
	s, err := newServer(opts)
	if err != nil {
		return nil, err
	}
	return s.newMCPServer(), nil
}

func newServer(opts Options) (*server, error) {
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.MaxInputBytes <= 0 {
		opts.MaxInputBytes = DefaultMaxInputBytes
	}
	if opts.MaxOutputBytes <= 0 {
		opts.MaxOutputBytes = DefaultMaxOutputBytes
	}
	if opts.MaxCells <= 0 {
		opts.MaxCells = DefaultMaxCells
	}
	if opts.MaxSheets <= 0 {
		opts.MaxSheets = DefaultMaxSheets
	}
	if opts.MaxPreviewBytes <= 0 {
		opts.MaxPreviewBytes = DefaultMaxPreviewBytes
	}
	roots, err := resolveRoots(opts.AllowDirs, opts.CreateAllowDirs)
	if err != nil {
		return nil, err
	}
	return &server{
		opts:  opts,
		roots: roots,
		sem:   make(chan struct{}, 2),
	}, nil
}

func (s *server) newMCPServer() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "json2xlsx",
		Version: s.opts.Version,
	}, &mcp.ServerOptions{
		Logger: s.opts.Logger,
		Instructions: `json2xlsx converts SheetJS-style JSON or high-level table JSON into XLSX and back.
For simple tables, prefer the "table" input with header/rows/columns. Use json2xlsx_validate before json2xlsx_convert, and json2xlsx_preview to inspect the rendered result.
The json2xlsx://schema, json2xlsx://schema/table, json2xlsx://skill/table, and json2xlsx://samples/* resources contain the full formats and examples.`,
	})
	s.addTools(srv)
	s.addResources(srv)
	return srv
}

// acquire は同時変換数を制限する．
func (s *server) acquire(ctx context.Context) error {
	select {
	case s.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *server) release() {
	<-s.sem
}

func resolveRoots(dirs []string, create bool) ([]string, error) {
	if len(dirs) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		dirs = []string{cwd}
	}
	out := make([]string, 0, len(dirs))
	seen := map[string]bool{}
	for _, d := range dirs {
		if strings.TrimSpace(d) == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			return nil, fmt.Errorf("resolve allow-dir %q: %w", d, err)
		}
		if create {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return nil, fmt.Errorf("create allow-dir %q: %w", d, err)
			}
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, fmt.Errorf("resolve allow-dir %q: %w", d, err)
		}
		info, err := os.Stat(real)
		if err != nil {
			return nil, fmt.Errorf("stat allow-dir %q: %w", d, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("allow-dir %q is not a directory", d)
		}
		if !seen[real] {
			seen[real] = true
			out = append(out, real)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid allow-dir was provided")
	}
	return out, nil
}
