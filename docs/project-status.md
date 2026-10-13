# Estado actual de ImageFlow

> Documento vivo de handoff. Última actualización: 12 de octubre de 2026.
>
> Al retomar el proyecto en otra sesión, leer primero este archivo, después
> [`architecture.md`](architecture.md) y los ADR de [`adr/`](adr/).

## Objetivo del producto

ImageFlow busca convertirse en una API integrable para procesamiento asíncrono
de imágenes. Otros productos podrán enviar una imagen y una secuencia de
operaciones, consultar el estado del trabajo y obtener el archivo resultante.

El proyecto está en una primera etapa funcional. La infraestructura asíncrona,
la persistencia, la API y la consola ya trabajan de extremo a extremo. El
worker ya procesa imágenes JPEG/PNG y publica el resultado mediante la API.

## Estado funcional

Actualmente funciona este flujo:

```text
React Console
    |
    | POST /v1/jobs
    v
Go API
    |
    | transacción: job + outbox event
    v
PostgreSQL
    |
    | outbox publisher
    v
NATS JetStream
    |
    | durable consumer
    v
Go Worker
    |
    | descarga segura -> resize/conversión -> almacenamiento local
    v
PostgreSQL -> polling de la consola -> GET /v1/jobs/{id}/result
```

La consola permite indicar:

- URL de origen.
- Ancho y alto de salida.
- Formato de salida.
- Calidad de compresión.

La interfaz ofrece tema claro y oscuro persistente, consulta periódica del
trabajo y visualización del payload. La familia tipográfica prioriza Google
Sans cuando está instalada y utiliza fallbacks del sistema en caso contrario.

## Qué está implementado

### Backend

- Go con arquitectura hexagonal pragmática.
- Binarios independientes para API, worker y migraciones.
- Dominio y casos de uso sin dependencia directa de HTTP, PostgreSQL o NATS.
- Persistencia con `pgx` y migraciones SQL embebidas.
- Transactional Outbox para crear el trabajo y su evento atómicamente.
- Publicador de outbox con bloqueo, lease, backoff y deduplicación en NATS.
- Consumidor durable de JetStream.
- Reclamo condicional de trabajos e idempotencia por estado.
- Reintentos espaciados y límite configurable a nivel de aplicación.
- Dead-letter stream `IMAGEFLOW_JOBS_DLQ` para entregas agotadas.
- Liveness, readiness y logs estructurados.
- Cierre ordenado de API y worker.
- Contrato OpenAPI 3.1 embebido y referencia interactiva con Scalar.
- Descarga HTTP con protección SSRF, redirecciones limitadas, timeout y límite
  de 20 MiB.
- Resize y conversión real para JPEG/PNG con la biblioteca estándar de Go.
- Almacenamiento local intercambiable mediante un puerto de aplicación.
- Metadatos de salida y endpoint `GET /v1/jobs/{id}/result`.

### Frontend

- React, TypeScript, Vite y Tailwind CSS.
- Organización por feature.
- Formulario de creación de trabajos.
- Polling y actualización manual del estado.
- Tema claro/oscuro con preferencia guardada en `localStorage`.
- Estados visuales para `queued`, `processing`, `completed`, `failed` y
  `cancelled`.
- Enlace al archivo generado cuando el trabajo termina.

### Infraestructura local

- PostgreSQL 17.
- NATS 2.11 con JetStream.
- Docker Compose con healthchecks.
- `infra:up` espera hasta que los servicios estén saludables.
- PostgreSQL guarda sus datos en `infra/docker/data/postgres`, excluido de Git.
- NATS conserva sus datos en el volumen Docker `imageflow_nats_data`.

## Lo que todavía no hace

La primera implementación real está limitada deliberadamente a JPEG y PNG.
Todavía faltan:

- Resize con interpolación de mayor calidad.
- Soporte WebP y AVIF mediante un motor apropiado.
- Almacenamiento S3-compatible para despliegues distribuidos; el adaptador
  actual usa `infra/docker/data/objects` y solo es apropiado para desarrollo.
- Autenticación, API keys, cuotas y rate limiting.
- Métricas, trazas y alertas.
- Kubernetes y despliegue productivo.
- Autenticación de la documentación interactiva antes de un despliegue público.

## Estructura relevante

```text
apps/
├── api/
│   ├── cmd/api/                 API HTTP
│   ├── cmd/worker/              consumidor y publicador
│   ├── cmd/migrate/             migrador
│   ├── internal/jobs/
│   │   ├── domain/              entidades y reglas
│   │   ├── application/         casos de uso y puertos
│   │   └── adapters/            HTTP, PostgreSQL, NATS y memoria
│   ├── internal/outbox/         publicación confiable de eventos
│   ├── internal/platform/       configuración, DB y mensajería
│   └── migrations/              migraciones SQL embebidas
└── console/
    └── src/
        ├── app/
        ├── features/jobs/
        └── shared/

infra/docker/                    PostgreSQL y NATS locales
docs/                            arquitectura, ADR y estado
```

