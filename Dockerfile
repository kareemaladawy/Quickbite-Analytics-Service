# stage 1: Build
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o bin/api ./cmd/api

# stage 2: Run
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/bin/api ./api

# Port
EXPOSE 4100

CMD ["./api"]
