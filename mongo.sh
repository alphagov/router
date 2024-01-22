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
  docker run --name router-mongo -dp 27017:27017 mongo:2.6 --replSet rs0 --quiet
}

init_replicaset() {
  docker exec router-mongo mongo --quiet --eval 'rs.initiate();' >/dev/null 2>&1
}

healthy() {
  docker exec router-mongo mongo --quiet --eval \
    'if (rs.status().members[0].health==1) print("healthy");' \
    2>&1 | grep healthy >/dev/null
}

# usage: retry_or_fatal description command-to-try
retry_or_fatal() {
  n=10
  echo -n "Waiting up to $n s for $1"; shift
  while [ "$n" -ge 0 ]; do
    if "$@"; then
      echo " done"
      return
    fi
    sleep 1 && echo -n .
    n=$((n-1))
  done
  echo "gave up"
  exit 1
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
