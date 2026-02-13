package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"

	"github.com/kineticloom/plydb/mcpserver"
	"github.com/kineticloom/plydb/semanticcontext"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RunMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the connection config JSON file")
	transport := fs.String("transport", "stdio", "transport type: stdio or http")
	addr := fs.String("addr", "localhost:8080", "address for HTTP transport")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: plydb mcp [flags]

Start an MCP server exposing a SQL query tool.

Flags:`)
		fs.PrintDefaults()
	}
	fs.Parse(reorderArgs(args))

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		fs.Usage()
		os.Exit(1)
	}

	cfg, engine := LoadConfigAndEngine(*configPath)
	defer engine.Close()

	provider := semanticcontext.NewAutoScanProvider(cfg, engine)
	model, err := provider.Provide(context.Background(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning semantic context: %v\n", err)
		os.Exit(1)
	}

	server := mcpserver.NewServer(cfg, engine, model)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch *transport {
	case "stdio":
		// Use a pipe to decouple stdin EOF from the MCP server's read loop.
		// Without this, when stdin is a pipe (e.g. echo '...' | plydb mcp),
		// EOF arrives immediately after the last message and races with
		// response writes — the SDK rejects writes once it detects EOF on
		// the read side, so responses are never sent.
		//
		// By forwarding stdin through a pipe and only closing the write end
		// when the context is cancelled, the server keeps running long enough
		// to process all messages and write responses.
		pr, pw := io.Pipe()
		go func() {
			io.Copy(pw, os.Stdin)
			// stdin closed; keep the pipe open until shutdown is requested.
			<-ctx.Done()
			pw.Close()
		}()

		t := &mcp.IOTransport{
			Reader: pr,
			Writer: nopWriteCloser{os.Stdout},
		}
		if err := server.Run(ctx, t); err != nil {
			fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
			os.Exit(1)
		}

	case "http":
		handler := mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return server },
			nil,
		)
		httpServer := &http.Server{Addr: *addr, Handler: handler}

		go func() {
			<-ctx.Done()
			httpServer.Close()
		}()

		fmt.Fprintf(os.Stderr, "MCP HTTP server listening on %s\n", *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "http server error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "error: unknown transport %q (use stdio or http)\n", *transport)
		os.Exit(1)
	}
}

// nopWriteCloser wraps an io.Writer with a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
