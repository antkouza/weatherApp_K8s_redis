# STAGE 1: Build
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o weather-server .

# STAGE 2: Run
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/weather-server .
EXPOSE 8080
CMD ["./weather-server"]