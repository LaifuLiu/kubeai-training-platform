# Stage 1: Build the binary
FROM golang:1.27-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# Stage 2: Create a minimal runtime image
FROM alpine:3.20

RUN apk --no-cache add ca-certificates \
  && addgroup -S app \
  && adduser -S -G app app

WORKDIR /app
COPY --from=builder /app/server /app/server

USER app

EXPOSE 8080
CMD ["/app/server"]