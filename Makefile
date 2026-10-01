.PHONY: start-gateway start-ledger

start-gateway:
	@go -C gateway run ./cmd/gateway

start-ledger:
	@go -C ledger run ./cmd/ledger
