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

	if _, err := usersCol.Indexes().CreateOne(ctx, gomongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	if _, err := eventsCol.Indexes().CreateMany(ctx, []gomongo.IndexModel{
		{
			Keys:    bson.D{{Key: "title", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "title", Value: 1},
				{Key: "created_by", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "created_by", Value: 1}},
		},
	}); err != nil {
		return err
	}

	return nil
}
