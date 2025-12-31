from fastapi import APIRouter, Depends
from sqlmodel import Session

from server.core.auth import get_or_create_api_key_with_session
from server.core.database import get_session
from server.models.schemas import ApiKeyResponse

router = APIRouter()


@router.get("/key", response_model=ApiKeyResponse)
def get_api_key(session: Session = Depends(get_session)):
    """Retrieve the API key for agent configuration."""
    return ApiKeyResponse(api_key=get_or_create_api_key_with_session(session))
