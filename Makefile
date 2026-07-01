UNIT_PACKAGES=./internal/cache ./internal/config ./internal/handlers ./internal/middleware ./internal/repository ./internal/service

test:
	go test ./...

test-cover:
	go test $(UNIT_PACKAGES) -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-integration:
	go test ./internal/repository

run:
	docker compose up --build

down:
	docker compose down