# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o por-backend cmd/server/main.go

# Run stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/por-backend ./por-backend
COPY ./config ./config
COPY ./data ./data
COPY .env .env
EXPOSE 8082
CMD ["./por-backend"] 