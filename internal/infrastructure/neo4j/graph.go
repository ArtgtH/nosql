package neo4j

import (
	"context"

	recommendationsService "nosql/internal/service/recommendations"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Graph struct {
	driver neo4jdriver.DriverWithContext
}

func NewGraph(driver neo4jdriver.DriverWithContext) *Graph {
	return &Graph{driver: driver}
}

func (g *Graph) CreateUser(ctx context.Context, id string) error {
	query := `MERGE (:User {id: $id})`
	_, err := g.write(ctx, query, map[string]any{"id": id})
	return err
}

func (g *Graph) CreateEvent(ctx context.Context, id, title string) error {
	query := `
MERGE (e:Event {id: $id})
SET e.title = $title`
	_, err := g.write(ctx, query, map[string]any{
		"id":    id,
		"title": title,
	})
	return err
}

func (g *Graph) LikeEvent(ctx context.Context, userID, eventID, title string) error {
	query := `
MERGE (u:User {id: $user_id})
MERGE (e:Event {id: $event_id})
SET e.title = $title
MERGE (u)-[:LIKED]->(e)`
	_, err := g.write(ctx, query, map[string]any{
		"user_id":  userID,
		"event_id": eventID,
		"title":    title,
	})
	return err
}

func (g *Graph) Recommendations(ctx context.Context, userID string) ([]recommendationsService.Candidate, error) {
	query := `
MATCH (u:User {id: $user_id})-[:LIKED]->(:Event)<-[:LIKED]-(other:User)-[:LIKED]->(candidate:Event)
WHERE NOT (u)-[:LIKED]->(candidate)
RETURN candidate.id AS event_id, count(DISTINCT other) AS score
ORDER BY score DESC`
	result, err := g.read(ctx, query, map[string]any{"user_id": userID})
	if err != nil {
		return nil, err
	}

	records, ok := result.([]recommendationsService.Candidate)
	if !ok {
		return []recommendationsService.Candidate{}, nil
	}
	return records, nil
}

func (g *Graph) write(ctx context.Context, query string, params map[string]any) (any, error) {
	session := g.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)
	return session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		_, err = result.Consume(ctx)
		return nil, err
	})
}

func (g *Graph) read(ctx context.Context, query string, params map[string]any) (any, error) {
	session := g.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)
	return session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		recommendations := []recommendationsService.Candidate{}
		for result.Next(ctx) {
			record := result.Record()
			eventID, _ := record.Get("event_id")
			score, _ := record.Get("score")
			id, ok := eventID.(string)
			if !ok || id == "" {
				continue
			}
			recommendations = append(recommendations, recommendationsService.Candidate{
				EventID: id,
				Score:   toInt(score),
			})
		}
		if err := result.Err(); err != nil {
			return nil, err
		}
		return recommendations, nil
	})
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	default:
		return 0
	}
}
