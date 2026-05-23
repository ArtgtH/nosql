.DEFAULT_GOAL := run

COMPOSE := docker compose --env-file .env.local

.PHONY: run
run:
	$(COMPOSE) up -d --build

.PHONY: rund
rund:
	$(COMPOSE) up --build

.PHONY: stop
stop:
	$(COMPOSE) down

.PHONY: clean
clean:
	$(COMPOSE) down -v

.PHONY: logs
logs:
	$(COMPOSE) logs -f

.PHONY: ps
ps:
	$(COMPOSE) ps

.PHONY: test
test:
	go test ./...

.PHONY: compose-config
compose-config:
	$(COMPOSE) config
