from fastapi import APIRouter, Depends
from sqlmodel import Session, select

from server.core.database import get_session
from server.models.models import Log

router = APIRouter()


@router.get("/")
def get_all_logs(session: Session = Depends(get_session)):
    logs = session.exec(select(Log).order_by(Log.timestamp.desc())).all()
    return logs
