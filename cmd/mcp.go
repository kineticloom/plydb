package cmd

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ypt/experiment-nexus/mcpserver"
)

func RunMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the connection config JSON file")
	transport := fs.String("transport", "stdio", "transport type: stdio or http")
	addr := fs.String("addr", "localhost:8080", "address for HTTP transport")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: nexus mcp [flags]

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

	server := mcpserver.NewServer(cfg, engine)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch *transport {
	case "stdio":
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
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
