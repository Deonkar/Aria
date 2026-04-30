.PHONY: build run seed schema-pipeline test lint

build:
	docker compose build

run:
	docker compose up

seed:
	docker compose up -d postgres
	docker compose exec -T postgres psql -U $$POSTGRES_USER -d $$POSTGRES_DB -f /docker-entrypoint-initdb.d/001_init.sql
	docker compose exec -T postgres psql -U $$POSTGRES_USER -d $$POSTGRES_DB -f /db/seed.sql

schema-pipeline:
	docker compose run --rm schema-pipeline python /app/run.py --help

test:
	docker compose build api
	docker compose run --rm api /bin/sh -lc "go test ./... -v"

lint:
	docker compose run --rm api /bin/sh -lc "go vet ./..."
