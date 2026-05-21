package cassandra

import (
	"context"
	"fmt"
	"time"

	reactionsService "nosql/internal/service/reactions"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

const (
	eventReactionsTable = "event_reactions"
	eventReviewsTable   = "event_reviews"
)

type ReactionRepository struct {
	session *gocql.Session
}

func NewReactionRepository(session *gocql.Session) *ReactionRepository {
	return &ReactionRepository{session: session}
}

func (r *ReactionRepository) Upsert(ctx context.Context, eventID, userID string, likeValue bool, createdAt time.Time) error {
	value := int8(-1)
	if likeValue {
		value = 1
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (event_id, created_by, like_value, created_at) VALUES (?, ?, ?, ?)",
		eventReactionsTable,
	)

	return r.session.Query(query, eventID, userID, value, createdAt).WithContext(ctx).Exec()
}

func (r *ReactionRepository) CountByEventIDs(ctx context.Context, eventIDs []string) (map[string]reactionsService.Counts, error) {
	result := make(map[string]reactionsService.Counts, len(eventIDs))
	for _, eventID := range eventIDs {
		counts, err := r.countByEventID(ctx, eventID)
		if err != nil {
			return nil, err
		}
		result[eventID] = counts
	}
	return result, nil
}

func (r *ReactionRepository) countByEventID(ctx context.Context, eventID string) (reactionsService.Counts, error) {
	query := fmt.Sprintf("SELECT like_value FROM %s WHERE event_id = ?", eventReactionsTable)
	iter := r.session.Query(query, eventID).WithContext(ctx).Iter()

	counts := reactionsService.Counts{}
	var likeValue int8
	for iter.Scan(&likeValue) {
		if likeValue == 1 {
			counts.Likes++
		} else {
			counts.Dislikes++
		}
	}

	if err := iter.Close(); err != nil {
		return reactionsService.Counts{}, err
	}

	return counts, nil
}
