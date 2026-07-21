// Command mcp-fetch is a local (stdio) MCP server exposing a single
// "fetch_url" tool that lets an LLM retrieve internet content through the
// internal/secfetch SSRF-hardened client. See internal/secfetch/config.go
// for the environment variables that configure its allowlist and limits.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/johnnyc20/myproject/internal/secfetch"
)

func main() {
	cfg := secfetch.Load()
	if len(cfg.AllowedHosts) == 0 {
		log.Println("warning: MCP_FETCH_ALLOWED_HOSTS is unset — every fetch_url call will be rejected")
	}
	client := secfetch.NewClient(cfg)

	s := server.NewMCPServer("myproject-mcp-fetch", "1.0.0")
	s.AddTool(fetchURLTool(), fetchURLHandler(client))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("mcp-fetch server error: %v", err)
	}
}

func fetchURLTool() mcp.Tool {
	return mcp.NewTool("fetch_url",
		mcp.WithDescription(
			"Fetch the contents of a URL over HTTPS. Only hosts on this server's "+
				"configured allowlist can be reached; requests to private, loopback, "+
				"link-local, and cloud-metadata addresses are always rejected regardless "+
				"of the allowlist.",
		),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The https:// URL to fetch."),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

func fetchURLHandler(client *secfetch.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rawURL, err := req.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		res, err := client.Fetch(ctx, rawURL)
		if err != nil {
			// Policy rejections and network failures are reported to the
			// model as a tool error, not a protocol-level error, so it can
			// see why and try a different URL instead of the call failing
			// opaquely.
			return mcp.NewToolResultError(err.Error()), nil
		}

		text := fmt.Sprintf("HTTP %d\nContent-Type: %s\nTruncated: %t\n\n%s",
			res.StatusCode, res.Header.Get("Content-Type"), res.Truncated, res.Body)
		return mcp.NewToolResultText(text), nil
	}
}
