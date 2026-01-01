# Erdtree

FastAPI-based backend server for the Grace system monitoring platform.

## Core Concepts

**Architecture**: RESTful API built with FastAPI and SQLModel, using SQLite for data persistence.

**Device Registration**: Devices register using a persistent API key. The key is auto-generated on first startup and retrieved via the admin endpoint for configuring agents.

**Metrics Collection**: Devices push JSON-formatted metrics to the server, which stores them with timestamps for querying and analysis.

**Database Logging**: All operations, errors, and significant events are automatically logged to the database via helper functions in `server/core/logger.py`.

## Project Structure

```
server/
├── main.py              # FastAPI application, middleware, exception handlers
├── core/
│   ├── auth.py          # API key management and authentication
│   ├── database.py      # Database engine and session management
│   └── logger.py        # Database-backed logging utilities
├── models/
│   ├── models.py        # SQLModel ORM models
│   └── schemas.py       # Pydantic request/response schemas
└── api/
    ├── admin.py         # API key retrieval endpoint
    ├── devices.py       # Device registration and management
    ├── sensors.py       # Sensor data endpoints
    ├── metrics.py       # Metrics storage and retrieval
    └── logs.py          # Log retrieval endpoints
```

## Configuration

**Database**: SQLite database location is configured in `server/core/database.py`:

```python
DATABASE_URL = "sqlite:///./erdtree.db"
```

**CORS**: Cross-origin settings are in `server/main.py`. Currently set to allow all origins for development.

**Dependencies**: Managed in `pyproject.toml` with separate dev dependencies for testing and linting.

## Development Setup

### Prerequisites

- Python 3.11 or higher
- pip

### Installation

```bash
cd erdtree
pip install -e ".[dev]"
```

This installs the package in editable mode with all development dependencies.

### Running the Server

```bash
cd erdtree
.venv/Scripts/uvicorn server.main:app --reload
```

The API will be available at `http://localhost:8000`. Interactive API docs are at `http://localhost:8000/docs`.

## Testing

### Run All Tests

```bash
cd erdtree
.venv/Scripts/pytest tests/ -v
```

### Run Specific Test File

```bash
cd erdtree
.venv/Scripts/pytest tests/test_devices.py -v
```

### Run Single Test

```bash
cd erdtree
.venv/Scripts/pytest tests/test_devices.py::test_register_device -v
```

Tests use an in-memory SQLite database and are isolated from the development database.

## Code Quality

### Linting

```bash
cd erdtree
.venv/Scripts/ruff check server/ tests/
```

### Formatting

```bash
cd erdtree
.venv/Scripts/ruff format server/ tests/
```

Ruff configuration is in `pyproject.toml` with a line length of 88 and Python 3.11 target.

## API Overview

| Endpoint                           | Method     | Purpose                                  |
| ---------------------------------- | ---------- | ---------------------------------------- |
| `/health`                          | GET        | Health check                             |
| `/api/devices/register`            | POST       | Register a new device                    |
| `/api/devices`                     | GET        | List all devices (includes latest metrics) |
| `/api/devices/{id}`                | GET/DELETE | Get or delete a device                   |
| `/api/devices/{id}/metrics`        | POST       | Push metrics from device                 |
| `/api/devices/{id}/metrics`        | GET        | Retrieve historical metrics              |
| `/api/devices/{id}/sensors`        | GET/POST   | Get or report device sensors             |
| `/api/devices/{id}/sensors/config` | GET/PUT    | Get or update sensor config              |
| `/api/admin/key`                   | GET        | Retrieve API key for agent configuration |
| `/api/logs`                        | GET        | Retrieve application logs                |

Query parameters for metrics retrieval: `start`, `end`, `limit`
