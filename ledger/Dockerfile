FROM golang:1.27.1-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ledger ./cmd/ledger

FROM alpine:3.24
WORKDIR /app
COPY --from=builder /out/ledger .
USER 65534:65534
CMD ["/app/ledger"]
