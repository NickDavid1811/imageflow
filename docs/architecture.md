# Arquitectura de ImageFlow

ImageFlow usa un monorepo con aplicaciones desplegables independientes.

```text
apps/console -> apps/api -> PostgreSQL
                       \-> transactional outbox -> publisher -> NATS JetStream
                                                                  |
                                                                  v
                                                               worker
                                                                  |
                                                                  v
                                                             PostgreSQL
```

## Aplicaciones

- `apps/api`: módulo Go que produce los binarios API, worker y migraciones.
- `apps/console`: consola React que consume la API HTTP.

## Infraestructura

`infra/docker` contiene únicamente el entorno de desarrollo local. Los despliegues de Kubernetes se incorporarán cuando exista una primera versión completa del flujo de procesamiento.

## Principios

- Monolito modular en el backend.
- Arquitectura hexagonal pragmática en Go.
- Organización feature-based en React.
- Procesos API y worker escalables de forma independiente.
- Eventos confiables mediante Transactional Outbox.

## Entrega de trabajos

La API crea el trabajo y el evento `job.requested.v1` en una misma transacción. El worker ejecuta dos procesos coordinados:

1. El publicador reclama eventos pendientes de la outbox y los publica con un identificador estable en JetStream.
2. El consumidor durable reclama el trabajo en PostgreSQL, simula su procesamiento y persiste el progreso.

La entrega es al menos una vez. Las actualizaciones condicionales de estado hacen que el consumidor sea idempotente. Un trabajo `processing` sin actividad puede recuperarse después de 15 segundos.
