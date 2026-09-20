.PHONY: run build test tidy fmt vet db-up db-down db-reset psql clean

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./... -race -cover

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

# Sobe o Postgres local e espera ficar saudável.
db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

# Apaga o volume: todos os dados locais são perdidos.
db-reset:
	docker compose down -v && docker compose up -d --wait postgres

psql:
	docker compose exec postgres psql -U financas -d financas

clean:
	rm -rf bin
