# syntax=docker/dockerfile:1
FROM golang:1.27.1-alpine AS builder
WORKDIR /src/gateway
COPY go.mod go.sum ./
COPY --from=ledger go.mod go.sum /src/ledger/
RUN go mod download
COPY . .
COPY --from=ledger . /src/ledger/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM alpine:3.24
WORKDIR /app
COPY --from=builder /out/gateway .
USER 65534:65534
ENV HTTP_ADDR=:8080
EXPOSE 8080
CMD ["/app/gateway"]
