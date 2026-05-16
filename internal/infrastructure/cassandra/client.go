package cassandra

import (
	"context"
	"strings"
	"time"

	"nosql/internal/config"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

func NewSession(ctx context.Context, cfg config.Config) (*gocql.Session, error) {
	cluster, err := newClusterConfig(cfg.Cassandra, cfg.Cassandra.Keyspace)
	if err != nil {
		return nil, err
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	if err := session.Query("SELECT release_version FROM system.local").WithContext(ctx).Exec(); err != nil {
		session.Close()
		return nil, err
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
