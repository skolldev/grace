from fastapi.testclient import TestClient


def test_get_api_key(client: TestClient):
    response = client.get("/api/admin/key")
    assert response.status_code == 200
    data = response.json()
    assert "api_key" in data
    assert data["api_key"].startswith("grc_")
    assert len(data["api_key"]) == 52  # grc_ + 48 hex chars


def test_api_key_is_stable(client: TestClient):
    """Test that API key remains the same across calls."""
    response1 = client.get("/api/admin/key")
    response2 = client.get("/api/admin/key")
    assert response1.json()["api_key"] == response2.json()["api_key"]
