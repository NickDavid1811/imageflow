# ImageFlow

ImageFlow is an asynchronous image-processing API designed to integrate with other applications.

## Current state

The first vertical slice provides:

- `POST /v1/jobs` to validate and enqueue an image-processing job.
- `GET /v1/jobs/{id}` to retrieve a job.
- Liveness and readiness endpoints.
- PostgreSQL persistence through `pgx`.
- A transactional outbox record created with every job.
- Reliable publication to NATS JetStream.
- An idempotent durable worker with simulated progress.
- Separate API and worker entrypoints.

Object storage and real image processing will be added through adapters without changing the domain contract.

## Run

```bash
cd /home/archlinux/Documents/proyectos/imageflow
pnpm infra:up
pnpm migrate
pnpm api
```

The default database connection is
`postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable`.
Override it with `IMAGEFLOW_DATABASE_URL` for the API and `DATABASE_URL` for migrations.

The worker also accepts:

```text
IMAGEFLOW_NATS_URL=nats://localhost:4222
IMAGEFLOW_NATS_STREAM=IMAGEFLOW_JOBS
IMAGEFLOW_NATS_SUBJECT=imageflow.jobs.requested.v1
IMAGEFLOW_NATS_CONSUMER=imageflow-workers
```

Create a job:

```bash
curl -i http://localhost:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "source_url": "https://example.com/product.jpg",
    "operations": [
      {"type": "resize", "width": 800},
      {"type": "convert", "format": "webp", "quality": 80}
    ]
  }'
```

## Verify

```bash
pnpm check
pnpm build
```
