#!/bin/bash

set -e

__workdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
__rootdir=$(dirname "${__workdir}")

cd "${__rootdir}"

while ! docker-compose exec -T source mysql --user=root --password=root_pwd -e "status" &>/dev/stderr; do
  echo "Waiting for source MySQL connection..."
  sleep 1
done

while ! docker-compose exec -T upstream mysql --user=root --password=root_pwd -e "status" &>/dev/stderr; do
  echo "Waiting for upstream MySQL connection..."
  sleep 1
done
