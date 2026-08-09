BACKEND_DIR=backend
UNIT_PACKAGES=./internal/cache ./internal/config ./internal/handlers ./internal/middleware ./internal/repository ./internal/service

test:
	cd $(BACKEND_DIR) && go test ./...

test-cover:
	cd $(BACKEND_DIR) && go test $(UNIT_PACKAGES) -coverprofile=coverage.out
	cd $(BACKEND_DIR) && go tool cover -func=coverage.out

test-integration:
	cd $(BACKEND_DIR) && go test ./internal/repository

run:
	docker compose up --build

down:
	docker compose down
