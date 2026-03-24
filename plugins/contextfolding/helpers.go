package contextfolding

import (
	"context"
	"strings"

	"github.com/kooshapari/bifrost-extensions/slm"
	"github.com/maximhq/bifrost/core/schemas"
)

// estimateTokens estimates tokens in a message
func (cf *ContextFolding) estimateTokens(msg *schemas.ChatMessage) int {
	if msg.Content == nil {
		return 0
	}
	if msg.Content.ContentStr != nil {
		return len(*msg.Content.ContentStr) / 4
	}
	if msg.Content.ContentBlocks != nil {
		total := 0
		for _, block := range msg.Content.ContentBlocks {
			if block.Text != nil {
				total += len(*block.Text) / 4
			}
		}
		return total
	}
	return 0
}

// messagesToText converts messages to plain text
func (cf *ContextFolding) messagesToText(messages []schemas.ChatMessage) string {
	var sb strings.Builder
	for _, msg := range messages {
		if msg.Content != nil && msg.Content.ContentStr != nil {
			sb.WriteString(string(msg.Role))
			sb.WriteString(": ")
			sb.WriteString(*msg.Content.ContentStr)
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// summarizeResponse summarizes a response for future context
func (cf *ContextFolding) summarizeResponse(ctx context.Context, resp *schemas.BifrostResponse) {
	if cf.slmClients == nil || cf.slmClients.Summarizer == nil {
		return
	}

	// Extract response content
	var content string
	if resp.ChatCompletion != nil && len(resp.ChatCompletion.Choices) > 0 {
		choice := resp.ChatCompletion.Choices[0]
		if choice.Message != nil && choice.Message.Content != nil {
			content = *choice.Message.Content
		}
	}

	if content == "" || len(content) < cf.config.SummarizeThreshold {
		return
	}

	// Generate multi-resolution summaries asynchronously
	_, _ = cf.slmClients.Summarize(ctx, &slm.SummarizeRequest{
		Text:          content,
		Mode:          "response",
		DesiredLength: "short",
	})
}

// RetrieveRelevantContext retrieves relevant context from database
func (cf *ContextFolding) RetrieveRelevantContext(
	ctx context.Context,
	embedding []float32,
	sessionID string,
	limit int,
) ([]string, error) {
	if cf.queries == nil {
		return nil, nil
	}

	// This would use vector similarity search
	// For now, return empty
	// TODO: Implement once we have embedding generation
	return nil, nil
}
