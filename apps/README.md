# Smart Home Sensor Management API

## Prerequisites

- Docker and Docker Compose

## Services

- `postgres` — PostgreSQL 16 с инициализацией из `smart_home/init.sql`.
- `temperature-api` — эмулятор датчика температуры на Go, порт `8081`.
- `app` — монолит на Go, порт `8080`, использует `temperature-api` для актуальных показаний.

`temperature-api` разделён по слоям чистой архитектуры:

- `cmd/temperature-api` — точка входа и сборка зависимостей.
- `internal/domain` — сущность показания и каталог комнат.
- `internal/usecase` — сценарий получения случайного показания.
- `internal/adapter` — HTTP-обработчики, генератор случайных значений, системные часы.
- `internal/config` — конфигурация из переменных окружения.

## Getting Started

### Option 1: Using Docker Compose (Recommended)

The easiest way to start the application is to use Docker Compose:

```bash
./init.sh
```

This script will:

1. Build and start the PostgreSQL, temperature API, and application containers
2. Wait for the services to be ready
3. Display information about how to access the API

Alternatively, you can run Docker Compose directly:

```bash
docker compose up --build -d
```

The API will be available at http://localhost:8080, the temperature API at http://localhost:8081.

To reset the database and rerun `init.sql`, remove the volume:

```bash
docker compose down -v
```

### Option 2: Manual setup

If you prefer to run the application without Docker:

1. Start the PostgreSQL database:

```bash
docker compose up -d postgres
```

2. Build and run the application:

```bash
go build -o smarthome
./smarthome
```

## API Testing

A Postman collection is provided for testing the API. Import the `smarthome-api.postman_collection.json` file into Postman to get started.

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/sensors` - Get all sensors
- `GET /api/v1/sensors/:id` - Get a specific sensor
- `POST /api/v1/sensors` - Create a new sensor
- `PUT /api/v1/sensors/:id` - Update a sensor
- `DELETE /api/v1/sensors/:id` - Delete a sensor
- `PATCH /api/v1/sensors/:id/value` - Update a sensor's value and status

### Temperature API

- `GET /health` - Health check
- `GET /temperature?location=Living%20Room` - Random temperature for a location
- `GET /temperature/:sensorId` - Random temperature for a sensor ID

The monolith calls `/temperature/:sensorId` when returning the sensor list, so every `Get All Sensors` request shows a new value.
