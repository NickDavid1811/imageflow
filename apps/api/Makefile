.PHONY: run-api run-worker migrate-up migrate-down test fmt vet

DATABASE_URL ?= postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

migrate-up:
	IMAGEFLOW_DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	IMAGEFLOW_DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

vet:
	go vet ./...
