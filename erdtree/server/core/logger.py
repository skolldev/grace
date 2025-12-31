from sqlmodel import Session

from server.core.database import get_engine
from server.models.models import Log


def log(content: str, source: str, type: str = "info") -> Log:
    """Save a log entry to the database.

    Args:
        content: The log message
        source: Where the log originated (e.g., "devices", "metrics", "admin")
        type: Log level - "info", "warning", or "error"

    Returns:
        The created Log object
    """
    with Session(get_engine()) as session:
        entry = Log(content=content, source=source, type=type)
        session.add(entry)
        session.commit()
        session.refresh(entry)
        return entry


def log_info(content: str, source: str) -> Log:
    return log(content, source, "info")


def log_warning(content: str, source: str) -> Log:
    return log(content, source, "warning")


def log_error(content: str, source: str) -> Log:
    return log(content, source, "error")
