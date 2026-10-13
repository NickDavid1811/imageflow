# Desarrollo local

## Requisitos

- Go según la versión declarada en `apps/api/go.mod`.
- Node.js y pnpm.
- Docker con Docker Compose.

## Preparación

Desde la raíz del repositorio:

```bash
pnpm bootstrap
```

## Ejecución completa

```bash
pnpm dev
```

El comando realiza esta secuencia:

```mermaid
flowchart LR
    A[Docker Compose up] --> B[Esperar healthchecks]
    B --> C[Aplicar migraciones]
    C --> D[API :8080]
    C --> E[Worker]
    C --> F[Console :5173]
    A --> G[MinIO API :9000 / Console :9001]
```

Los procesos también pueden iniciarse por separado:

```bash
pnpm infra:up
pnpm migrate
pnpm api
pnpm worker
pnpm console
```

No deben ejecutarse dos instancias locales de `pnpm dev` sobre los mismos
puertos. Para comprobarlos:

```bash
ss -ltnp | grep -E ':8080|:5173|:5432|:4222'
```

## Verificación

```bash
pnpm check
pnpm build
pnpm test:integration
```

Las pruebas de integración crean la base aislada `imageflow_test` y streams
temporales de NATS. Los tests Go permanecen junto al paquete que prueban; los
de integración usan `//go:build integration`.

## Documentación de API

Con la API en ejecución:

```text
http://localhost:8080/docs
http://localhost:8080/openapi.yaml
```

`openapi/openapi.yaml` es el contrato fuente. Al modificar un endpoint se debe
actualizar el contrato y sus pruebas en el mismo cambio.

## Datos locales

PostgreSQL escribe en `infra/docker/data/postgres` y MinIO en
`infra/docker/data/minio`; ambos quedan excluidos de Git. NATS usa el volumen
`imageflow_nats_data`. La consola de MinIO está disponible en
`http://localhost:9001` con las credenciales locales `imageflow` /
`imageflow_dev_secret`.

El almacenamiento S3-compatible es el predeterminado. Variables disponibles:

```text
IMAGEFLOW_STORAGE_DRIVER=s3
IMAGEFLOW_S3_ENDPOINT=http://localhost:9000
IMAGEFLOW_S3_REGION=us-east-1
IMAGEFLOW_S3_BUCKET=imageflow
IMAGEFLOW_S3_ACCESS_KEY=imageflow
IMAGEFLOW_S3_SECRET_KEY=imageflow_dev_secret
```

Para usar archivos locales sin MinIO, define
`IMAGEFLOW_STORAGE_DRIVER=local` y opcionalmente `IMAGEFLOW_STORAGE_PATH`.

Para detener servicios conservando los datos:

```bash
pnpm infra:down
```
