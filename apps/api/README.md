# ImageFlow

ImageFlow is an asynchronous image-processing API designed to integrate with other applications.

## Current state

The first vertical slice provides:

- `POST /v1/jobs` to validate and enqueue an image-processing job.
- `GET /v1/jobs/{id}` to retrieve a job.
- Liveness and readiness endpoints.
- PostgreSQL persistence through `pgx`.
- A transactional outbox record created with every job.
- Separate API and worker entrypoints.

NATS JetStream and object storage will be added through adapters without changing the domain contract.

## Run

```bash
cd /home/archlinux/Documents/proyectos/imageflow
make infra-up
make migrate
make api
```

The default database connection is
`postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable`.
Override it with `IMAGEFLOW_DATABASE_URL` for the API and `DATABASE_URL` for migrations.

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
make fmt
make test
make vet
```
