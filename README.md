# Grace

![Erdtree CI](https://github.com/skolldev/grace/workflows/Erdtree%20CI/badge.svg)
![Tarnished CI](https://github.com/skolldev/grace/workflows/Tarnished%20CI/badge.svg)

Self-hosted system monitoring platform with lightweight agent-based metrics collection.

## Components

- **[erdtree/](erdtree/)** - FastAPI backend server for storing and serving metrics
- **[tarnished/](tarnished/)** - Go-based monitoring agent for collecting system metrics
- **[grace-ui/](grace-ui/)** - Frontend UI (in progress)

## Quick Start

### 1. Start the Erdtree Server

```bash
cd erdtree
pip install -e ".[dev]"
.venv/Scripts/uvicorn server.main:app --reload
```

Server will be available at `http://localhost:8000`

### 2. Get API Key

```bash
curl http://localhost:8000/api/admin/key
```

### 3. Run Tarnished Agent

```bash
cd tarnished
make build
./bin/tarnished run --server http://localhost:8000 --api-key grc_your_api_key
```

## Development

See component READMEs for detailed setup and development instructions:

- [erdtree/README.md](erdtree/README.md) - Backend API server
- [tarnished/README.md](tarnished/README.md) - Monitoring agent

## CI/CD

GitHub Actions workflows automatically run tests and linting on all pull requests:

- **Erdtree CI**: Python 3.11-3.12, pytest, ruff
- **Tarnished CI**: Go 1.21-1.22, tests on Linux/Windows/macOS, golangci-lint
