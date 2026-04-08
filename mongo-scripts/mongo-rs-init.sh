#!/bin/sh

set -eu

mode="$1"

wait_for_local() {
  port="$1"

  until mongosh --quiet --host localhost --port "$port" --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; do
    sleep 2
  done
}

wait_for_primary_local() {
  port="$1"

  until [ "$(mongosh --quiet --host localhost --port "$port" --eval 'try { rs.status().members.filter(m => m.stateStr === "PRIMARY").length } catch (e) { 0 }' 2>/dev/null || echo 0)" = "1" ]; do
    sleep 2
  done
}

case "$mode" in
  cfg)
    wait_for_local 27119
    mongosh --quiet --host localhost --port 27119 <<'EOF'
const cfg = {
  _id: "cfgRS",
  configsvr: true,
  members: [
    { _id: 0, host: "mongo-cfg-1:27119" },
    { _id: 1, host: "mongo-cfg-2:27120" },
    { _id: 2, host: "mongo-cfg-3:27121" },
  ],
};

try {
  rs.status();
  print("cfgRS already initialized");
} catch (e) {
  rs.initiate(cfg);
}
EOF
    wait_for_primary_local 27119
    ;;

  shard1)
    wait_for_local 27217
    mongosh --quiet --host localhost --port 27217 <<'EOF'
const cfg = {
  _id: "shard1RS",
  members: [
    { _id: 0, host: "mongo-shard1-a:27217" },
    { _id: 1, host: "mongo-shard1-b:27218" },
    { _id: 2, host: "mongo-shard1-c:27219" },
  ],
};

try {
  rs.status();
  print("shard1RS already initialized");
} catch (e) {
  rs.initiate(cfg);
}
EOF
    wait_for_primary_local 27217
    ;;

  shard2)
    wait_for_local 27317
    mongosh --quiet --host localhost --port 27317 <<'EOF'
const cfg = {
  _id: "shard2RS",
  members: [
    { _id: 0, host: "mongo-shard2-a:27317" },
    { _id: 1, host: "mongo-shard2-b:27318" },
    { _id: 2, host: "mongo-shard2-c:27319" },
  ],
};

try {
  rs.status();
  print("shard2RS already initialized");
} catch (e) {
  rs.initiate(cfg);
}
EOF
    wait_for_primary_local 27317
    ;;

  *)
    echo "unknown mode: $mode" >&2
    exit 1
    ;;
esac