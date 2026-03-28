package mongo

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"nosql/internal/config"

	gomongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewClient(ctx context.Context, cfg config.Config) (*gomongo.Client, error) {
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d",
		url.QueryEscape(cfg.Mongo.User),
		url.QueryEscape(cfg.Mongo.Password),
		cfg.Mongo.Host,
		cfg.Mongo.Port,
	)

	client, err := gomongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return client, nil
}
