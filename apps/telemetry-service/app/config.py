import os


class Settings:
    def __init__(self) -> None:
        self.database_url = os.getenv(
            "DATABASE_URL",
            "postgresql://postgres:postgres@localhost:5432/telemetry",
        )
        self.nats_url = os.getenv("NATS_URL", "nats://localhost:4222")
        self.nats_subject = os.getenv("NATS_TELEMETRY_SUBJECT", "telemetry.recorded")
        self.nats_stream = os.getenv("NATS_STREAM", "WARMHOUSE")
        self.nats_consumer = os.getenv("NATS_CONSUMER", "telemetry-service")


settings = Settings()
