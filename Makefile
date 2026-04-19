.DEFAULT_GOAL := run

COMPOSE = docker compose --env-file .env.local

.PHONY: run
run:
	$(COMPOSE) up -d --build

.PHONY: rund
rund:
	$(COMPOSE) up --build

.PHONY: services
services:
	$(COMPOSE) ps

.PHONY: stop
stop:
	$(COMPOSE) down

.PHONY: clean
clean:
	$(COMPOSE) down -v

.PHONY: logs
logs:
	$(COMPOSE) logs -f