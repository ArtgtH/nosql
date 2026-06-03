#!/bin/sh
set -eu

url="${NEO4J_URL:?NEO4J_URL is required}"
user="${NEO4J_USERNAME:?NEO4J_USERNAME is required}"
password="${NEO4J_PASSWORD:?NEO4J_PASSWORD is required}"

until cypher-shell -a "$url" -u "$user" -p "$password" "RETURN 1" >/dev/null 2>&1; do
  sleep 2
done

cypher-shell -a "$url" -u "$user" -p "$password" <<'CYPHER'
CREATE CONSTRAINT user_id IF NOT EXISTS
FOR (u:User)
REQUIRE u.id IS UNIQUE;

CREATE CONSTRAINT event_id IF NOT EXISTS
FOR (e:Event)
REQUIRE e.id IS UNIQUE;
CYPHER

echo "Neo4j schema initialized"
