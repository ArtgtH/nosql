.DEFAULT_GOAL := run

COMPOSE = docker compose --env-file .env.local

.PHONY: run
run:
	$(COMPOSE) up -d --build
	@cid="$$($(COMPOSE) ps -q mongo-router-init)"; \
	while [ -z "$$cid" ]; do \
		sleep 1; \
		cid="$$($(COMPOSE) ps -q mongo-router-init)"; \
	done; \
	while [ "$$(docker inspect -f '{{.State.Status}}' $$cid)" != "exited" ]; do \
		sleep 1; \
	done; \
	test "$$(docker inspect -f '{{.State.ExitCode}}' $$cid)" = "0"
	@port="$$(awk -F= '$$1=="APP_PORT"{print $$2}' .env.local)"; \
	until curl -fsS "http://localhost:$$port/health" >/dev/null 2>&1; do \
		sleep 1; \
	done

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