// Package neo4j provides multi-tenant Neo4j support for sharing a single
// Neo4j Aura Free instance across multiple projects using namespace isolation.
package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ProjectNamespace represents a project's isolated namespace in the shared Neo4j instance
type ProjectNamespace string

const (
	NamespaceBifrost   ProjectNamespace = "bifrost"
	NamespaceVibeProxy ProjectNamespace = "vibeproxy"
	NamespaceJarvis    ProjectNamespace = "jarvis"
	NamespaceTrace     ProjectNamespace = "trace"
	NamespaceDefault   ProjectNamespace = "default"
)

// MultiTenantClient wraps Client with project namespace isolation
type MultiTenantClient struct {
	*Client
	namespace ProjectNamespace
}

// NewMultiTenantClient creates a client with namespace isolation
func NewMultiTenantClient(config Config, namespace ProjectNamespace) (*MultiTenantClient, error) {
	client, err := New(config)
	if err != nil {
		return nil, err
	}
	return &MultiTenantClient{
		Client:    client,
		namespace: namespace,
	}, nil
}

// InitializeNamespace sets up the namespace with constraints and indexes
func (c *MultiTenantClient) InitializeNamespace(ctx context.Context) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	// Create namespace-specific constraints
	queries := []string{
		// Unique constraint on project-scoped nodes
		fmt.Sprintf(`CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s_Model) REQUIRE n.id IS UNIQUE`, c.namespace),
		fmt.Sprintf(`CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s_Role) REQUIRE n.id IS UNIQUE`, c.namespace),
		fmt.Sprintf(`CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s_Tool) REQUIRE n.id IS UNIQUE`, c.namespace),
		fmt.Sprintf(`CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s_Policy) REQUIRE n.id IS UNIQUE`, c.namespace),
		fmt.Sprintf(`CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s_User) REQUIRE n.id IS UNIQUE`, c.namespace),

		// Indexes for common lookups
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS FOR (n:%s_Model) ON (n.name)`, c.namespace),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS FOR (n:%s_Role) ON (n.name)`, c.namespace),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS FOR (n:%s_Tool) ON (n.name)`, c.namespace),
	}

	for _, q := range queries {
		if _, err := session.Run(ctx, q, nil); err != nil {
			return fmt.Errorf("failed to initialize namespace %s: %w", c.namespace, err)
		}
	}
	return nil
}

// CreateModel creates a model node in this namespace
func (c *MultiTenantClient) CreateModel(ctx context.Context, model ModelNode) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MERGE (m:%s_Model {id: $id})
		SET m.name = $name,
		    m.provider = $provider,
		    m.updatedAt = datetime()
	`, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"id":       model.ID,
		"name":     model.Name,
		"provider": model.Provider,
	})
	return err
}

// CreateRole creates a role node in this namespace
func (c *MultiTenantClient) CreateRole(ctx context.Context, role RoleNode) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MERGE (r:%s_Role {id: $id})
		SET r.name = $name,
		    r.riskLevel = $riskLevel,
		    r.updatedAt = datetime()
	`, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"id":        role.ID,
		"name":      role.Name,
		"riskLevel": role.RiskLevel,
	})
	return err
}

// LinkModelToRole creates a PERFORMS_ON relationship
func (c *MultiTenantClient) LinkModelToRole(ctx context.Context, modelID, roleID string, score float64) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (m:%s_Model {id: $modelId})
		MATCH (r:%s_Role {id: $roleId})
		MERGE (m)-[rel:PERFORMS_ON]->(r)
		SET rel.score = $score, rel.updatedAt = datetime()
	`, c.namespace, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"modelId": modelID,
		"roleId":  roleID,
		"score":   score,
	})
	return err
}

