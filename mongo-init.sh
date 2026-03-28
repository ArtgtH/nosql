#!/bin/sh
set -eu

mongosh --host localhost \
  --authenticationDatabase admin \
  -u "$MONGO_INITDB_ROOT_USERNAME" \
  -p "$MONGO_INITDB_ROOT_PASSWORD" <<EOF
use $MONGODB_DATABASE

db.createUser({
  user: "$MONGODB_USER",
  pwd: "$MONGODB_PASSWORD",
  roles: [
    { role: "readWrite", db: "$MONGODB_DATABASE" }
  ]
})
EOF