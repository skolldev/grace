# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Grace is a self-hosted system monitoring platform with three components:
- **erdtree/** - FastAPI backend server (Python)
- **grace-ui/** - Frontend UI (in progress)
- **tarnished/** - Go-based monitoring agent (in progress)

## Common Commands

### Erdtree (Backend)

```bash
# Install dependencies
cd erdtree && pip install -e ".[dev]"

# Run tests
cd erdtree && .venv/Scripts/pytest tests/ -v

# Run single test file
cd erdtree && .venv/Scripts/pytest tests/test_devices.py -v

# Run single test
cd erdtree && .venv/Scripts/pytest tests/test_devices.py::test_register_device -v

# Start dev server
cd erdtree && .venv/Scripts/uvicorn server.main:app --reload

# Lint
cd erdtree && .venv/Scripts/ruff check server/ tests/
cd erdtree && .venv/Scripts/ruff format server/ tests/
```

## Architecture

### Erdtree Backend Structure

```
erdtree/server/
├── main.py              # FastAPI app, middleware, exception handlers
├── core/
│   ├── database.py      # SQLModel engine, session management
│   └── logger.py        # DB-backed logging (log_info, log_warning, log_error)
├── models/
│   ├── models.py        # ORM models (Device, Metric, RegistrationToken, Log)
│   └── schemas.py       # Pydantic request/response schemas
└── api/
    ├── devices.py       # Device registration & CRUD
    ├── metrics.py       # Metrics push & retrieval
    ├── admin.py         # Token management
    └── logs.py          # Log retrieval
```

### Data Flow

1. **Registration**: Admin creates token → Device registers with token → Token marked used
2. **Metrics**: Device pushes JSON metrics → Stored with timestamp → Retrieved with filters
3. **Logging**: All operations logged via `logger.py` helpers → Stored in DB → Queryable via API

### Key Patterns

- **Dependency injection**: `session: Session = Depends(get_session)` for DB access
- **UTC timestamps**: Use `utc_now()` from `models.py` (naive UTC for SQLite compatibility)
- **Global exception handlers**: All errors logged to DB automatically
- **Test isolation**: In-memory SQLite with dependency override in `conftest.py`

### API Routes

| Route | Purpose |
|-------|---------|
| `POST /api/devices/register` | Register device with token |
| `GET /api/devices` | List devices |
| `GET/DELETE /api/devices/{id}` | Get/delete device |
| `POST /api/metrics` | Push metrics |
| `GET /api/metrics/{device_id}` | Get metrics (supports `?start=`, `?end=`, `?limit=`) |
| `POST /api/admin/tokens` | Create registration token |
| `GET /api/admin/tokens` | List tokens |
| `GET /api/logs/` | Get all logs |
| `GET /health` | Health check |
