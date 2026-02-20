# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 3001

CMD ["./app"]