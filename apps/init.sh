#!/bin/bash

# Exit on any error
set -e

echo "Starting the Smart Home Sensor API..."
echo "Building and starting containers..."
compose=(docker compose)
if ! "${compose[@]}" version > /dev/null 2>&1; then
  compose=(docker-compose)
fi
"${compose[@]}" up --build -d

echo "Waiting for services to be ready..."
# Wait for PostgreSQL to be ready
for i in {1..30}; do
  if docker exec smarthome-postgres pg_isready -U postgres -d smarthome > /dev/null 2>&1; then
    echo "PostgreSQL is ready!"
    break
  fi
  echo "Waiting for PostgreSQL to start... ($i/30)"
  sleep 1
done

# Check if PostgreSQL is ready
if ! docker exec smarthome-postgres pg_isready -U postgres -d smarthome > /dev/null 2>&1; then
  echo "Error: PostgreSQL did not start within the expected time."
  exit 1
fi

echo "All services are up and running!"
echo "The API is available at http://localhost:8080"
echo "The temperature API is available at http://localhost:8081"
echo "The device service is available at http://localhost:8082"
echo "The telemetry service is available at http://localhost:8083"
echo ""
echo "To view logs, run: ${compose[*]} logs -f"
echo "To stop the services, run: ${compose[*]} down"
