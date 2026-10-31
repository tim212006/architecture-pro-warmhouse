from datetime import datetime
from typing import Any

from pydantic import AliasChoices, BaseModel, ConfigDict, Field


class TelemetryRecord(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    event_id: str | None = Field(default=None, alias="eventId", validation_alias=AliasChoices("eventId", "event_id"))
    device_id: str = Field(alias="device_id", validation_alias=AliasChoices("deviceId", "device_id"))
    metric: str
    value: float
    unit: str = ""
    recorded_at: datetime = Field(alias="recorded_at", validation_alias=AliasChoices("recordedAt", "recorded_at"))


class TelemetryReading(TelemetryRecord):
    id: int
    created_at: datetime


class TelemetryList(BaseModel):
    items: list[TelemetryReading]
    total: int
