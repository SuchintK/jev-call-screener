.PHONY: run test vet build docker-build

run:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -o bin/jev-call-screener ./cmd/server

docker-build:
	docker build -t jev-call-screener .
