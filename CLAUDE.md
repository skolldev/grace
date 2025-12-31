# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Grace is a self-hosted system monitoring platform with three components:
- **erdtree/** - FastAPI backend server (Python 3.11)
- **tarnished/** - Go-based monitoring agent
- **grace-ui/** - Angular 21 frontend (in progress)

## Build & Development Commands

### Erdtree (Python Backend)

```bash
cd erdtree
pip install -e ".[dev]"                              # Install with dev dependencies
.venv/Scripts/uvicorn server.main:app --reload       # Run server (localhost:8000)
.venv/Scripts/pytest tests/ -v                       # Run all tests
.venv/Scripts/pytest tests/test_devices.py::test_register_device -v  # Single test
.venv/Scripts/ruff check server/ tests/              # Lint
.venv/Scripts/ruff format server/ tests/             # Format
```

### Tarnished (Go Agent)

```bash
cd tarnished
go mod download                    # Install dependencies
make build                         # Build for current platform
make build-all                     # Cross-compile all platforms
go test -v ./...                   # Run all tests
go test -v ./internal/config       # Run specific package tests
```

### Grace-UI (Angular Frontend)

```bash
cd grace-ui
pnpm install                       # Install dependencies
pnpm start                         # Run dev server
pnpm build                         # Build
pnpm test                          # Run tests (Vitest)
```

## Architecture

### Data Flow

1. Tarnished agents register with Erdtree using an API key obtained from `/api/admin/key`
2. Agents push metrics to Erdtree at configurable intervals
3. Erdtree stores metrics in SQLite with timestamps
4. Grace-UI queries Erdtree API for metrics visualization

### Erdtree Structure (erdtree/server/)

- `main.py` - FastAPI app, middleware, exception handlers
- `core/auth.py` - API key management
- `core/database.py` - SQLite via SQLModel
- `core/logger.py` - Database-backed logging
- `models/` - SQLModel ORM models and Pydantic schemas
- `api/` - Route handlers (devices, sensors, metrics, logs, admin)

### Tarnished Structure (tarnished/internal/)

- `agent/` - Core orchestration and lifecycle management
- `collector/` - System metrics collection (CPU, memory, disk, network)
- `config/` - Config loading (file, env vars, flags)
- `httpclient/` - Erdtree API client
- `queue/` - Offline metric queue for disconnected operation
- `registration/` - Device registration and state persistence
- `sensors/` - Optional hardware sensor support (temperature, fans, voltage)

## CI/CD

Both components have GitHub Actions workflows triggered on path-specific changes:
- **Erdtree CI**: pytest, ruff (lint + format check)
- **Tarnished CI**: Tests on Linux/Windows/macOS, golangci-lint, cross-platform builds

## Angular Guidelines (grace-ui)

- Use standalone components (default in Angular 21, do NOT set `standalone: true`)
- Use signals for state management with `input()`, `output()`, `computed()`
- Set `changeDetection: ChangeDetectionStrategy.OnPush`
- Use native control flow (`@if`, `@for`, `@switch`) instead of structural directives
- Use `inject()` function instead of constructor injection
- Use host bindings in `@Component` decorator, not `@HostBinding`/`@HostListener`
- Use `NgOptimizedImage` for static images
- Prefer reactive forms over template-driven
