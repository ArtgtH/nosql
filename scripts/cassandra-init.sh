#!/bin/sh
set -eu

host="${CASSANDRA_INIT_HOST:-}"
if [ -z "$host" ]; then
  host="$(printf '%s' "$CASSANDRA_HOSTS" | cut -d ',' -f 1 | tr -d '[:space:]')"
fi

port="${CASSANDRA_PORT:-9042}"
keyspace="${CASSANDRA_KEYSPACE:?CASSANDRA_KEYSPACE is required}"

if ! printf '%s' "$keyspace" | grep -Eq '^[A-Za-z_][A-Za-z0-9_]*$'; then
  echo "Invalid CASSANDRA_KEYSPACE=$keyspace" >&2
  exit 1
fi

auth_args=""
if [ -n "${CASSANDRA_USERNAME:-}" ]; then
  auth_args="-u $CASSANDRA_USERNAME -p $CASSANDRA_PASSWORD"
fi

until cqlsh "$host" "$port" $auth_args -e "DESCRIBE KEYSPACES" >/dev/null 2>&1; do
  sleep 2
done

schema_file="/tmp/cassandra-schema.cql"
cat > "$schema_file" <<EOF
CREATE KEYSPACE IF NOT EXISTS $keyspace
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS $keyspace.event_reactions (
  event_id text,
  created_by text,
  like_value tinyint,
  created_at timestamp,
  PRIMARY KEY ((event_id), created_by)
);

CREATE INDEX IF NOT EXISTS event_reactions_like_value_idx
ON $keyspace.event_reactions (like_value);

CREATE INDEX IF NOT EXISTS event_reactions_created_by_idx
ON $keyspace.event_reactions (created_by);
EOF

cqlsh "$host" "$port" $auth_args -f "$schema_file"
echo "Cassandra schema initialized"
