#!/usr/bin/env bash
set -ea

# Wait for the postgres server to start
RETRIES=5
until psql -c "select 1" > /dev/null 2>&1 || [ $RETRIES -eq 0 ]; do
  echo "Waiting for postgres server, $((RETRIES--)) remaining attempts..."
  sleep 1
done

# make sure the timescale tables and views scripts are run before starting up
psql < /usr/share/sparkplug/tables.sql || exit 1
psql < /usr/share/sparkplug/views.sql || exit 1

exec "$@"
