package cassandra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nosql/internal/config"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

const eventReactionsTable = "event_reactions"

func NewSession(ctx context.Context, cfg config.Config) (*gocql.Session, error) {
	adminCluster, err := newClusterConfig(cfg.Cassandra, "")
	if err != nil {
		return nil, err
	}

	adminSession, err := adminCluster.CreateSession()
	if err != nil {
		return nil, err
	}
	defer adminSession.Close()

	keyspaceQuery := fmt.Sprintf(
		"CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
		cfg.Cassandra.Keyspace,
	)
	if err := adminSession.Query(keyspaceQuery).WithContext(ctx).Exec(); err != nil {
		return nil, err
	}

	cluster, err := newClusterConfig(cfg.Cassandra, cfg.Cassandra.Keyspace)
	if err != nil {
		return nil, err
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	queries := []string{
		fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s (
	event_id text,
	created_by text,
	like_value tinyint,
	created_at timestamp,
	PRIMARY KEY ((event_id), created_by)
)`,
			cfg.Cassandra.Keyspace,
			eventReactionsTable,
		),
		fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s_like_value_idx ON %s.%s (like_value)",
			eventReactionsTable,
			cfg.Cassandra.Keyspace,
			eventReactionsTable,
		),
		fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s_created_by_idx ON %s.%s (created_by)",
			eventReactionsTable,
			cfg.Cassandra.Keyspace,
			eventReactionsTable,
		),
	}

	for _, query := range queries {
		if err := session.Query(query).WithContext(ctx).Exec(); err != nil {
			session.Close()
			return nil, err
		}
	}

	return session, nil
}

func newClusterConfig(cfg config.CassandraConfig, keyspace string) (*gocql.ClusterConfig, error) {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Port = cfg.Port
	cluster.Consistency = parseConsistency(cfg.Consistency)
	cluster.ConnectTimeout = 10 * time.Second
	cluster.Timeout = 10 * time.Second
	cluster.NumConns = 1
	cluster.ProtoVersion = 4
	cluster.DisableInitialHostLookup = true

	if keyspace != "" {
		cluster.Keyspace = keyspace
	}

	if cfg.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	return cluster, nil
}

func parseConsistency(value string) gocql.Consistency {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ANY":
		return gocql.Any
	case "ONE":
		return gocql.One
	case "TWO":
		return gocql.Two
	case "THREE":
		return gocql.Three
	case "QUORUM":
		return gocql.Quorum
	case "ALL":
		return gocql.All
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum
	case "EACH_QUORUM":
		return gocql.EachQuorum
	case "LOCAL_ONE":
		return gocql.LocalOne
	default:
		return gocql.One
	}
}
