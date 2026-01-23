#!/bin/sh
set -e

# Run migrations
echo "Running database migrations..."
./server migrate up

# Start the server
echo "Starting API server..."
exec ./server
