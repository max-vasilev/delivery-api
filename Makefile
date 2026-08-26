-include .env
export

run:
	go run ./cmd/server/
migrate-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL пуст. Скопируйте .env.example в .env"; exit 1; fi
	@goose -dir migrations postgres "$(DATABASE_URL)" up
migrate-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL пуст. Скопируйте .env.example в .env"; exit 1; fi
	@goose -dir migrations postgres "$(DATABASE_URL)" down
db-up:
	docker compose up -d
db-down:
	docker compose down
db-shell:
	@docker compose exec db psql -U delivery_user -d delivery
fmt:
	go fmt ./...
lint:
	golangci-lint run
vet:
	go vet ./...
check: fmt vet lint
