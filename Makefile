.PHONY: infra-up infra-down infra-status infra-logs migrate api worker console test build

infra-up:
	pnpm infra:up

infra-down:
	pnpm infra:down

infra-status:
	pnpm infra:status

infra-logs:
	pnpm infra:logs

migrate:
	pnpm migrate

api:
	pnpm api

worker:
	pnpm worker

console:
	pnpm console

test:
	pnpm test

build:
	pnpm build
