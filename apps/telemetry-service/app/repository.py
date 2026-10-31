from datetime import datetime
from typing import Any

import asyncpg

from .models import TelemetryRecord, TelemetryReading


class TelemetryRepository:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self.pool = pool

    async def insert(self, record: TelemetryRecord) -> TelemetryReading | None:
        async with self.pool.acquire() as connection:
            row = await connection.fetchrow(
                """
                INSERT INTO telemetry_data
                    (event_id, device_id, metric, value, unit, recorded_at)
                VALUES ($1, $2, $3, $4, $5, $6)
                ON CONFLICT (event_id) DO NOTHING
                RETURNING id, event_id, device_id, metric, value, unit,
                          recorded_at, created_at
                """,
                record.event_id,
                record.device_id,
                record.metric,
                record.value,
                record.unit,
                record.recorded_at,
            )
            if row is None:
                return None
            return self._to_reading(row)

    async def list(
        self,
        device_id: str | None,
        metric: str | None,
        from_time: datetime | None,
        to_time: datetime | None,
        limit: int,
    ) -> list[TelemetryReading]:
        conditions: list[str] = []
        values: list[Any] = []

        def add(condition: str, value: Any) -> None:
            values.append(value)
            conditions.append(condition.replace("?", f"${len(values)}"))

        if device_id:
            add("device_id = ?", device_id)
        if metric:
            add("metric = ?", metric)
        if from_time:
            add("recorded_at >= ?", from_time)
        if to_time:
            add("recorded_at <= ?", to_time)

        where = " AND ".join(conditions) or "TRUE"
        values.append(limit)
        rows = await self.pool.fetch(
            f"""
            SELECT id, event_id, device_id, metric, value, unit,
                   recorded_at, created_at
            FROM telemetry_data
            WHERE {where}
            ORDER BY recorded_at DESC, id DESC
            LIMIT ${len(values)}
            """,
            *values,
        )
        return [self._to_reading(row) for row in rows]

    async def latest(self, device_id: str, metric: str | None) -> TelemetryReading | None:
        row = await self.pool.fetchrow(
            """
            SELECT id, event_id, device_id, metric, value, unit,
                   recorded_at, created_at
            FROM telemetry_data
            WHERE device_id = $1 AND ($2 = '' OR metric = $2)
            ORDER BY recorded_at DESC, id DESC
            LIMIT 1
            """,
            device_id,
            metric or "",
        )
        return self._to_reading(row) if row else None

    @staticmethod
    def _to_reading(row: asyncpg.Record) -> TelemetryReading:
        return TelemetryReading(
            id=row["id"],
            eventId=row["event_id"],
            device_id=row["device_id"],
            metric=row["metric"],
            value=row["value"],
            unit=row["unit"],
            recorded_at=row["recorded_at"],
            created_at=row["created_at"],
        )
