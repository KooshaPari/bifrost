package toolrouter

import (
	"context"

	"github.com/kooshapari/bifrost-extensions/slm"
	"github.com/maximhq/bifrost/core/schemas"
)

// getToolProfile gets tool profile from context
func (tr *ToolRouter) getToolProfile(ctx context.Context) slm.ToolProfile {
	if profile, ok := ctx.Value(toolProfileKey).(slm.ToolProfile); ok {
		return profile
	}
	return slm.ToolProfile{}
}

// filterTools filters and prioritizes tools based on profile
func (tr *ToolRouter) filterTools(
	ctx context.Context,
	req *schemas.BifrostRequest,
	profile slm.ToolProfile,
) *schemas.BifrostRequest {
	if req.ChatRequest == nil || req.ChatRequest.Params == nil {
		return req
	}

	tools := req.ChatRequest.Params.Tools
	if tools == nil || len(tools) == 0 {
		return req
	}

	// If no profile, try to get one from SLM
	if len(profile.PrioritizedTools) == 0 && tr.slmClients != nil {
		profile = tr.classifyTools(ctx, req)
	}

	// Filter and reorder tools
	filteredTools := tr.applyProfile(tools, profile)

	// Limit number of tools
	if len(filteredTools) > tr.config.MaxToolsPerRequest {
		filteredTools = filteredTools[:tr.config.MaxToolsPerRequest]
	}

	// Create modified request
	modifiedReq := *req
	modifiedReq.ChatRequest = &schemas.ChatCompletionRequest{
		Input:  req.ChatRequest.Input,
		Params: req.ChatRequest.Params,
	}
	modifiedReq.ChatRequest.Params.Tools = filteredTools

	return &modifiedReq
}

// classifyTools uses SLM to classify tools for this request
func (tr *ToolRouter) classifyTools(ctx context.Context, req *schemas.BifrostRequest) slm.ToolProfile {
	if tr.slmClients == nil || tr.slmClients.Router == nil {
		return slm.ToolProfile{}
	}

	// Build tool names list
	var toolNames []string
	if req.ChatRequest != nil && req.ChatRequest.Params != nil && req.ChatRequest.Params.Tools != nil {
		for _, tool := range req.ChatRequest.Params.Tools {
			if tool.Function != nil {
				toolNames = append(toolNames, tool.Function.Name)
			}
		}
	}

	// Get last user message for context
	var userMessage string
	if req.ChatRequest != nil && req.ChatRequest.Input != nil {
		for i := len(req.ChatRequest.Input) - 1; i >= 0; i-- {
			msg := req.ChatRequest.Input[i]
			if msg.Role == "user" && msg.Content != nil && msg.Content.ContentStr != nil {
				userMessage = *msg.Content.ContentStr
				break
			}
		}
	}

	// Call router SLM to classify
	resp, err := tr.slmClients.Classify(ctx, &slm.ClassifyRequest{
		Text:   userMessage,
		Labels: toolNames,
		Mode:   "tool_selection",
	})
	if err != nil {
		return slm.ToolProfile{}
	}

	// Build profile from classification
	profile := slm.ToolProfile{
		PrioritizedTools: make([]string, 0, len(resp.Classifications)),
	}
	for _, c := range resp.Classifications {
		if c.Confidence >= tr.config.SemanticMatchThreshold {
			profile.PrioritizedTools = append(profile.PrioritizedTools, c.Label)
		}
	}

	return profile
}

// applyProfile reorders tools based on profile
func (tr *ToolRouter) applyProfile(tools []schemas.Tool, profile slm.ToolProfile) []schemas.Tool {
	if len(profile.PrioritizedTools) == 0 {
		return tools
	}

	// Create priority map
	priority := make(map[string]int)
	for i, name := range profile.PrioritizedTools {
		priority[name] = i
	}

	// Separate prioritized and other tools
	prioritized := make([]schemas.Tool, 0)
	other := make([]schemas.Tool, 0)

	for _, tool := range tools {
		if tool.Function != nil {
			if _, ok := priority[tool.Function.Name]; ok {
				prioritized = append(prioritized, tool)
			} else if tr.config.FallbackToAllTools {
				other = append(other, tool)
			}
		}
	}

	// Combine: prioritized first, then others
	result := make([]schemas.Tool, 0, len(prioritized)+len(other))
	result = append(result, prioritized...)
	result = append(result, other...)

	return result
}

// trackToolUsage tracks which tools were used for learning
func (tr *ToolRouter) trackToolUsage(ctx context.Context, resp *schemas.BifrostResponse) {
	// TODO: Extract tool calls from response and record metrics
}
