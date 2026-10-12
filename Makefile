.PHONY: infra-up infra-down infra-status infra-logs migrate api worker console test build

COMPOSE := docker compose --env-file .env -f infra/docker/compose.yaml

infra-up:
	$(COMPOSE) up -d

infra-down:
	$(COMPOSE) down

infra-status:
	$(COMPOSE) ps

infra-logs:
	$(COMPOSE) logs -f

migrate:
	$(MAKE) -C apps/api migrate-up

api:
	$(MAKE) -C apps/api run-api

worker:
	$(MAKE) -C apps/api run-worker

console:
	pnpm --filter @imageflow/console dev

test:
	cd apps/api && go test ./...
	pnpm --filter @imageflow/console typecheck

build:
	pnpm --filter @imageflow/console build
