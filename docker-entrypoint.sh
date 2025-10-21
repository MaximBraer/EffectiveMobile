#!/bin/sh
set -e

echo "Waiting for PostgreSQL to be ready..."
until nc -z postgres 5432; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 1
done

echo "PostgreSQL is up - running migrations..."
./migrator -dsn "postgres://postgres:postgres@postgres:5432/subscriptions?sslmode=disable" -migrations-path ./migrations

echo "Migrations completed - starting application..."
exec ./main

