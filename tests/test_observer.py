import sys
import os
from unittest.mock import MagicMock

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "plugins", "livekit-plugins-posthog")))

from livekit.plugins import posthog
from livekit.plugins.posthog.observer import PostHogAgentObserver, Observer, AgentObserver, observe
from livekit.plugins.posthog.tools import Tools, DualChannelTools
from livekit.plugins.posthog.feature_flags import FeatureFlags
from livekit.plugins.posthog.models import PluginOptions


def test_idiomatic_livekit_exports():
    # Verify LiveKit idiomatic non-redundant naming
    assert posthog.Observer is posthog.PostHogAgentObserver
    assert posthog.AgentObserver is posthog.PostHogAgentObserver
    assert posthog.Tools is posthog.PostHogDualChannelTools
    assert posthog.FeatureFlags is posthog.PostHogFeatureFlags
    assert posthog.PluginOptions is posthog.PostHogPluginOptions

    mock_session = MagicMock()
    obs = posthog.observe(mock_session, session_id="RM_idiomatic_1")
    assert isinstance(obs, posthog.Observer)
    mock_session.on.assert_called_with("metrics_collected", obs.on_metrics_collected)


def test_observer_initialization():
    observer = PostHogAgentObserver(
        api_key="phc_mock_test_key",
        host="https://mock.posthog.com",
        session_id="RM_test_123",
    )
    assert observer.session_id == "RM_test_123"
    assert observer.flags is not None



def test_observer_record_llm_metrics():
    observer = PostHogAgentObserver(session_id="RM_test_456")
    observer.client = MagicMock()

    class MockLLMMetrics:
        input_tokens = 150
        output_tokens = 30
        duration = 0.5
        time_to_first_token = 0.15
        model = "gpt-4o-mini"
        cancelled = False
        request_id = "req_test_abc"

    observer.record_llm_metrics(MockLLMMetrics())

    observer.client.capture.assert_called_once()
    call_args = observer.client.capture.call_args
    assert call_args.kwargs["event"] == "$ai_generation"
    props = call_args.kwargs["properties"]

    assert props["$session_id"] == "RM_test_456"
    assert props["$ai_trace_id"] == "req_test_abc"
    assert props["$ai_model"] == "gpt-4o-mini"
    assert props["$ai_provider"] == "openai"
    assert props["$ai_input_tokens"] == 150
    assert props["$ai_output_tokens"] == 30
    assert props["$ai_latency"] == 0.5
    assert props["$ai_time_to_first_token"] == 0.15
    assert props["tokens_per_second"] == 60.0
    assert props["cancelled"] is False


def test_observer_record_interruption_metrics():
    observer = PostHogAgentObserver(session_id="RM_test_789")
    observer.client = MagicMock()

    class MockInterruptionMetrics:
        duration = 1.25

    observer.record_interruption_metrics(MockInterruptionMetrics())

    observer.client.capture.assert_called_once()
    call_args = observer.client.capture.call_args
    assert call_args.kwargs["event"] == "voice_agent_interrupted"
    assert call_args.kwargs["properties"]["speech_duration_before_cut"] == 1.25


def test_observer_record_tts_metrics():
    observer = PostHogAgentObserver(session_id="RM_test_tts")
    observer.client = MagicMock()

    class MockTTSMetrics:
        ttfb = 0.28
        duration = 0.75
        audio_duration = 2.4
        characters_count = 85
        label = "openai/tts-1"
        cancelled = False

    observer.record_tts_metrics(MockTTSMetrics())

    observer.client.capture.assert_called_once()
    call_args = observer.client.capture.call_args
    assert call_args.kwargs["event"] == "voice_agent_tts_latency"
    props = call_args.kwargs["properties"]
    assert props["tts_ttfb"] == 0.28
    assert props["tts_duration"] == 0.75
    assert props["characters_count"] == 85
    assert props["tts_model"] == "openai/tts-1"


def test_observer_record_stt_metrics():
    observer = PostHogAgentObserver(session_id="RM_test_stt")
    observer.client = MagicMock()

    class MockSTTMetrics:
        duration = 0.35
        audio_duration = 1.8
        label = "openai/whisper"
        streamed = True

    observer.record_stt_metrics(MockSTTMetrics())

    observer.client.capture.assert_called_once()
    call_args = observer.client.capture.call_args
    assert call_args.kwargs["event"] == "voice_agent_stt_latency"
    props = call_args.kwargs["properties"]
    assert props["stt_duration"] == 0.35
    assert props["audio_duration"] == 1.8
    assert props["stt_model"] == "openai/whisper"


def test_observer_record_eou_metrics():
    observer = PostHogAgentObserver(session_id="RM_test_eou")
    observer.client = MagicMock()

    class MockEOUMetrics:
        end_of_utterance_delay = 0.45
        transcription_delay = 0.22
        on_user_turn_completed_delay = 0.12

    observer.record_eou_metrics(MockEOUMetrics())

    observer.client.capture.assert_called_once()
    call_args = observer.client.capture.call_args
    assert call_args.kwargs["event"] == "voice_agent_endpointing_delay"
    props = call_args.kwargs["properties"]
    assert props["eou_delay"] == 0.45
    assert props["transcription_delay"] == 0.22


def test_observer_link_session_replay():
    observer = PostHogAgentObserver(
        api_key="phc_mock_test_key",
        host="https://us.posthog.com",
        session_id="RM_replay_test_999",
        project_id="proj_888",
    )
    observer.client = MagicMock()

    replay_url = observer.link_session_replay(
        livekit_room="prism-demo",
        participant_identity="user-alice",
    )

    assert "https://us.posthog.com/project/proj_888/replay/RM_replay_test_999" in replay_url
    observer.client.capture.assert_called_once()
    call_args = observer.client.capture.call_args
    assert call_args.kwargs["event"] == "$session_replay_link"
    props = call_args.kwargs["properties"]
    assert props["$session_id"] == "RM_replay_test_999"
    assert props["livekit_room"] == "prism-demo"
    assert props["participant_identity"] == "user-alice"
    assert props["webrtc_stream_linked"] is True

