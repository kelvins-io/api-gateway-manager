.PHONY: deps up down backend frontend

deps:
	docker compose up -d

up: deps
	@echo "PostgreSQL + Kong ready. Start backend and frontend separately."

down:
	docker compose down

backend:
	cd backend && go run ./cmd/server -config configs/config.yaml

frontend:
	cd frontend && npm run dev
