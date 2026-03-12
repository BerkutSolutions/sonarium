BINARY_NAME=music-server

.PHONY: build run docker-build docker-run migrate-up migrate-down scan

build:
	go build -o bin/$(BINARY_NAME) ./cmd/server

run:
	go run ./cmd/server

docker-build:
	docker compose build

docker-run:
	docker compose up --build

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

scan:
	docker compose run --rm music-server ./scanner
