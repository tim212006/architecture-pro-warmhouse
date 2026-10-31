import json
import logging

import asyncpg
import nats
from nats.js import JetStreamContext

from .config import settings
from .models import TelemetryRecord
from .repository import TelemetryRepository

logger = logging.getLogger(__name__)


class TelemetryConsumer:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self.pool = pool
        self.connection: nats.NATS | None = None
        self.js: JetStreamContext | None = None
        self.subscription = None

    async def start(self) -> None:
        self.connection = await nats.connect(settings.nats_url)
        self.js = self.connection.jetstream()
        try:
            await self.js.add_stream(
                name=settings.nats_stream,
                subjects=[settings.nats_subject],
            )
        except Exception:
            await self.js.update_stream(
                name=settings.nats_stream,
                subjects=[settings.nats_subject],
            )
        self.subscription = await self.js.pull_subscribe(
            settings.nats_subject,
            durable=settings.nats_consumer,
            stream=settings.nats_stream,
        )
        logger.info("subscribed to %s", settings.nats_subject)

    async def consume(self) -> None:
        if self.subscription is None:
            raise RuntimeError("NATS consumer is not started")

        repository = TelemetryRepository(self.pool)
        while True:
            try:
                messages = await self.subscription.fetch(10, timeout=1)
            except TimeoutError:
                continue
            for message in messages:
                try:
                    payload = json.loads(message.data)
                    record = TelemetryRecord.model_validate(payload)
                    await repository.insert(record)
                    await message.ack()
                except Exception:
                    logger.exception("failed to process telemetry event")
                    await message.nak(delay=2)

    async def stop(self) -> None:
        if self.connection is not None:
            await self.connection.drain()
