FROM golang:1.15.6 AS builder
WORKDIR /builder
COPY ./ /builder

RUN apt-get update && apt-get install patch

WORKDIR /builder
RUN go build -o ./dist/bin/service-discovery -i ./cmd/service-discovery/main.go

FROM debian:bullseye-slim

WORKDIR /app
COPY --from=builder /builder/dist /app/