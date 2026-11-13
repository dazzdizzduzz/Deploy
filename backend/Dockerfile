FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server/

FROM alpine:3.19

# ca-certificates untuk TLS/HTTPS eksternal (S3, SMTP, FCM, DB SSL)
# tzdata wajib untuk zona waktu absensi (Asia/Makassar / Asia/Jakarta)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/certs ./certs
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
