# Arquitectura de ImageFlow

ImageFlow usa un monorepo con aplicaciones desplegables independientes.

```text
apps/console -> apps/api -> PostgreSQL
                       \-> transactional outbox -> NATS -> worker
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
