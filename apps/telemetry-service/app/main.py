import asyncio
import logging
from contextlib import asynccontextmanager
from datetime import datetime

import asyncpg
from fastapi import FastAPI, HTTPException, Query

from .config import settings
from .models import TelemetryList, TelemetryRecord, TelemetryReading
from .nats_consumer import TelemetryConsumer
from .repository import TelemetryRepository

logging.basicConfig(level=logging.INFO)
pool: asyncpg.Pool | None = None
consumer: TelemetryConsumer | None = None
consumer_task: asyncio.Task | None = None


@asynccontextmanager
async def lifespan(_: FastAPI):
    global pool, consumer, consumer_task
    pool = await asyncpg.create_pool(settings.database_url, min_size=1, max_size=10)
    consumer = TelemetryConsumer(pool)
    await consumer.start()
    consumer_task = asyncio.create_task(consumer.consume())
    yield
    if consumer_task:
        consumer_task.cancel()
        await asyncio.gather(consumer_task, return_exceptions=True)
    if consumer:
        await consumer.stop()
    await pool.close()


app = FastAPI(title="Warm House Telemetry Service", version="1.0.0", lifespan=lifespan)


def repository() -> TelemetryRepository:
    if pool is None:
        raise HTTPException(status_code=503, detail="database is not ready")
    return TelemetryRepository(pool)


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/telemetry", response_model=TelemetryReading, status_code=201)
async def record_telemetry(record: TelemetryRecord) -> TelemetryReading:
    stored = await repository().insert(record)
    if stored is None:
        latest = await repository().latest(record.device_id, record.metric)
        if latest is None:
            raise HTTPException(status_code=409, detail="duplicate telemetry event")
        return latest
    return stored


@app.get("/api/v1/telemetry", response_model=TelemetryList)
async def list_telemetry(
    device_id: str | None = None,
    metric: str | None = None,
    from_time: datetime | None = Query(default=None, alias="from"),
    to_time: datetime | None = Query(default=None, alias="to"),
    limit: int = Query(default=100, ge=1, le=1000),
) -> TelemetryList:
    items = await repository().list(device_id, metric, from_time, to_time, limit)
    return TelemetryList(items=items, total=len(items))


@app.get("/api/v1/telemetry/latest", response_model=TelemetryReading)
async def latest_telemetry(device_id: str, metric: str | None = None) -> TelemetryReading:
    reading = await repository().latest(device_id, metric)
    if reading is None:
        raise HTTPException(status_code=404, detail="reading not found")
    return reading


@app.get("/api/v1/devices/{device_id}/telemetry", response_model=TelemetryList)
async def device_telemetry(
    device_id: str,
    metric: str | None = None,
    limit: int = Query(default=100, ge=1, le=1000),
) -> TelemetryList:
    items = await repository().list(device_id, metric, None, None, limit)
    return TelemetryList(items=items, total=len(items))
