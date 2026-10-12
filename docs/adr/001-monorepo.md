# ADR 001: Monorepo para ImageFlow

- Estado: aceptado
- Fecha: 2026-10-12

## Contexto

La API, el worker, la consola y la infraestructura local evolucionan conjuntamente y comparten contratos operativos.

## Decisión

Mantener los componentes de ImageFlow en un monorepo. Las aplicaciones permanecen desplegables de forma independiente bajo `apps/`, mientras la infraestructura local reside en `infra/`.

## Consecuencias

- Los cambios coordinados pueden verificarse en una sola revisión.
- Existe una única entrada para desarrollo y documentación.
- Go Modules y pnpm continúan administrando sus dependencias independientemente.
- El monorepo no implica desplegar todos los procesos como una sola unidad.
