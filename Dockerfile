FROM golang:1.24-alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o nullguard cmd/nullguard/main.go

FROM alpine:3.21

RUN apk add --no-cache wireguard-tools iptables

WORKDIR /app

COPY --from=builder /build/nullguard .
COPY --from=builder /build/banner.txt .
COPY --from=builder /build/VERSION .
COPY --from=builder /build/templates ./templates
COPY --from=builder /build/static ./static

# Expose web UI port
EXPOSE 8080

# Expose WireGuard UDP ports (range for multiple servers)
EXPOSE 51820-51830/udp

CMD ["./nullguard"]
