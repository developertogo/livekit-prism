import os
from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from livekit.api import AccessToken, VideoGrants
from dotenv import load_dotenv

load_dotenv()

LIVEKIT_URL = os.getenv("LIVEKIT_URL", "ws://127.0.0.1:7880")
LIVEKIT_API_KEY = os.getenv("LIVEKIT_API_KEY", "devkey")
LIVEKIT_API_SECRET = os.getenv("LIVEKIT_API_SECRET", "secret")

app = FastAPI(
    title="Prism WebRTC Harness",
    description="WebRTC Token Server & UI Harness for LiveKit Prism Multimodal Gateway",
    version="0.1.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/health")
async def health():
    return {"status": "ok", "service": "prism-web"}

@app.get("/api/token")
async def get_token(
    room: str = Query("prism-demo", description="Room name to join"),
    identity: str = Query("user-browser", description="Participant identity"),
    name: str = Query("Browser User", description="Display name"),
):
    """Mints a LiveKit WebRTC AccessToken with permissions for room join and data-channel communication."""
    try:
        grants = VideoGrants(
            room_join=True,
            room=room,
            can_publish=True,
            can_subscribe=True,
            can_publish_data=True,
        )
        token = (
            AccessToken(LIVEKIT_API_KEY, LIVEKIT_API_SECRET)
            .with_identity(identity)
            .with_name(name)
            .with_grants(grants)
            .to_jwt()
        )
        return {
            "token": token,
            "url": LIVEKIT_URL,
            "room": room,
            "identity": identity,
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to mint token: {str(e)}")

@app.get("/", response_class=HTMLResponse)
async def serve_index():
    template_path = os.path.join(os.path.dirname(__file__), "templates", "index.html")
    if not os.path.exists(template_path):
        raise HTTPException(status_code=404, detail="Template index.html not found")
    with open(template_path, "r", encoding="utf-8") as f:
        return HTMLResponse(content=f.read())
