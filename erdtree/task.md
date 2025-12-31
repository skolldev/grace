# Task: Simplify Authentication to Shared API Key

## Overview

Replace the per-device token registration system with a single shared API key. The server auto-generates the key on first startup and persists it in the database. This removes the need to generate tokens for each device while still providing basic authentication.

## Current Flow (Token-based)

1. Admin creates a registration token in UI
2. Token is single-use and expires
3. Agent registers with token
4. Server validates token, marks it used, returns device ID
5. Agent stores device ID for future requests

**Problems:**

- Requires manual token generation for each device
- Tokens expire and can only be used once
- Extra database table and API endpoints for token management

## New Flow (Shared API Key)

1. Server starts, checks for API key in database
2. If missing, generates a random key and stores it
3. Admin retrieves key from authenticated endpoint (`GET /api/admin/key`)
4. All agents configured with the same API key
5. Agent sends `Authorization: Bearer <api-key>` on all requests
6. If no stored device ID, agent calls `/api/devices/register` with system info
7. Server validates API key, creates device, returns device ID
8. Agent stores device ID, continues pushing metrics

**Benefits:**

- One key for unlimited agents
- No token generation, expiry, or management
- Agents just show up and register themselves
- Zero configuration on server side (key auto-generated)
- Simpler codebase (remove token table and endpoints)

## Implementation

### Server Changes

**1. Add Settings model for persistence**

```python
# models/models.py
class Setting(SQLModel, table=True):
    __tablename__ = "settings"

    key: str = Field(primary_key=True)
    value: str
```

**2. Add API key management**

```python
# core/auth.py
import secrets
from sqlmodel import Session, select
from server.models.models import Setting
from server.core.database import engine

def get_or_create_api_key() -> str:
    """Get existing API key or generate a new one."""
    with Session(engine) as session:
        setting = session.get(Setting, "api_key")
        if setting:
            return setting.value

        # Generate new key with recognizable prefix
        key = "grc_" + secrets.token_hex(24)  # 48 chars + prefix
        setting = Setting(key="api_key", value=key)
        session.add(setting)
        session.commit()
        return key

# Cache the key at module level after first load
_api_key: str | None = None

def get_api_key() -> str:
    global _api_key
    if _api_key is None:
        _api_key = get_or_create_api_key()
    return _api_key
```

**3. Add authentication dependency**

```python
# core/auth.py
from fastapi import Header, HTTPException

def verify_api_key(authorization: str = Header(None)):
    if not authorization:
        raise HTTPException(status_code=401, detail="Missing authorization header")

    if not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Invalid authorization format")

    token = authorization[7:]  # Strip "Bearer "
    if token != get_api_key():
        raise HTTPException(status_code=401, detail="Invalid API key")
```

**4. Add admin endpoint to retrieve key (UI-authenticated)**

```python
# api/admin.py
@router.get("/key")
def get_api_key_endpoint(
    # TODO: Add UI session authentication dependency
):
    """Retrieve the API key for agent configuration."""
    return {"api_key": get_api_key()}
```

**5. Apply auth to agent-facing routes**

```python
# api/devices.py
@router.post("/register", response_model=RegisterResponse)
def register_device(
    request: RegisterRequest,
    session: Session = Depends(get_session),
    _: None = Depends(verify_api_key),  # Add auth
):
    # Remove token validation logic
    # Just create device directly
    device = Device(
        hostname=request.hostname,
        os=request.os,
        arch=request.arch,
        ip_address=request.ip_address,
    )
    session.add(device)
    session.commit()
    session.refresh(device)
    return RegisterResponse(device_id=device.id, message="Device registered")
```

**6. Update RegisterRequest schema**

```python
# models/schemas.py
class RegisterRequest(BaseModel):
    # Remove: token: str
    hostname: str
    os: str
    arch: str
    ip_address: Optional[str] = None
```

**7. Remove token-related code**

- Delete `RegistrationToken` model
- Delete `POST /api/admin/tokens` endpoint
- Delete `GET /api/admin/tokens` endpoint
- Remove token validation from `register_device`
- Remove/update related tests

### Agent Changes

**1. Add API key to config**

```go
// config/config.go
type Config struct {
    Server         string
    APIKey         string  // New field, replaces Token
    Interval       time.Duration
    LogLevel       string
    SensorsEnabled bool
}

// Bind env var
viper.BindEnv("api_key", "GRACE_API_KEY")
```

**2. Update HTTP client to send auth header**

```go
// httpclient/client.go
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
    if c.apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.apiKey)
    }
    return c.httpClient.Do(req)
}
```

**3. Simplify registration**

```go
// registration/registration.go
type RegisterRequest struct {
    // Remove Token field
    Hostname  string `json:"hostname"`
    OS        string `json:"os"`
    Arch      string `json:"arch"`
    IPAddress string `json:"ip_address,omitempty"`
}
```

**4. Update CLI flags**

```go
// cmd/tarnished/main.go
rootCmd.PersistentFlags().String("api-key", "", "API key for server authentication")
viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))

// Remove --token flag
```

### Install Script Changes

```bash
# Old
curl ... | bash -s -- --server https://x --token abc123

# New
curl ... | bash -s -- --server https://x --api-key your-shared-key
```

Or via environment variable:

```bash
GRACE_SERVER=https://x GRACE_API_KEY=your-key ./install.sh
```

## Database Migration

- Add `settings` table for key-value storage
- `RegistrationToken` table can be dropped
- No data migration needed (tokens are ephemeral)

## Security Considerations

- API key is auto-generated with secure random (48 hex chars + prefix)
- Never logged — only retrievable via authenticated admin endpoint
- Transmit only over HTTPS in production
- Key is shared across all agents — if compromised, regenerate via database
- Key stored in database, backed up with regular DB backups

## Testing Checklist

- [ ] Server generates API key on first startup
- [ ] Server reuses existing API key on subsequent startups
- [ ] `GET /api/admin/key` returns the key (when authenticated)
- [ ] Server rejects agent requests without API key
- [ ] Server rejects agent requests with wrong API key
- [ ] Server accepts agent requests with correct API key
- [ ] Agent registers successfully with API key
- [ ] Agent pushes metrics with API key in header
- [ ] Multiple agents can register with same API key
- [ ] Old token-related tests removed/updated

## Files to Modify

### Server (erdtree)

- `server/core/auth.py` — new file for API key generation and auth dependency
- `server/models/models.py` — add `Setting` model, remove `RegistrationToken`
- `server/models/schemas.py` — remove token from `RegisterRequest`
- `server/api/devices.py` — remove token validation, add auth dependency
- `server/api/admin.py` — replace token endpoints with `GET /key`
- `server/main.py` — ensure settings table created on startup
- `tests/test_devices.py` — update registration tests
- `tests/test_admin.py` — replace token tests with key retrieval test

### Agent (tarnished)

- `internal/config/config.go` — replace `Token` with `APIKey`
- `internal/httpclient/client.go` — add auth header to all requests
- `internal/registration/registration.go` — remove token from request
- `cmd/tarnished/main.go` — update CLI flags (`--api-key` instead of `--token`)
