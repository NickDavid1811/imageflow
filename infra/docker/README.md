# Infraestructura local

El archivo `compose.yaml` administra las dependencias locales de ImageFlow:

- PostgreSQL en `localhost:5432`.
- NATS JetStream en `localhost:4222`.
- Monitor de NATS en `http://localhost:8222`.

La configuración raíz se encuentra en `.env`. El volumen externo de PostgreSQL mantiene el nombre usado por el Compose anterior para preservar los datos existentes. `pnpm infra:up` lo reutiliza o lo crea automáticamente si todavía no existe.
