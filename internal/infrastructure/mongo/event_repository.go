package mongo

import (
	"context"
	"errors"
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
		return "", errors.New("unexpected inserted id type")
	}

	return id.Hex(), nil
}

func (r *EventRepository) ExistsByTitle(ctx context.Context, title string) (bool, error) {
	count, err := r.col.CountDocuments(ctx, bson.M{"title": title}, options.Count().SetLimit(1))
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (eventsService.Event, bool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return eventsService.Event{}, false, nil
	}

	var event eventsService.Event
	err = r.col.FindOne(ctx, bson.M{"_id": objectID}).Decode(&event)
	if errors.Is(err, gomongo.ErrNoDocuments) {
		return eventsService.Event{}, false, nil
	}
	if err != nil {
		return eventsService.Event{}, false, err
	}

	return event, true, nil
}

func (r *EventRepository) UpdateByOrganizer(ctx context.Context, eventID, organizerID string, patch eventsService.EventPatch) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return false, nil
	}

	setFields := bson.M{}
	unsetFields := bson.M{}

	if patch.Category != nil {
		setFields["category"] = *patch.Category
	}
	if patch.Price != nil {
		setFields["price"] = *patch.Price
	}
	if patch.City != nil {
		if *patch.City == "" {
			unsetFields["location.city"] = ""
		} else {
			setFields["location.city"] = *patch.City
		}
	}

	update := bson.M{}
	if len(setFields) > 0 {
		update["$set"] = setFields
	}
	if len(unsetFields) > 0 {
		update["$unset"] = unsetFields
	}

	if len(update) == 0 {
		count, err := r.col.CountDocuments(ctx, bson.M{
			"_id":        objectID,
			"created_by": organizerID,
		}, options.Count().SetLimit(1))
		if err != nil {
			return false, err
		}
		return count > 0, nil
	}

	res, err := r.col.UpdateOne(ctx, bson.M{
		"_id":        objectID,
		"created_by": organizerID,
	}, update)
	if err != nil {
		return false, err
	}

	return res.MatchedCount > 0, nil
}

func (r *EventRepository) List(ctx context.Context, filter eventsService.ListFilter) ([]eventsService.Event, error) {
	mongoFilter := bson.M{}

	if filter.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(filter.ID)
		if err != nil {
			return []eventsService.Event{}, nil
		}
		mongoFilter["_id"] = objectID
	}
	if filter.Title != "" {
		mongoFilter["title"] = bson.M{
			"$regex": regexp.QuoteMeta(filter.Title),
		}
	}
	if filter.Category != "" {
		mongoFilter["category"] = filter.Category
	}
	if filter.City != "" {
		mongoFilter["location.city"] = filter.City
	}
	if filter.CreatedBy != "" {
		mongoFilter["created_by"] = filter.CreatedBy
	}
	if filter.PriceFrom != nil || filter.PriceTo != nil {
		priceFilter := bson.M{}
		if filter.PriceFrom != nil {
			priceFilter["$gte"] = *filter.PriceFrom
		}
		if filter.PriceTo != nil {
			priceFilter["$lte"] = *filter.PriceTo
		}
		mongoFilter["price"] = priceFilter
	}
	if filter.DateFrom != "" || filter.DateToExclusive != "" {
		startedAtFilter := bson.M{}
		if filter.DateFrom != "" {
			startedAtFilter["$gte"] = filter.DateFrom
		}
		if filter.DateToExclusive != "" {
			startedAtFilter["$lt"] = filter.DateToExclusive
		}
		mongoFilter["started_at"] = startedAtFilter
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

func (r *EventRepository) ListByTitles(ctx context.Context, titles []string) ([]eventsService.Event, error) {
	if len(titles) == 0 {
		return []eventsService.Event{}, nil
	}

	cursor, err := r.col.Find(ctx, bson.M{
		"title": bson.M{"$in": titles},
	})
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

func (r *EventRepository) ListByIDs(ctx context.Context, ids []string) ([]eventsService.Event, error) {
	if len(ids) == 0 {
		return []eventsService.Event{}, nil
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}
	if len(objectIDs) == 0 {
		return []eventsService.Event{}, nil
	}

	cursor, err := r.col.Find(ctx, bson.M{
		"_id": bson.M{"$in": objectIDs},
	})
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
