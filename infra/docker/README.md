# Infraestructura local

El archivo `compose.yaml` administra las dependencias locales de ImageFlow:

- PostgreSQL en `localhost:5432`.
- NATS JetStream en `localhost:4222`.
- Monitor de NATS en `http://localhost:8222`.

La configuración raíz se encuentra en `.env`. PostgreSQL persiste sus archivos localmente en `infra/docker/data/postgres`; este directorio está excluido de Git. NATS conserva sus datos en un volumen administrado por Docker.

`pnpm infra:up` espera a que ambos servicios superen sus healthchecks antes de devolver el control, por lo que las migraciones pueden ejecutarse inmediatamente después.
