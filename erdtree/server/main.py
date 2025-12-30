from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from fastapi.exceptions import HTTPException, RequestValidationError

from server.core.database import create_db_and_tables
from server.core.logger import log_error
from server.api import devices, metrics, admin, logs

@asynccontextmanager
async def lifespan(app: FastAPI):
    create_db_and_tables()
    yield


app = FastAPI(title="Erdtree API", version="0.1.0", description="Self-hosted system monitoring", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Tighten in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(devices.router, prefix="/api/devices", tags=["devices"])
app.include_router(metrics.router, prefix="/api/metrics", tags=["metrics"])
app.include_router(admin.router, prefix="/api/admin", tags=["admin"])
app.include_router(logs.router, prefix="/api/logs", tags=["logs"])


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    log_error(f"{type(exc).__name__}: {exc}", "system")
    return JSONResponse(status_code=500, content={"detail": "Internal server error"})


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    if exc.status_code >= 400:
        log_error(f"{exc.detail}", "system")
    return JSONResponse(status_code=exc.status_code, content={"detail": exc.detail})


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    log_error(f"Validation error: {exc.errors()}", "system")
    return JSONResponse(status_code=422, content={"detail": exc.errors()})


@app.get("/health")
async def health_check():
    return {"status": "ok"}
