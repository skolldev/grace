from sqlmodel import SQLModel, Session, create_engine

DATABASE_URL = "sqlite:///./erdtree.db"

engine = create_engine(
    DATABASE_URL, echo=True, connect_args={"check_same_thread": False}
)

# Allow tests to override the engine
_engine_override = None


def get_engine():
    """Get the current database engine (allows test override)."""
    return _engine_override if _engine_override is not None else engine


def set_engine_override(new_engine):
    """Override the engine for testing."""
    global _engine_override
    _engine_override = new_engine


def create_db_and_tables():
    SQLModel.metadata.create_all(get_engine())


def get_session():
    with Session(get_engine()) as session:
        yield session
