# ImageFlow

Servicio asíncrono de procesamiento de imágenes y su consola operativa.

## Estructura

```text
apps/api       Backend, worker y migraciones en Go
apps/console   Consola React, TypeScript y Tailwind
infra/docker   PostgreSQL, NATS JetStream y MinIO para desarrollo
docs           Arquitectura y decisiones
```

## Preparación

```bash
pnpm bootstrap
```

La configuración predeterminada funciona sin archivos adicionales. Las variables locales opcionales se configuran en `.env`, que nunca se versiona.

## Desarrollo

```bash
pnpm dev
```

Este comando levanta PostgreSQL, NATS y MinIO, aplica migraciones e inicia API, worker y consola en paralelo. También pueden iniciarse individualmente:

```bash
pnpm infra:up
pnpm migrate
pnpm api
pnpm worker
pnpm console
```

## Verificación

```bash
pnpm check
pnpm build
```

Para detener la infraestructura:

```bash
pnpm infra:down
```

Consulta [docs/architecture.md](docs/architecture.md) para conocer los límites entre componentes.
Para retomar el trabajo en una nueva sesión, comienza por [docs/project-status.md](docs/project-status.md).

Documentación adicional:

- [Desarrollo local](docs/development.md)
- [Operación del backend](docs/operations.md)
- [Contrato OpenAPI](apps/api/openapi/openapi.yaml)
