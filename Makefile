COMPOSE_FILE := deployments/docker-compose.yml

-include .env
export

.PHONY: up down migrate-schema-up migrate-schema-down migrate-data-up migrate-data-down migrate-status migrate-data-status generate

up:
	docker compose -f $(COMPOSE_FILE) up -d --build

down:
	docker compose -f $(COMPOSE_FILE) down

migrate-schema-up:
	$(GOOSE_BIN) -dir $(GOOSE_SCHEMA_DIR) -table $(GOOSE_SCHEMA_TABLE) mysql "$(DB_DSN)" up

migrate-schema-down:
	$(GOOSE_BIN) -dir $(GOOSE_SCHEMA_DIR) -table $(GOOSE_SCHEMA_TABLE) mysql "$(DB_DSN)" down

migrate-data-up:
	$(GOOSE_BIN) -dir $(GOOSE_DATA_DIR) -table $(GOOSE_DATA_TABLE) mysql "$(DB_DSN)" up

migrate-data-down:
	$(GOOSE_BIN) -dir $(GOOSE_DATA_DIR) -table $(GOOSE_DATA_TABLE) mysql "$(DB_DSN)" down

migrate-status:
	$(GOOSE_BIN) -dir $(GOOSE_SCHEMA_DIR) -table $(GOOSE_SCHEMA_TABLE) mysql "$(DB_DSN)" status

migrate-data-status:
	$(GOOSE_BIN) -dir $(GOOSE_DATA_DIR) -table $(GOOSE_DATA_TABLE) mysql "$(DB_DSN)" status

generate:
	$(OAPI_CODEGEN_BIN) -config $(OAPI_CODEGEN_CONFIG) $(OAPI_SPEC)
