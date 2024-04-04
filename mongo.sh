#!/bin/dash
set -eu

usage() {
  echo "$0 restart|start|stop"
  exit 64
}

failure_hints() {
  echo '
  Failed to start mongo. If using Docker Desktop:
    - Go into Settings -> Features in development
      - untick "Use containerd"
      - tick "Use Rosetta"'
  exit 1
}

docker_run() {
  docker run --name router-mongo -dp 27017:27017 ghcr.io/alphagov/govuk-infrastructure/mongodb:016364bbefd79123a59be4ad2ce0f338688dc16f --replSet rs0
}

init_replicaset() {
  docker exec router-mongo mongo --eval 'rs.initiate();'
}

healthy() {
  docker exec router-mongo mongo --eval \
    'print(rs.status());'
}

# usage: retry_or_fatal description command-to-try
retry_or_fatal() {
  n=20
  echo -n "Waiting up to $n s for $1"; shift
  while [ "$n" -ge 0 ]; do
    sleep 1 && echo -n .
    n=$((n-1))
  done
}

stop() {
  if ! docker stop router-mongo >/dev/null 2>&1; then
    echo "router-mongo not running"
    return
  fi
  echo -n Waiting for router-mongo container to exit.
  docker wait router-mongo >/dev/null || true
  docker rm -f router-mongo >/dev/null 2>&1 || true
  echo " done"
}

start() {
  if healthy; then
    echo router-mongo already running.
    return
  fi
  stop
  docker_run || failure_hints
  retry_or_fatal "for successful rs.initiate()" init_replicaset
  retry_or_fatal "for healthy rs.status()" healthy
}

case $1 in
  start) $1;;
  stop) $1;;
  *) usage
esac
