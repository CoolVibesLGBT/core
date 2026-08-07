# Build stage
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /app/core .

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates ffmpeg

WORKDIR /app

COPY --from=builder /app/core ./core
COPY --from=builder /app/static ./static
COPY --from=builder /app/views ./views
COPY --from=builder /app/seeders/places/places.json ./seeders/places/places.json
RUN mkdir -p ./static/uploads

EXPOSE 3001 3002

CMD ["sh", "-c", "./core -migrate && exec ./core"]
