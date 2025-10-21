FROM golang:1.24-alpine AS builder
LABEL authors="MaximBraer"

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd/subscription
RUN go build -o migrator ./cmd/migrator

FROM alpine:latest
RUN apk --no-cache add ca-certificates netcat-openbsd
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/migrator .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["./docker-entrypoint.sh"]