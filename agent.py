import asyncio
import logging
import os
import sys

from dotenv import load_dotenv

# Ensure local plugin directory is discoverable
plugins_path = os.path.join(os.path.dirname(__file__), "plugins", "livekit-plugins-posthog")
if plugins_path not in sys.path:
    sys.path.insert(0, plugins_path)

from livekit.agents import (
    Agent,
    AgentSession,
    AutoSubscribe,
    JobContext,
    WorkerOptions,
    cli,
)
from livekit.plugins import openai, silero
from livekit.plugins.posthog import PostHogAgentObserver, PostHogDualChannelTools

load_dotenv()
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("prism-agent")

LIVEKIT_URL = os.getenv("LIVEKIT_URL", "ws://127.0.0.1:7880")
LIVEKIT_API_KEY = os.getenv("LIVEKIT_API_KEY", "devkey")
LIVEKIT_API_SECRET = os.getenv("LIVEKIT_API_SECRET", "secret")


async def entrypoint(ctx: JobContext):
    logger.info("Connecting agent to room: %s", ctx.room.name)
    await ctx.connect(auto_subscribe=AutoSubscribe.AUDIO_ONLY)

    # 1. Initialize PostHog AI Observability Observer
    observer = PostHogAgentObserver(
        api_key=os.getenv("POSTHOG_API_KEY", "phc_mock_key_prism"),
        host=os.getenv("POSTHOG_HOST", "https://us.i.posthog.com"),
        session_id=ctx.room.name,
    )

    # 2. Initialize Dual-Channel Analytical Tools
    tools = PostHogDualChannelTools(
        gateway_url=os.getenv("PRISM_GATEWAY_URL", "http://localhost:8080"),
        room_name=ctx.room.name,
        livekit_room=ctx.room,
    )

    # 3. Configure Voice Agent with tools, VAD, STT, LLM, TTS
    agent = Agent(
        instructions="You are an analytical assistant connected to PostHog and ClickHouse. "
                     "When asked about conversion funnels or metrics, call the query_funnel tool. "
                     "Keep verbal replies concise and under 30 words, with natural language numbers.",
        tools=tools.get_tools(),
        vad=silero.VAD.load(),
        stt=openai.STT(),
        llm=openai.LLM(model=os.getenv("OPENAI_MODEL", "gpt-4o-mini")),
        tts=openai.TTS(),
    )

    session = AgentSession()

    # 4. Attach PostHog metrics observer to track LLMMetrics & InterruptionMetrics
    session.on("metrics_collected", observer.on_metrics_collected)

    # 5. Start assistant loop in the room
    session.start(agent, room=ctx.room)

    # Initial greeting
    await session.say(
        "Hello! I am connected to your PostHog analytics and ClickHouse data. How can I help you?",
        allow_interruptions=True,
    )


if __name__ == "__main__":
    cli.run_app(
        WorkerOptions(
            entrypoint_fnc=entrypoint,
            ws_url=LIVEKIT_URL,
            api_key=LIVEKIT_API_KEY,
            api_secret=LIVEKIT_API_SECRET,
        )
    )
