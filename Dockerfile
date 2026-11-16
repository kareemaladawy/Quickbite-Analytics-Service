# stage 1: Build
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o bin/api ./cmd/api
RUN go build -o bin/backfill-aggs ./cmd/backfill-aggs

# stage 2: Run
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/bin/api ./api
COPY --from=builder /app/bin/backfill-aggs ./backfill-aggs

# Port
EXPOSE 4100

CMD ["./api"]
