#!/bin/sh

set -eu

wait_for_local_mongo() {
  until mongosh --quiet --host localhost --port 27017 --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; do
    sleep 2
  done
}

wait_for_local_mongo

until mongosh --quiet --host localhost --port 27017 <<EOF
const rootUser = "${MONGO_INITDB_ROOT_USERNAME}";
const rootPass = "${MONGO_INITDB_ROOT_PASSWORD}";
const appDbName = "${MONGODB_DATABASE}";
const appUser = "${MONGODB_USER}";
const appPass = "${MONGODB_PASSWORD}";

const admin = db.getSiblingDB("admin");

if (!admin.getUser(rootUser)) {
  admin.createUser({
    user: rootUser,
    pwd: rootPass,
    roles: [{ role: "root", db: "admin" }],
    mechanisms: ["SCRAM-SHA-1", "SCRAM-SHA-256"],
  });
}

if (!admin.auth(rootUser, rootPass)) {
  throw new Error("failed to authenticate as root user");
}

const listShardsResult = admin.runCommand({ listShards: 1 });
const shardNames = (listShardsResult.shards || []).map((shard) => shard._id);

if (!shardNames.includes("shard1RS")) {
  sh.addShard("shard1RS/mongo-shard1-a:27217,mongo-shard1-b:27218,mongo-shard1-c:27219");
}

if (!shardNames.includes("shard2RS")) {
  sh.addShard("shard2RS/mongo-shard2-a:27317,mongo-shard2-b:27318,mongo-shard2-c:27319");
}

try {
  sh.enableSharding(appDbName);
} catch (e) {
  const msg = String(e);
  if (!msg.includes("already")) {
    throw e;
  }
}

const appDB = db.getSiblingDB(appDbName);

if (appDB.getUser(appUser)) {
  appDB.updateUser(appUser, {
    pwd: appPass,
    roles: [
      { role: "root", db: "admin" },
      { role: "dbOwner", db: appDbName },
    ],
    mechanisms: ["SCRAM-SHA-1", "SCRAM-SHA-256"],
  });
} else {
  appDB.createUser({
    user: appUser,
    pwd: appPass,
    roles: [
      { role: "root", db: "admin" },
      { role: "dbOwner", db: appDbName },
    ],
    mechanisms: ["SCRAM-SHA-1", "SCRAM-SHA-256"],
  });
}

try {
  sh.shardCollection(appDbName + ".events", { created_by: "hashed" });
} catch (e) {
  const msg = String(e);
  if (!msg.includes("already") && !msg.includes("exists")) {
    throw e;
  }
}
EOF
do
  sleep 2
done