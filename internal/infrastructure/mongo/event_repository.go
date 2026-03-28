package mongo

import (
	"context"
	"regexp"

	eventsService "nosql/internal/service/events"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	gomongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EventRepository struct {
	col *gomongo.Collection
}

func NewEventRepository(db *gomongo.Database) *EventRepository {
	return &EventRepository{
		col: db.Collection("events"),
	}
}

func (r *EventRepository) Create(ctx context.Context, event eventsService.Event) (string, error) {
	res, err := r.col.InsertOne(ctx, event)
	if err != nil {
		if gomongo.IsDuplicateKeyError(err) {
			return "", eventsService.ErrEventExists
		}
		return "", err
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", gomongo.ErrClientDisconnected
	}

	return id.Hex(), nil
}

func (r *EventRepository) List(ctx context.Context, filter eventsService.ListFilter) ([]eventsService.Event, error) {
	mongoFilter := bson.M{}

	if filter.Title != "" {
		mongoFilter["title"] = bson.M{
			"$regex": regexp.QuoteMeta(filter.Title),
		}
	}

	opts := options.Find()

	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	cursor, err := r.col.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []eventsService.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	return events, nil
}
