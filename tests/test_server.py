import sys
import os
from fastapi.testclient import TestClient

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from server import app

client = TestClient(app)


def test_health_endpoint():
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ok"
    assert data["service"] == "prism-web"


def test_token_endpoint():
    resp = client.get("/api/token?room=unit-test-room&identity=test-user")
    assert resp.status_code == 200
    data = resp.json()
    assert "token" in data
    assert len(data["token"]) > 20
    assert data["room"] == "unit-test-room"
    assert data["identity"] == "test-user"


def test_serve_index():
    resp = client.get("/")
    assert resp.status_code == 200
    assert "PRISM" in resp.text
    assert "posthog-viz-card" in resp.text
