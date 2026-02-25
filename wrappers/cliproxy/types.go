package cliproxy

import (
	"time"

	"github.com/kooshapari/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/kooshapari/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

// Re-export key types from CLIProxyAPI for convenience

// Auth represents an authentication entry
type Auth = auth.Auth

// AuthStatus represents the status of an auth entry
type AuthStatus = auth.Status

// Status constants
const (
	StatusActive   = auth.StatusActive
	StatusError    = auth.StatusError
	StatusDisabled = auth.StatusDisabled
	StatusPending  = auth.StatusPending
)

// Request represents an execution request (re-exported from executor)
type Request = executor.Request

// Response represents an execution response
type Response = executor.Response

// StreamChunk represents a streaming response chunk
type StreamChunk = executor.StreamChunk

// Options represents execution options
type Options = executor.Options

// ExtendedRequest wraps executor.Request with additional fields for convenience
type ExtendedRequest struct {
	Request
	Messages    []map[string]interface{} `json:"messages,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature *float64                 `json:"temperature,omitempty"`
}

// ProviderExecutor is the interface for provider executors
type ProviderExecutor = auth.ProviderExecutor

// AuthResult captures execution outcome
type AuthResult = auth.Result

// AuthError represents an auth error
type AuthError = auth.Error

// AuthHook captures lifecycle callbacks
type AuthHook = auth.Hook

// AuthSelector chooses an auth candidate
type AuthSelector = auth.Selector

// Manager is the auth manager
type Manager = auth.Manager

// NewAuthManager creates a new auth manager
func NewAuthManager(store auth.Store, selector auth.Selector, hook auth.Hook) *Manager {
	return auth.NewManager(store, selector, hook)
}

// ModelState represents per-model state
type ModelState struct {
	Model          string    `json:"model"`
	Status         string    `json:"status"`
	Unavailable    bool      `json:"unavailable"`
	NextRetryAfter time.Time `json:"next_retry_after,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
}

// QuotaState represents quota tracking state
type QuotaState struct {
	Exceeded      bool      `json:"exceeded"`
	Reason        string    `json:"reason,omitempty"`
	NextRecoverAt time.Time `json:"next_recover_at,omitempty"`
	BackoffLevel  int       `json:"backoff_level"`
}

// AuthInfo provides a summary of an auth entry
type AuthInfo struct {
	ID           string            `json:"id"`
	Provider     string            `json:"provider"`
	Label        string            `json:"label"`
	Status       string            `json:"status"`
	Disabled     bool              `json:"disabled"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	LastUsed     time.Time         `json:"last_used,omitempty"`
	ModelCount   int               `json:"model_count"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// ToAuthInfo converts an Auth to AuthInfo summary
func ToAuthInfo(a *Auth) *AuthInfo {
	if a == nil {
		return nil
	}
	return &AuthInfo{
		ID:         a.ID,
		Provider:   a.Provider,
		Label:      a.Label,
		Status:     string(a.Status),
		Disabled:   a.Disabled,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
		Attributes: a.Attributes,
	}
}

// ExtendedRequestBuilder helps construct extended execution requests
type ExtendedRequestBuilder struct {
	req ExtendedRequest
}

// NewExtendedRequest creates a new extended request builder
func NewExtendedRequest() *ExtendedRequestBuilder {
	return &ExtendedRequestBuilder{
		req: ExtendedRequest{},
	}
}

// WithModel sets the model
func (b *ExtendedRequestBuilder) WithModel(model string) *ExtendedRequestBuilder {
	b.req.Model = model
	return b
}

// WithMessages sets the messages
func (b *ExtendedRequestBuilder) WithMessages(messages []map[string]interface{}) *ExtendedRequestBuilder {
	b.req.Messages = messages
	return b
}

// WithStream sets streaming mode
func (b *ExtendedRequestBuilder) WithStream(stream bool) *ExtendedRequestBuilder {
	b.req.Stream = stream
	return b
}

// WithMaxTokens sets max tokens
func (b *ExtendedRequestBuilder) WithMaxTokens(max int) *ExtendedRequestBuilder {
	b.req.MaxTokens = max
	return b
}

// WithTemperature sets temperature
func (b *ExtendedRequestBuilder) WithTemperature(temp float64) *ExtendedRequestBuilder {
	b.req.Temperature = &temp
	return b
}

// WithPayload sets the raw payload
func (b *ExtendedRequestBuilder) WithPayload(payload []byte) *ExtendedRequestBuilder {
	b.req.Payload = payload
	return b
}

// Build returns the constructed extended request
func (b *ExtendedRequestBuilder) Build() ExtendedRequest {
	return b.req
}

// ToRequest converts to base Request (for executor compatibility)
func (b *ExtendedRequestBuilder) ToRequest() Request {
	return b.req.Request
}

// OptionsBuilder helps construct execution options
type OptionsBuilder struct {
	opts Options
}

// NewOptions creates a new options builder
func NewOptions() *OptionsBuilder {
	return &OptionsBuilder{
		opts: Options{},
	}
}

// WithStream sets streaming mode
func (b *OptionsBuilder) WithStream(stream bool) *OptionsBuilder {
	b.opts.Stream = stream
	return b
}

// WithAlt sets the alternate format hint
func (b *OptionsBuilder) WithAlt(alt string) *OptionsBuilder {
	b.opts.Alt = alt
	return b
}

// WithMetadata sets execution metadata
func (b *OptionsBuilder) WithMetadata(metadata map[string]any) *OptionsBuilder {
	b.opts.Metadata = metadata
	return b
}

// Build returns the constructed options
func (b *OptionsBuilder) Build() Options {
	return b.opts
}

