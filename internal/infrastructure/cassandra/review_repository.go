package cassandra

import (
	"context"
	"fmt"
	"time"

	reviewsService "nosql/internal/service/reviews"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type ReviewRepository struct {
	session *gocql.Session
}

func NewReviewRepository(session *gocql.Session) *ReviewRepository {
	return &ReviewRepository{session: session}
}

func (r *ReviewRepository) Create(ctx context.Context, review reviewsService.Review) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (event_id, created_by, id, rating, comment, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		eventReviewsTable,
	)
	return r.session.Query(query,
		review.EventID,
		review.CreatedBy,
		review.ID,
		review.Rating,
		review.Comment,
		review.CreatedAt,
		review.UpdatedAt,
	).WithContext(ctx).Exec()
}

func (r *ReviewRepository) GetByEventAndUser(ctx context.Context, eventID, userID string) (reviewsService.Review, bool, error) {
	query := fmt.Sprintf(
		"SELECT id, event_id, rating, comment, created_at, created_by, updated_at FROM %s WHERE event_id = ? AND created_by = ?",
		eventReviewsTable,
	)
	var review reviewsService.Review
	err := r.session.Query(query, eventID, userID).WithContext(ctx).Scan(
		&review.ID,
		&review.EventID,
		&review.Rating,
		&review.Comment,
		&review.CreatedAt,
		&review.CreatedBy,
		&review.UpdatedAt,
	)
	if err == gocql.ErrNotFound {
		return reviewsService.Review{}, false, nil
	}
	if err != nil {
		return reviewsService.Review{}, false, err
	}
	return review, true, nil
}

func (r *ReviewRepository) GetByEventAndID(ctx context.Context, eventID, reviewID string) (reviewsService.Review, bool, error) {
	reviews, err := r.ListByEventID(ctx, eventID, 0, 0)
	if err != nil {
		return reviewsService.Review{}, false, err
	}
	for _, review := range reviews {
		if review.ID == reviewID {
			return review, true, nil
		}
	}
	return reviewsService.Review{}, false, nil
}

func (r *ReviewRepository) UpdateByEventAndUser(ctx context.Context, eventID, userID string, patch reviewsService.Patch, updatedAt time.Time) error {
	if patch.Comment != nil && patch.Rating != nil {
		query := fmt.Sprintf("UPDATE %s SET comment = ?, rating = ?, updated_at = ? WHERE event_id = ? AND created_by = ?", eventReviewsTable)
		return r.session.Query(query, *patch.Comment, *patch.Rating, updatedAt, eventID, userID).WithContext(ctx).Exec()
	}
	if patch.Comment != nil {
		query := fmt.Sprintf("UPDATE %s SET comment = ?, updated_at = ? WHERE event_id = ? AND created_by = ?", eventReviewsTable)
		return r.session.Query(query, *patch.Comment, updatedAt, eventID, userID).WithContext(ctx).Exec()
	}
	if patch.Rating != nil {
		query := fmt.Sprintf("UPDATE %s SET rating = ?, updated_at = ? WHERE event_id = ? AND created_by = ?", eventReviewsTable)
		return r.session.Query(query, *patch.Rating, updatedAt, eventID, userID).WithContext(ctx).Exec()
	}
	query := fmt.Sprintf("UPDATE %s SET updated_at = ? WHERE event_id = ? AND created_by = ?", eventReviewsTable)
	return r.session.Query(query, updatedAt, eventID, userID).WithContext(ctx).Exec()
}

func (r *ReviewRepository) ListByEventID(ctx context.Context, eventID string, limit, offset uint64) ([]reviewsService.Review, error) {
	query := fmt.Sprintf(
		"SELECT id, event_id, rating, comment, created_at, created_by, updated_at FROM %s WHERE event_id = ?",
		eventReviewsTable,
	)
	iter := r.session.Query(query, eventID).WithContext(ctx).Iter()
	reviews, err := scanReviews(iter)
	if err != nil {
		return nil, err
	}

	if offset >= uint64(len(reviews)) {
		return []reviewsService.Review{}, nil
	}
	reviews = reviews[offset:]
	if limit > 0 && limit < uint64(len(reviews)) {
		reviews = reviews[:limit]
	}
	return reviews, nil
}

func (r *ReviewRepository) ListByEventIDs(ctx context.Context, eventIDs []string) (map[string][]reviewsService.Review, error) {
	result := make(map[string][]reviewsService.Review, len(eventIDs))
	for _, eventID := range eventIDs {
		reviews, err := r.ListByEventID(ctx, eventID, 0, 0)
		if err != nil {
			return nil, err
		}
		result[eventID] = reviews
	}
	return result, nil
}

func scanReviews(iter *gocql.Iter) ([]reviewsService.Review, error) {
	reviews := []reviewsService.Review{}
	for {
		var review reviewsService.Review
		if !iter.Scan(
			&review.ID,
			&review.EventID,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
			&review.CreatedBy,
			&review.UpdatedAt,
		) {
			break
		}
		reviews = append(reviews, review)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return reviews, nil
}