## Endpoints actuales

```text
GET  /health/live
GET  /health/ready
GET  /openapi.yaml
GET  /docs
POST /v1/jobs
GET  /v1/jobs/{id}
```

`POST /v1/jobs` responde `202 Accepted`. La consola consulta posteriormente
`GET /v1/jobs/{id}` hasta alcanzar un estado terminal.

## Comandos cotidianos

Desde la raíz del monorepo:

```bash
pnpm bootstrap          # instalar dependencias JS y descargar módulos Go
pnpm dev                # infraestructura, migraciones, API, worker y consola
pnpm check              # tests unitarios, typecheck y go vet
pnpm build              # build del backend y frontend
pnpm test:integration   # PostgreSQL y NATS reales
pnpm infra:status       # estado de los contenedores
pnpm infra:logs         # logs de PostgreSQL y NATS
pnpm infra:down         # detener infraestructura sin borrar datos
```

Servicios locales:

```text
Console:       http://localhost:5173
API:           http://localhost:8080
PostgreSQL:    localhost:5432
NATS:          localhost:4222
NATS monitor:  http://localhost:8222
```

## Pruebas

Los tests Go se ubican junto al paquete probado, siguiendo la convención del
lenguaje. Las pruebas reales usan el build tag `integration`.

```bash
pnpm check
pnpm test:integration
```

La prueba de integración crea o reutiliza la base aislada `imageflow_test`. El
nombre debe terminar en `_test` para impedir que el test use accidentalmente
la base de desarrollo.

## Configuración principal

La aplicación funciona con valores locales predeterminados. Variables
importantes:

```text
IMAGEFLOW_HTTP_ADDRESS
IMAGEFLOW_DATABASE_URL
IMAGEFLOW_NATS_URL
IMAGEFLOW_NATS_STREAM
IMAGEFLOW_NATS_SUBJECT
IMAGEFLOW_NATS_CONSUMER
IMAGEFLOW_NATS_DLQ_STREAM
IMAGEFLOW_NATS_DLQ_SUBJECT
IMAGEFLOW_NATS_MAX_DELIVERIES
```

Los archivos `.env` están excluidos de Git. Nunca colocar secretos reales en
los valores predeterminados, Compose, documentación o commits.

## Decisiones vigentes

- Monorepo administrado con pnpm workspaces, sin Nx por ahora.
- Monolito modular en Go; API y worker se despliegan como procesos separados.
- Arquitectura hexagonal pragmática, sin carpetas ceremoniales innecesarias.
- PostgreSQL es la fuente de verdad del estado de los trabajos.
- NATS transporta eventos; no sustituye la persistencia del dominio.
- Entrega al menos una vez e idempotencia mediante actualizaciones
  condicionales.
- Outbox para evitar perder el evento después de crear un trabajo.
- Tests unitarios junto al código; futuros E2E pueden vivir en `tests/e2e`.
- OpenAPI 3.1 será contract-first y Scalar será la interfaz de documentación.

## Riesgos y deuda técnica conocida

- Un trabajo `processing` se recupera tras 15 segundos sin actividad. El
  procesamiento real necesitará lease o heartbeat para operaciones largas.
- La DLQ directa entre NATS y PostgreSQL no es una transacción distribuida;
  debe evaluarse una outbox específica para eventos terminales al endurecer la
  operación productiva.
- No existen todavía pruebas E2E de caja negra sobre toda la plataforma.
- No hay autenticación ni aislamiento por cliente; la API es solo local.
- Los puertos de Compose son para desarrollo y no deben exponerse así en
  producción.
- Google Sans no está incluida como webfont; se usa solo si está disponible en
  el sistema.

## Próximo incremento recomendado

Implementar el núcleo real de procesamiento detrás de un puerto de aplicación:

1. Definir contratos `ImageFetcher`, `ImageProcessor` y `ObjectStorage`.
2. Descargar con límites estrictos y defensa SSRF.
3. Procesar inicialmente JPEG, PNG y WebP.
4. Guardar el resultado en un adaptador local compatible con posterior S3.
5. Persistir metadatos y URL de salida en el trabajo.
6. Añadir pruebas con fixtures pequeñas y una prueba E2E.
7. Mantener el contrato OpenAPI 3.1 y Scalar sincronizados con cada endpoint.

## Checklist para retomar en otra sesión

1. Leer este documento y `docs/architecture.md`.
2. Ejecutar `git status` y revisar cambios no confirmados.
3. Ejecutar `pnpm check`.
4. Levantar el entorno con `pnpm dev`.
5. Crear un trabajo desde la consola y confirmar que termina en `completed`.
6. Antes de cambiar contratos, revisar los ADR existentes y crear uno nuevo si
   la decisión afecta arquitectura, persistencia o compatibilidad pública.
