FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest
# ca-certificates for HTTPS calls to api-football.com; tzdata for time zones
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /build/server .
USER app
EXPOSE 8080
CMD ["/app/server"]
