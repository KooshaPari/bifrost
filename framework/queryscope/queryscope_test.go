package queryscope

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// recordingScope is a sentinel QueryScope that flips a flag when invoked
// so tests can assert whether the wrapper actually called it.
type recordingScope struct{ called bool }

func (r *recordingScope) apply(db *gorm.DB) *gorm.DB {
	r.called = true
	return db
}

func TestWithQueryScope_NilScopeIsNoOp(t *testing.T) {
	ctx := context.Background()
	out := WithQueryScope(ctx, nil)
	// Nil scope must not stash anything, so FromContext on the returned
	// ctx must report no scope present.
	assert.Nil(t, FromContext(out), "nil scope should not be retrievable as a scope")
}

func TestWithQueryScope_StashesScope(t *testing.T) {
	r := &recordingScope{}
	ctx := WithQueryScope(context.Background(), r.apply)

	got := FromContext(ctx)
	if assert.NotNil(t, got, "scope should be retrievable from ctx") {
		got(nil)
		assert.True(t, r.called, "retrieved scope should be the same closure that was stashed")
	}
}

func TestFromContext_NilCtxReturnsNil(t *testing.T) {
	assert.Nil(t, FromContext(nil))
}

func TestFromContext_MissingKeyReturnsNil(t *testing.T) {
	assert.Nil(t, FromContext(context.Background()))
}

func TestFromContext_WrongTypeReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(),
		schemas.BifrostContextKeyQueryScope, "not a closure")
	assert.Nil(t, FromContext(ctx),
		"a value of the wrong type at the scope key must not be treated as a scope")
}

func TestFromContext_NilInterfaceValueReturnsNil(t *testing.T) {
	// A typed nil QueryScope stashed on ctx must be safe to retrieve.
	// The retrieved scope can be nil, but FromContext must not panic.
	ctx := context.WithValue(context.Background(),
		schemas.BifrostContextKeyQueryScope, QueryScope(nil))
	got := FromContext(ctx)
	assert.Nil(t, got, "typed-nil QueryScope must be returned as a usable nil")
}

func TestWithQueryScope_NilCtxIsSafe(t *testing.T) {
	// Defensive: callers should pass a real ctx, but nil ctx must
	// not panic. Go's context.WithValue panics on nil parent so
	// WithQueryScope short-circuits via the nil-scope guard.
	out := WithQueryScope(nil, nil)
	assert.Nil(t, out, "nil ctx + nil scope must propagate nil cleanly")
}

func TestFromContext_CancelledCtxStillReturnsScope(t *testing.T) {
	// Cancellation must not affect scope retrieval; the scope is a
	// value on ctx, not a lifecycle resource.
	parent, cancel := context.WithCancel(context.Background())
	r := &recordingScope{}
	ctx := WithQueryScope(parent, r.apply)
	cancel()
	got := FromContext(ctx)
	if assert.NotNil(t, got, "cancellation should not strip the scope value") {
		got(nil)
		assert.True(t, r.called)
	}
}

// --- Strict / fail-closed accessors ---

func TestRequireFromContext_NilCtxReturnsError(t *testing.T) {
	scope, err := RequireFromContext(nil)
	assert.Nil(t, scope)
	assert.Error(t, err, "nil ctx must fail closed")
	assert.True(t, IsMissingScopeError(err),
		"nil ctx error must wrap ErrMissingScope")
}

func TestRequireFromContext_MissingKeyReturnsError(t *testing.T) {
	// The headline regression test: a background ctx that never had a
	// QueryScope attached MUST be rejected, not silently treated as
	// "unrestricted". This is the gap that allowed cross-tenant reads
	// in the pre-fix ScopedDB code path.
	scope, err := RequireFromContext(context.Background())
	assert.Nil(t, scope,
		"missing scope must not be silently coerced to nil-and-proceed")
	assert.Error(t, err,
		"missing scope must fail closed with ErrMissingScope")
	assert.True(t, IsMissingScopeError(err),
		"errors.Is(err, ErrMissingScope) must be true so callers can branch on it")
}

func TestRequireFromContext_TypedNilScopeReturnsError(t *testing.T) {
	// WithQueryScope(nil) stashes nothing, so RequireFromContext should
	// see "no scope present" and fail closed — invoking a nil closure
	// would panic, and a no-op fallback would leak rows.
	ctx := WithQueryScope(context.Background(), nil)
	scope, err := RequireFromContext(ctx)
	assert.Nil(t, scope)
	assert.Error(t, err)
	assert.True(t, IsMissingScopeError(err))
}

func TestRequireFromContext_WrongTypeAtKeyReturnsError(t *testing.T) {
	// A foreign caller stashing a wrong-typed value at the scope key
	// must NOT be treated as a no-op scope — that would bypass the DAC
	// boundary. The strict accessor rejects it.
	ctx := context.WithValue(context.Background(),
		schemas.BifrostContextKeyQueryScope, "not a closure")
	scope, err := RequireFromContext(ctx)
	assert.Nil(t, scope)
	assert.Error(t, err,
		"wrong-typed value at the scope key must fail closed")
	assert.True(t, IsMissingScopeError(err))
}

func TestRequireFromContext_NonNilScopeReturnsScopeAndNoError(t *testing.T) {
	// Happy path: a real scope attached via WithQueryScope is returned
	// with nil error and is the same closure that was stashed.
	r := &recordingScope{}
	ctx := WithQueryScope(context.Background(), r.apply)

	scope, err := RequireFromContext(ctx)
	require.NoError(t, err)
	require.NotNil(t, scope)
	scope(nil)
	assert.True(t, r.called,
		"the strict accessor must hand back the original closure so the scope is applied as the caller intended")
}

func TestRequireFromContext_PermissiveFromContextStaysPermissive(t *testing.T) {
	// FromContext is intentionally the permissive accessor and is kept
	// for non-query callers (e.g. cache-suitability checks like
	// shouldUseFilterDataCache). This test pins that contract: a
	// background ctx returns nil from FromContext without panicking,
	// even though RequireFromContext on the same ctx returns an error.
	ctx := context.Background()
	assert.Nil(t, FromContext(ctx),
		"FromContext must keep returning nil for callers that explicitly want the permissive signal")

	_, err := RequireFromContext(ctx)
	assert.Error(t, err,
		"RequireFromContext on the same ctx must still fail closed")
}

func TestIsMissingScopeError_TrueForWrappedErrors(t *testing.T) {
	// ErrMissingScope is wrapped via %w in every RequireFromContext
	// error path; IsMissingScopeError must follow errors.Is semantics
	// (true for any error in the chain), so callers can branch on
	// it without unwinding manually.
	wrapped := fmt.Errorf("scoped read failed: %w", ErrMissingScope)
	assert.True(t, IsMissingScopeError(wrapped),
		"IsMissingScopeError must be true for errors that wrap ErrMissingScope")
}

func TestIsMissingScopeError_FalseForOtherErrors(t *testing.T) {
	other := errors.New("some other failure")
	assert.False(t, IsMissingScopeError(other),
		"IsMissingScopeError must not match unrelated errors")
	assert.False(t, IsMissingScopeError(nil),
		"IsMissingScopeError(nil) must be false")
}
