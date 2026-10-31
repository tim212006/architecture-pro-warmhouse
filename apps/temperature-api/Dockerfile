FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w" \
    -o /out/temperature-api \
    ./cmd/temperature-api

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/temperature-api /app/temperature-api

USER app

ENV PORT=8081
EXPOSE 8081

ENTRYPOINT ["/app/temperature-api"]
