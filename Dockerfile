FROM golang:1.20 AS builder

WORKDIR /app

COPY . .

RUN go mod download
RUN go mod verify

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o telegram-openai-bot

# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser/

COPY --from=builder /app/telegram-openai-bot .

# Change to non-root user
USER appuser

CMD ["./telegram-openai-bot"]