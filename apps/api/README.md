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
- An idempotent durable worker with real JPEG/PNG processing.
- S3-compatible result storage backed by MinIO in local development.
- Separate API and worker entrypoints.

Object storage and real image processing will be added through adapters without changing the domain contract.

## Run

```bash
cd /home/archlinux/Documents/proyectos/imageflow
pnpm infra:up
pnpm migrate
pnpm api
```

The public contract and interactive reference are available while the API is running:

```text
http://localhost:8080/openapi.yaml
http://localhost:8080/docs
```

The default database connection is
`postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable`.
Override it with `IMAGEFLOW_DATABASE_URL` for the API, worker and migrations.

The worker also accepts:

```text
IMAGEFLOW_NATS_URL=nats://localhost:4222
IMAGEFLOW_NATS_STREAM=IMAGEFLOW_JOBS
IMAGEFLOW_NATS_SUBJECT=imageflow.jobs.requested.v1
IMAGEFLOW_NATS_CONSUMER=imageflow-workers
IMAGEFLOW_NATS_DLQ_STREAM=IMAGEFLOW_JOBS_DLQ
IMAGEFLOW_NATS_DLQ_SUBJECT=imageflow.jobs.failed.v1
IMAGEFLOW_NATS_MAX_DELIVERIES=10
```

Create a job:

```bash
curl -i http://localhost:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "source_url": "https://example.com/product.jpg",
    "operations": [
      {"type": "resize", "width": 800},
      {"type": "convert", "format": "jpeg", "quality": 80}
    ]
  }'
```

## Verify

```bash
pnpm check
pnpm build
```

The root command provisions the isolated `imageflow_test` database and runs the
PostgreSQL and NATS JetStream integration tests:

```bash
pnpm test:integration
```

When running the backend script directly, configure
`IMAGEFLOW_TEST_DATABASE_URL` (the database name must end in `_test`) and
`IMAGEFLOW_TEST_NATS_URL` explicitly.
