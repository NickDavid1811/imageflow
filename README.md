# ImageFlow

Servicio asíncrono de procesamiento de imágenes y su consola operativa.

## Estructura

```text
apps/api       Backend, worker y migraciones en Go
apps/console   Consola React, TypeScript y Tailwind
infra/docker   PostgreSQL y NATS JetStream para desarrollo
docs           Arquitectura y decisiones
```

## Preparación

```bash
cp .env.example .env
pnpm install
```

## Desarrollo

```bash
make infra-up
make migrate
make api
```

En terminales separadas:

```bash
make worker
make console
```

## Verificación

```bash
make test
make build
```

Consulta [docs/architecture.md](docs/architecture.md) para conocer los límites entre componentes.
