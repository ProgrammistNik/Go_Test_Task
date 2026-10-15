.PHONY: run test test-race fmt vet build up down logs

run:
	go run ./cmd/app

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

build:
	go build ./cmd/app

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f app
