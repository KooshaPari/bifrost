// Package queryscope provides the primitive that wrappers use to push a
// per-call SQL constraint onto the request context for inner stores to
// consume. The configstore and logstore packages both import it so the
// same QueryScope mechanism powers their ScopedDB read paths without
// introducing a cycle between the two stores.
//
// Security contract: a missing, empty, or wrong-typed QueryScope on a
// context is NEVER treated as "no restriction" by the strict accessors.
// Strict callers (ScopedDB on the data stores) MUST use
// RequireFromContext and refuse to issue a query when no scope is set —
// silently returning unscoped rows would cross tenant boundaries and
// leak data across the DAC scope boundary.
package queryscope

import (
	"context"
	"errors"
	"fmt"

	"github.com/maximhq/bifrost/core/schemas"
	"gorm.io/gorm"
)

// ErrMissingScope is returned by RequireFromContext when no QueryScope
// is associated with the supplied context. Callers that resolve scopes
// at query time (e.g. ScopedDB) MUST treat this as a hard failure and
// refuse to issue the query — falling back to an unscoped query would
// read rows across tenants and bypass the per-call DAC boundary.
var ErrMissingScope = errors.New("queryscope: no scope on context; refusing to issue an unscoped query")

// QueryScope mutates a query to enforce caller-driven row-level
// constraints. Set on ctx by an upstream wrapper; inner store query
// helpers apply it blindly via ScopedDB.
type QueryScope func(*gorm.DB) *gorm.DB

// IsMissingScopeError reports whether err (or any error in its chain)
// is ErrMissingScope. Useful for callers that need to distinguish a
// scope-related rejection from other store failures.
func IsMissingScopeError(err error) bool {
	return errors.Is(err, ErrMissingScope)
}

// WithQueryScope returns ctx carrying scope. A nil scope is a no-op
// (the original ctx is returned untouched) so wrappers that
// conditionally attach a scope can short-circuit cleanly.
func WithQueryScope(ctx context.Context, scope QueryScope) context.Context {
	if scope == nil {
		return ctx
	}
	return context.WithValue(ctx, schemas.BifrostContextKeyQueryScope, scope)
}

// FromContext returns the scope stashed on ctx, or nil when no scope
// is present.
//
// SECURITY NOTE: this is the PERMISSIVE accessor. Returning nil here
// means "no scope is present", which is NOT safe for query execution —
// code that runs a DB query against the result MUST use
// RequireFromContext (or the strict ScopedDB wrappers in the configstore
// and logstore packages) so a missing scope fails closed instead of
// silently reading across the DAC boundary.
//
// Permissive callers of FromContext are limited to non-query contexts
// such as cache-suitability checks (e.g. shouldUseFilterDataCache) that
// only inspect whether a scope was attached to ctx.
func FromContext(ctx context.Context) QueryScope {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(schemas.BifrostContextKeyQueryScope).(QueryScope); ok {
		return v
	}
	return nil
}

// RequireFromContext is the strict, fail-closed counterpart of
// FromContext. It MUST be used by any code path that is about to issue
// a SQL query whose row visibility depends on the QueryScope on ctx.
//
// Behavior:
//   - ctx is nil → returns ErrMissingScope (no ctx, no scope, no query).
//   - no value stashed at the scope key → ErrMissingScope. Falling back
//     to an unscoped query here is the cross-tenant leak this helper
//     exists to prevent.
//   - a non-QueryScope value at the scope key (foreign caller polluted
//     the key) → ErrMissingScope. Treating a wrong-typed value as a
//     no-op scope would also bypass the DAC boundary.
//   - a typed-nil QueryScope stashed via WithQueryScope → ErrMissingScope.
//     Invoking a nil function would panic; refusing to query is safer.
//   - a non-nil QueryScope → returns it with a nil error and the query
//     may proceed under that scope.
//
// Wrappers that build a DAC-aware read path should obtain the scope via
// RequireFromContext and propagate the error to the caller — that is
// the only behavior that closes the gap when a request arrives without
// a wrapper-attached scope (background jobs, internal lookups, OSS
// deployments, or any new code path that forgets to attach a scope).
func RequireFromContext(ctx context.Context) (QueryScope, error) {
	if ctx == nil {
		return nil, fmt.Errorf("RequireFromContext: %w", ErrMissingScope)
	}
	v := ctx.Value(schemas.BifrostContextKeyQueryScope)
	if v == nil {
		return nil, fmt.Errorf("RequireFromContext: %w", ErrMissingScope)
	}
	scope, ok := v.(QueryScope)
	if !ok {
		return nil, fmt.Errorf("RequireFromContext: value at scope key is not a QueryScope (%T): %w", v, ErrMissingScope)
	}
	if scope == nil {
		return nil, fmt.Errorf("RequireFromContext: %w", ErrMissingScope)
	}
	return scope, nil
}
