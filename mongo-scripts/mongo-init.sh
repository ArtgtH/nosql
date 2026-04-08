#!/bin/sh

set -eu

wait_for_mongo() {
  host="$1"
  port="$2"

  until mongosh --quiet --host "$host" --port "$port" --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; do
    sleep 2
  done
}

init_rs() {
  host="$1"
  port="$2"
  config="$3"

  mongosh --quiet --host "$host" --port "$port" --eval "try { rs.status().ok } catch (e) { rs.initiate($config) }" >/dev/null 2>&1 || true
}

wait_for_primary() {
  host="$1"
  port="$2"

  until [ "$(mongosh --quiet --host "$host" --port "$port" --eval 'try { rs.status().members.filter(m => m.stateStr === "PRIMARY").length } catch (e) { 0 }' 2>/dev/null || echo 0)" = "1" ]; do
    sleep 2
  done
}

wait_for_mongo mongo-cfg-1 27119
wait_for_mongo mongo-cfg-2 27120
wait_for_mongo mongo-cfg-3 27121
wait_for_mongo mongo-shard1-a 27217
wait_for_mongo mongo-shard1-b 27218
wait_for_mongo mongo-shard1-c 27219
wait_for_mongo mongo-shard2-a 27317
wait_for_mongo mongo-shard2-b 27318
wait_for_mongo mongo-shard2-c 27319

init_rs mongo-cfg-1 27119 '{ _id: "cfgRS", configsvr: true, members: [ { _id: 0, host: "mongo-cfg-1:27119" }, { _id: 1, host: "mongo-cfg-2:27120" }, { _id: 2, host: "mongo-cfg-3:27121" } ] }'
init_rs mongo-shard1-a 27217 '{ _id: "shard1RS", members: [ { _id: 0, host: "mongo-shard1-a:27217" }, { _id: 1, host: "mongo-shard1-b:27218" }, { _id: 2, host: "mongo-shard1-c:27219" } ] }'
init_rs mongo-shard2-a 27317 '{ _id: "shard2RS", members: [ { _id: 0, host: "mongo-shard2-a:27317" }, { _id: 1, host: "mongo-shard2-b:27318" }, { _id: 2, host: "mongo-shard2-c:27319" } ] }'

wait_for_primary mongo-cfg-1 27119
wait_for_primary mongo-shard1-a 27217
wait_for_primary mongo-shard2-a 27317
wait_for_mongo mongos 27017

mongosh --quiet --host mongos --port 27017 <<EOF2
try { sh.addShard("shard1RS/mongo-shard1-a:27217,mongo-shard1-b:27218,mongo-shard1-c:27219") } catch (e) {}
try { sh.addShard("shard2RS/mongo-shard2-a:27317,mongo-shard2-b:27318,mongo-shard2-c:27319") } catch (e) {}
try { sh.enableSharding("${MONGODB_DATABASE}") } catch (e) {}
try { sh.shardCollection("${MONGODB_DATABASE}.events", { created_by: "hashed" }) } catch (e) {}
EOF2