import asyncio
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "plugins", "livekit-plugins-posthog")))

from livekit.plugins.posthog.tools import PostHogDualChannelTools


def test_tools_initialization():
    tools = PostHogDualChannelTools(
        gateway_url="http://localhost:8080",
        room_name="prism-test-room",
    )
    assert tools.room_name == "prism-test-room"
    tool_list = tools.get_tools()
    assert len(tool_list) == 4
    assert tools.query_funnel is not None
    assert tools.query_errors is not None
    assert tools.query_retention is not None
    assert tools.query_hogql is not None


def test_query_funnel_fallback():
    tools = PostHogDualChannelTools(
        gateway_url="http://localhost:9999",  # unreachable port
        room_name="prism-test-room",
    )

    speech_delta = asyncio.run(tools._query_funnel_impl(funnel_id="checkout_v2"))

    # Contract B invariants:
    # 1. Concise, conversational string
    # 2. Under 35 words (< 30 tokens)
    # 3. No raw markdown tables
    assert "checkout funnel" in speech_delta
    assert "|" not in speech_delta
    words = speech_delta.split()
    assert len(words) <= 35


def test_query_errors_fallback():
    tools = PostHogDualChannelTools(
        gateway_url="http://localhost:9999",
        room_name="prism-test-room",
    )

    speech_delta = asyncio.run(tools._query_errors_impl(timeframe="24h"))
    assert "server errors" in speech_delta
    assert "|" not in speech_delta
    words = speech_delta.split()
    assert len(words) <= 35


def test_query_retention_fallback():
    tools = PostHogDualChannelTools(
        gateway_url="http://localhost:9999",
        room_name="prism-test-room",
    )

    speech_delta = asyncio.run(tools._query_retention_impl(cohort="weekly"))
    assert "retention" in speech_delta
    assert "|" not in speech_delta
    words = speech_delta.split()
    assert len(words) <= 35


def test_query_hogql_fallback():
    tools = PostHogDualChannelTools(
        gateway_url="http://localhost:9999",
        room_name="prism-test-room",
    )

    speech_delta = asyncio.run(tools._query_hogql_impl(query="SELECT count() FROM events"))
    assert "HogQL" in speech_delta
    assert "|" not in speech_delta
    words = speech_delta.split()
    assert len(words) <= 35


