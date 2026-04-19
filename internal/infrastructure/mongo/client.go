package mongo

import (
	"context"
	"fmt"
	"net/url"

	"nosql/internal/config"

	gomongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewClient(ctx context.Context, cfg config.Config) (*gomongo.Client, error) {
	dbName := url.QueryEscape(cfg.Mongo.Database)

	var uri string
	if cfg.Mongo.User != "" && cfg.Mongo.Password != "" {
		uri = fmt.Sprintf(
			"mongodb://%s:%s@%s:%d/%s?authSource=%s",
			url.QueryEscape(cfg.Mongo.User),
			url.QueryEscape(cfg.Mongo.Password),
			cfg.Mongo.Host,
			cfg.Mongo.Port,
			dbName,
			dbName,
		)
	} else {
		uri = fmt.Sprintf(
			"mongodb://%s:%d/%s",
			cfg.Mongo.Host,
			cfg.Mongo.Port,
			dbName,
		)
	}

	return gomongo.Connect(ctx, options.Client().ApplyURI(uri))
}
