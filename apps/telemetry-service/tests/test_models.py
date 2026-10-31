from datetime import datetime, timezone

from app.models import TelemetryRecord


def test_telemetry_record_accepts_event_contract() -> None:
    record = TelemetryRecord(
        eventId="event-1",
        device_id="1",
        metric="temperature",
        value=22.5,
        unit="celsius",
        recorded_at=datetime.now(timezone.utc),
    )

    assert record.event_id == "event-1"
    assert record.device_id == "1"
