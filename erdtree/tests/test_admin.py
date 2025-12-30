from fastapi.testclient import TestClient


def test_create_token(client: TestClient):
    response = client.post("/api/admin/tokens?expires_in_hours=24")
    assert response.status_code == 200
    data = response.json()
    assert "token" in data
    assert "expires_at" in data


def test_create_token_custom_expiry(client: TestClient):
    response = client.post("/api/admin/tokens?expires_in_hours=48")
    assert response.status_code == 200
    data = response.json()
    assert "token" in data


def test_list_tokens_empty(client: TestClient):
    response = client.get("/api/admin/tokens")
    assert response.status_code == 200
    assert response.json() == []


def test_list_tokens_after_creation(client: TestClient):
    client.post("/api/admin/tokens")
    client.post("/api/admin/tokens")

    response = client.get("/api/admin/tokens")
    assert response.status_code == 200
    tokens = response.json()
    assert len(tokens) == 2
