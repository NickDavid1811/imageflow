# ImageFlow Console

Interfaz operativa ligera para crear y consultar trabajos de ImageFlow.

## Requisitos

- Node.js 24+
- ImageFlow API ejecutándose en `http://localhost:8080`

## Desarrollo

```bash
cd /home/archlinux/Documents/proyectos/imageflow
pnpm install
make console
```

Vite expone la consola en `http://localhost:5173` y redirige las solicitudes `/api` hacia la API local de ImageFlow, evitando requerir CORS durante el desarrollo.

## Arquitectura

El código está organizado por funcionalidad. La dependencia principal fluye así:

```text
components -> hooks -> jobs-api -> shared/http-client
```

```text
src/
├── app/                    # Composición global
├── features/jobs/
│   ├── api/                # Operaciones remotas del módulo
│   ├── components/         # Presentación de trabajos
│   ├── hooks/              # Estado y casos de uso de interfaz
│   ├── model/              # Tipos, reglas y validación
│   └── pages/              # Composición de la funcionalidad
└── shared/
    ├── api/                # Cliente HTTP genérico
    ├── config/             # Configuración del entorno
    └── icons/              # Recursos visuales compartidos
```

Los componentes no llaman directamente a `fetch` y el modelo no depende de React.

## Verificación

```bash
pnpm --filter @imageflow/console typecheck
pnpm --filter @imageflow/console build
```
