FROM golang:latest AS builder
WORKDIR /app

COPY . .

RUN go mod download

WORKDIR /app/server/cmd

RUN go build -o /app/cmd main.go

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/cmd /app/
COPY --from=builder /app/server/cmd/.env /app/

EXPOSE 8080

CMD ["./cmd"]
