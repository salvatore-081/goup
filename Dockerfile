FROM golang:1.19.1-bullseye AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o /goup

FROM debian:bullseye-slim

COPY --from=builder /goup /app/goup

WORKDIR /app

COPY ./entrypoint.sh .

ENTRYPOINT ["/app/entrypoint.sh"]