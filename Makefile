.PHONY: deps up down backend frontend docker-up docker-down docker-build docker-logs

deps:
	docker compose up -d postgres

up: docker-up

down: docker-down

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-logs:
	docker compose logs -f

backend:
	cd backend && go run ./cmd/server -config configs/config.yaml

frontend:
	cd frontend && npm run dev
