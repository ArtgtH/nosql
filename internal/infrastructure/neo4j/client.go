package neo4j

import (
	"context"

	"nosql/internal/config"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func NewDriver(ctx context.Context, cfg config.Neo4jConfig) (neo4jdriver.DriverWithContext, error) {
	auth := neo4jdriver.NoAuth()
	if cfg.User != "" || cfg.Password != "" {
		auth = neo4jdriver.BasicAuth(cfg.User, cfg.Password, "")
	}

	driver, err := neo4jdriver.NewDriverWithContext(
		cfg.URL,
		auth,
	)
	if err != nil {
		return nil, err
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, err
	}
	return driver, nil
}
