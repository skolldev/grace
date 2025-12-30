from datetime import timedelta
from fastapi import APIRouter, Depends, Query
from sqlmodel import Session, select

from server.core.database import get_session
from server.models.models import RegistrationToken, utc_now
from server.models.schemas import TokenResponse

router = APIRouter()


@router.post("/tokens", response_model=TokenResponse)
def create_token(
    expires_in_hours: int = Query(
        24, ge=1, le=168, description="Token validity in hours"
    ),
    session: Session = Depends(get_session),
):
    token = RegistrationToken(
        expires_at=utc_now() + timedelta(hours=expires_in_hours)
    )
    session.add(token)
    session.commit()
    session.refresh(token)

    return TokenResponse(token=token.token, expires_at=token.expires_at)


@router.get("/tokens")
def list_tokens(session: Session = Depends(get_session)):
    tokens = session.exec(select(RegistrationToken)).all()
    return [
        {
            "token": t.token,
            "created_at": t.created_at.isoformat(),
            "expires_at": t.expires_at.isoformat(),
            "used": t.used,
            "expired": t.expires_at < utc_now(),
        }
        for t in tokens
    ]