// GetModelsForRole returns models suitable for a role in this namespace
func (c *MultiTenantClient) GetModelsForRole(ctx context.Context, roleName string) ([]ModelNode, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (m:%s_Model)-[rel:PERFORMS_ON]->(r:%s_Role {name: $roleName})
		RETURN m.id AS id, m.name AS name, m.provider AS provider, rel.score AS score
		ORDER BY rel.score DESC
	`, c.namespace, c.namespace)

	result, err := session.Run(ctx, query, map[string]any{"roleName": roleName})
	if err != nil {
		return nil, err
	}

	var models []ModelNode
	for result.Next(ctx) {
		rec := result.Record()
		models = append(models, ModelNode{
			ID:       rec.Values[0].(string),
			Name:     rec.Values[1].(string),
			Provider: rec.Values[2].(string),
		})
	}
	return models, result.Err()
}

// ModelNode represents a model in the graph
type ModelNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// RoleNode represents a role in the graph
type RoleNode struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RiskLevel string `json:"risk_level"`
}

// TraitNode represents a model/tool trait
type TraitNode struct {
	Name string `json:"name"`
	Type string `json:"type"` // performance, cost, quality, style, capability
}

// CreateTrait creates a trait node in this namespace
func (c *MultiTenantClient) CreateTrait(ctx context.Context, trait TraitNode) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MERGE (t:%s_Trait {name: $name})
		SET t.type = $type, t.updatedAt = datetime()
	`, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"name": trait.Name,
		"type": trait.Type,
	})
	return err
}

// LinkModelTrait creates a HAS_TRAIT relationship
func (c *MultiTenantClient) LinkModelTrait(ctx context.Context, modelID, traitName string, weight float64) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (m:%s_Model {id: $modelId})
		MATCH (t:%s_Trait {name: $traitName})
		MERGE (m)-[rel:HAS_TRAIT]->(t)
		SET rel.weight = $weight, rel.updatedAt = datetime()
	`, c.namespace, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"modelId":   modelID,
		"traitName": traitName,
		"weight":    weight,
	})
	return err
}

// GetModelsByTrait finds models with a specific trait
func (c *MultiTenantClient) GetModelsByTrait(ctx context.Context, traitName string, minWeight float64) ([]ModelNode, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (m:%s_Model)-[rel:HAS_TRAIT]->(t:%s_Trait {name: $traitName})
		WHERE rel.weight >= $minWeight
		RETURN m.id AS id, m.name AS name, m.provider AS provider, rel.weight AS weight
		ORDER BY rel.weight DESC
	`, c.namespace, c.namespace)

	result, err := session.Run(ctx, query, map[string]any{
		"traitName": traitName,
		"minWeight": minWeight,
	})
	if err != nil {
		return nil, err
	}

	var models []ModelNode
	for result.Next(ctx) {
		rec := result.Record()
		models = append(models, ModelNode{
			ID:       rec.Values[0].(string),
			Name:     rec.Values[1].(string),
			Provider: rec.Values[2].(string),
		})
	}
	return models, result.Err()
}

// RecordModelOutperformance records when one model outperforms another
func (c *MultiTenantClient) RecordModelOutperformance(ctx context.Context, winnerID, loserID, context string, samples int) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (winner:%s_Model {id: $winnerId})
		MATCH (loser:%s_Model {id: $loserId})
		MERGE (winner)-[rel:OUTPERFORMED]->(loser)
		ON CREATE SET rel.samples = $samples, rel.context = $context, rel.createdAt = datetime()
		ON MATCH SET rel.samples = rel.samples + $samples, rel.updatedAt = datetime()
	`, c.namespace, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"winnerId": winnerID,
		"loserId":  loserID,
		"context":  context,
		"samples":  samples,
	})
	return err
}

// ============================================================================
// TRACE NAMESPACE TYPES & METHODS
// ============================================================================

// TraceSpan represents a span in a distributed trace
type TraceSpan struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	Service      string            `json:"service"`
	StartTime    int64             `json:"start_time_ns"`
	Duration     int64             `json:"duration_ns"`
	Status       string            `json:"status"` // ok, error
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// TraceEvent represents an event within a span
type TraceEvent struct {
	SpanID    string `json:"span_id"`
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp_ns"`
	Message   string `json:"message,omitempty"`
}

