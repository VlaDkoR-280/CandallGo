FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./bot

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bot .
COPY .env .env
COPY loc.yaml loc.yaml
CMD ["./bot"]
