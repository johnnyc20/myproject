// Command mcp-deepseek is a local (stdio) MCP server exposing a single
// "deepseek_chat" tool that lets an LLM delegate a prompt to DeepSeek's
// chat-completions API. See internal/deepseek/config.go for the
// environment variables that configure it -- DEEPSEEK_API_KEY is required;
// every call is rejected with a clear error until it's set.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/johnnyc20/myproject/internal/deepseek"
)

func main() {
	cfg := deepseek.Load()
	if cfg.APIKey == "" {
		log.Println("warning: DEEPSEEK_API_KEY is unset — every deepseek_chat call will be rejected")
	}
	client := deepseek.NewClient(cfg)

	s := server.NewMCPServer("myproject-mcp-deepseek", "1.0.0")
	s.AddTool(deepseekChatTool(), deepseekChatHandler(client))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("mcp-deepseek server error: %v", err)
	}
}

func deepseekChatTool() mcp.Tool {
	return mcp.NewTool("deepseek_chat",
		mcp.WithDescription(
			"Send a prompt to DeepSeek's chat-completions API and return its reply. "+
				"Useful for delegating a question to DeepSeek specifically (e.g. to compare "+
				"its answer against your own). Requires DEEPSEEK_API_KEY to be configured on "+
				"this server.",
		),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("The user message to send."),
		),
		mcp.WithString("system",
			mcp.Description("Optional system prompt to set DeepSeek's behavior for this call."),
		),
		mcp.WithString("model",
			mcp.Description("Optional model override (defaults to this server's DEEPSEEK_MODEL setting, e.g. \"deepseek-chat\" or \"deepseek-reasoner\")."),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

func deepseekChatHandler(client *deepseek.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt, err := req.RequireString("prompt")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		system := req.GetString("system", "")
		model := req.GetString("model", "")

		messages := make([]deepseek.Message, 0, 2)
		if system != "" {
			messages = append(messages, deepseek.Message{Role: "system", Content: system})
		}
		messages = append(messages, deepseek.Message{Role: "user", Content: prompt})

		result, err := client.Chat(ctx, deepseek.ChatRequest{Model: model, Messages: messages})
		if err != nil {
			// API errors and network failures are reported to the model as
			// a tool error, not a protocol-level error, matching fetch_url's
			// convention -- the caller can see why and decide what to do.
			return mcp.NewToolResultError(err.Error()), nil
		}

		text := result.Content
		if result.ReasoningContent != "" {
			text = fmt.Sprintf("Reasoning:\n%s\n\nAnswer:\n%s", result.ReasoningContent, result.Content)
		}
		text += fmt.Sprintf("\n\n[model=%s finish_reason=%s tokens=%d+%d=%d]",
			result.Model, result.FinishReason, result.PromptTokens, result.CompletionTokens, result.TotalTokens)

		return mcp.NewToolResultText(text), nil
	}
}