// CreateSpan creates a trace span node
func (c *MultiTenantClient) CreateSpan(ctx context.Context, span TraceSpan) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MERGE (s:%s_Span {spanId: $spanId})
		SET s.traceId = $traceId,
		    s.parentSpanId = $parentSpanId,
		    s.name = $name,
		    s.service = $service,
		    s.startTime = $startTime,
		    s.duration = $duration,
		    s.status = $status,
		    s.attributes = $attributes
	`, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"spanId":       span.SpanID,
		"traceId":      span.TraceID,
		"parentSpanId": span.ParentSpanID,
		"name":         span.Name,
		"service":      span.Service,
		"startTime":    span.StartTime,
		"duration":     span.Duration,
		"status":       span.Status,
		"attributes":   span.Attributes,
	})
	return err
}

// LinkSpans creates parent-child relationship between spans
func (c *MultiTenantClient) LinkSpans(ctx context.Context, parentSpanID, childSpanID string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (parent:%s_Span {spanId: $parentSpanId})
		MATCH (child:%s_Span {spanId: $childSpanId})
		MERGE (parent)-[:PARENT_OF]->(child)
	`, c.namespace, c.namespace)

	_, err := session.Run(ctx, query, map[string]any{
		"parentSpanId": parentSpanID,
		"childSpanId":  childSpanID,
	})
	return err
}

// GetTrace retrieves all spans for a trace ID
func (c *MultiTenantClient) GetTrace(ctx context.Context, traceID string) ([]TraceSpan, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (s:%s_Span {traceId: $traceId})
		RETURN s.spanId, s.parentSpanId, s.name, s.service, s.startTime, s.duration, s.status
		ORDER BY s.startTime
	`, c.namespace)

	result, err := session.Run(ctx, query, map[string]any{"traceId": traceID})
	if err != nil {
		return nil, err
	}

	var spans []TraceSpan
	for result.Next(ctx) {
		rec := result.Record()
		span := TraceSpan{
			TraceID:   traceID,
			SpanID:    safeString(rec.Values[0]),
			ParentSpanID: safeString(rec.Values[1]),
			Name:      safeString(rec.Values[2]),
			Service:   safeString(rec.Values[3]),
			StartTime: safeInt64(rec.Values[4]),
			Duration:  safeInt64(rec.Values[5]),
			Status:    safeString(rec.Values[6]),
		}
		spans = append(spans, span)
	}
	return spans, result.Err()
}

// GetSpanTree returns the full tree of spans from a root span
func (c *MultiTenantClient) GetSpanTree(ctx context.Context, rootSpanID string) ([]TraceSpan, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (root:%s_Span {spanId: $rootSpanId})
		MATCH path = (root)-[:PARENT_OF*0..]->(descendant:%s_Span)
		RETURN descendant.spanId, descendant.parentSpanId, descendant.name,
		       descendant.service, descendant.startTime, descendant.duration,
		       descendant.status, descendant.traceId
		ORDER BY descendant.startTime
	`, c.namespace, c.namespace)

	result, err := session.Run(ctx, query, map[string]any{"rootSpanId": rootSpanID})
	if err != nil {
		return nil, err
	}

	var spans []TraceSpan
	for result.Next(ctx) {
		rec := result.Record()
		span := TraceSpan{
			SpanID:       safeString(rec.Values[0]),
			ParentSpanID: safeString(rec.Values[1]),
			Name:         safeString(rec.Values[2]),
			Service:      safeString(rec.Values[3]),
			StartTime:    safeInt64(rec.Values[4]),
			Duration:     safeInt64(rec.Values[5]),
			Status:       safeString(rec.Values[6]),
			TraceID:      safeString(rec.Values[7]),
		}
		spans = append(spans, span)
	}
	return spans, result.Err()
}

// Helper functions for safe type conversion
func safeString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func safeInt64(v any) int64 {
	if v == nil {
		return 0
	}
	if i, ok := v.(int64); ok {
		return i
	}
	return 0
}
