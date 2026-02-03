COMPOSE_FILE := deployments/docker-compose.yml

.PHONY: up down

up:
	docker compose -f $(COMPOSE_FILE) up -d --build

down:
	docker compose -f $(COMPOSE_FILE) down
