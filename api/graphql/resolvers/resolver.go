// Package resolvers implements GraphQL resolvers for the Bifrost API.
package resolvers

import (
	"context"

	"github.com/kooshapari/bifrost-extensions/api/graphql/gen"
	"github.com/kooshapari/bifrost-extensions/db"
)

// Resolver is the root resolver that provides access to all sub-resolvers.
type Resolver struct {
	db *db.DB
	// Add other dependencies as needed (NATS, Neo4j, etc.)
}

// NewResolver creates a new root resolver.
func NewResolver(database *db.DB) *Resolver {
	return &Resolver{
		db: database,
	}
}

// Query returns the query resolver.
func (r *Resolver) Query() gen.QueryResolver {
	return &queryResolver{r}
}

// Subscription returns the subscription resolver.
func (r *Resolver) Subscription() gen.SubscriptionResolver {
	return &subscriptionResolver{r}
}

// Mutation returns the mutation resolver (if needed).
func (r *Resolver) Mutation() gen.MutationResolver {
	return nil // No mutations for now
}
