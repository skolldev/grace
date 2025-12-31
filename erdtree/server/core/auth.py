import secrets

from fastapi import Depends, Header, HTTPException
from sqlmodel import Session

from server.models.models import Setting
from server.core.database import get_session

API_KEY_SETTING = "api_key"

# Module-level cache to avoid DB lookup on every request
_api_key: str | None = None


def get_or_create_api_key_with_session(session: Session) -> str:
    """Get existing API key or generate a new one using the provided session."""
    global _api_key
    if _api_key is not None:
        return _api_key

    setting = session.get(Setting, API_KEY_SETTING)
    if setting:
        _api_key = setting.value
        return _api_key

    # Generate new key with recognizable prefix
    key = "grc_" + secrets.token_hex(24)  # 48 hex chars + prefix
    setting = Setting(key=API_KEY_SETTING, value=key)
    session.add(setting)
    session.commit()
    _api_key = key
    return key


def get_or_create_api_key() -> str:
    """Get existing API key. Only works after cache is populated."""
    global _api_key
    if _api_key is not None:
        return _api_key
    raise RuntimeError(
        "API key not initialized. Call get_or_create_api_key_with_session first."
    )


def verify_api_key(
    authorization: str = Header(None),
    session: Session = Depends(get_session),
) -> None:
    """FastAPI dependency to validate Bearer token."""
    if not authorization:
        raise HTTPException(status_code=401, detail="Missing authorization header")

    if not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Invalid authorization format")

    token = authorization[7:]  # Strip "Bearer "
    api_key = get_or_create_api_key_with_session(session)
    if not secrets.compare_digest(token, api_key):
        raise HTTPException(status_code=401, detail="Invalid API key")


def reset_api_key_cache() -> None:
    """Reset the cached API key. Used for testing."""
    global _api_key
    _api_key = None
