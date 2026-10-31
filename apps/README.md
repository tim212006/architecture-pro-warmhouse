# Smart Home Sensor Management API

## Prerequisites

- Docker and Docker Compose

## Services

- `postgres` — PostgreSQL 16 с инициализацией баз `smarthome`, `devices` и `telemetry`.
- `temperature-api` — эмулятор датчика температуры на Go, порт `8081`.
- `device-service` — микросервис управления устройствами на Go, порт `8082`, база `devices`.
- `telemetry-service` — микросервис телеметрии на Python, порт `8083`, база `telemetry`.
- `nats` — брокер сообщений NATS JetStream, порты `4222` и `8222`, stream `WARMHOUSE`.
- `app` — монолит на Go, порт `8080`, использует `temperature-api` для актуальных показаний, публикует события в NATS и синхронизирует устройства с `device-service`.

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

1. Build and start the PostgreSQL, temperature API, device service, telemetry service, and application containers
2. Wait for the services to be ready
3. Display information about how to access the API

Alternatively, you can run Docker Compose directly:

```bash
docker compose up --build -d
```

The API will be available at http://localhost:8080, the temperature API at http://localhost:8081, device service at http://localhost:8082, and telemetry service at http://localhost:8083.

The monolith keeps its `sensors` table and gradually hands functionality over to the new services:

- `CreateSensor` and `UpdateSensor` publish `device.registered` and upsert the sensor into `device-service` as a device with `external_id` equal to the sensor ID.
- `GetSensors`, `GetSensorByID`, `UpdateSensor` and `UpdateSensorValue` synchronize the current state with `device-service`.
- Temperature readings are published as `telemetry.recorded` to NATS with `deviceId` equal to the sensor ID.
- `DELETE /api/v1/sensors/:id` also removes the device from `device-service`.
- `GET /api/v1/sensors/:id/telemetry` proxies the stored time series from `telemetry-service`.

`device-service` has a durable JetStream consumer for `device.registered`. `telemetry-service` has a durable Python consumer for `telemetry.recorded` and acknowledges a message only after PostgreSQL accepts it. `eventId` is unique in the telemetry database for deduplication.

To reset the databases and rerun the init scripts, remove the volume:

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
- `GET /api/v1/sensors/:id/telemetry` - Telemetry history for a sensor

### Temperature API

- `GET /health` - Health check
- `GET /temperature?location=Living%20Room` - Random temperature for a location
- `GET /temperature/:sensorId` - Random temperature for a sensor ID

The monolith calls `/temperature/:sensorId` when returning the sensor list, so every `Get All Sensors` request shows a new value.

### Device Service

- `GET /health` - Health check
- `GET /api/v1/devices` - List devices
- `POST /api/v1/devices` - Register or update a device by `external_id`
- `GET /api/v1/devices/:externalId` - Get a device
- `PATCH /api/v1/devices/:externalId/state` - Update status and merge reported state
- `DELETE /api/v1/devices/:externalId` - Delete a device
- `POST /api/v1/devices/:externalId/commands` - Queue a command for a device
- `GET /api/v1/devices/:externalId/commands` - List device commands

Example:

```bash
curl -s -X POST http://localhost:8082/api/v1/devices \
  -H 'Content-Type: application/json' \
  -d '{"external_id":"10","name":"Living Room Temperature","type":"temperature","location":"Living Room","unit":"°C","status":"active","state":{"value":22.4}}'

curl -s http://localhost:8082/api/v1/devices
```

### Telemetry Service

- `GET /health` - Health check
- `POST /api/v1/telemetry` - Store a reading
- `GET /api/v1/telemetry?device_id=10&metric=temperature&limit=50` - List readings
- `GET /api/v1/telemetry/latest?device_id=10&metric=temperature` - Latest reading
- `GET /api/v1/devices/:deviceId/telemetry` - Readings for a device

The REST API is used for reads and manual operations. The normal write path from the monolith is asynchronous through NATS JetStream.

Example:

```bash
curl -s -X POST http://localhost:8083/api/v1/telemetry \
  -H 'Content-Type: application/json' \
  -d '{"device_id":"10","metric":"temperature","value":22.4,"unit":"°C"}'

curl -s 'http://localhost:8083/api/v1/telemetry?device_id=10&metric=temperature&limit=10'
```
