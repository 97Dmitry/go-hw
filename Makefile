start-gateway:
	@go -C gateway run ./cmd/server

start-ledger:
	@go -C ledger run ./cmd/server