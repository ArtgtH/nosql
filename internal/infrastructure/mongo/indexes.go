package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	gomongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureIndexes(ctx context.Context, db *gomongo.Database) error {
	usersCol := db.Collection("users")
	eventsCol := db.Collection("events")

	if _, err := usersCol.Indexes().CreateMany(ctx, []gomongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "full_name", Value: 1}},
		},
	}); err != nil {
		return err
	}

	if _, err := eventsCol.Indexes().CreateMany(ctx, []gomongo.IndexModel{
		{
			Keys: bson.D{{Key: "title", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "title", Value: 1}, {Key: "created_by", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_by", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "price", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "location.city", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "started_at", Value: 1}},
		},
	}); err != nil {
		return err
	}

	return nil
}
