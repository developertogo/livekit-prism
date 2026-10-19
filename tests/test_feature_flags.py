import sys
import os
from unittest.mock import MagicMock

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "plugins", "livekit-plugins-posthog")))

from livekit.plugins.posthog.feature_flags import PostHogFeatureFlags
from livekit.plugins.posthog.models import PostHogPluginOptions, FunnelStep, VisualFunnelCard


def test_models_dataclasses():
    options = PostHogPluginOptions(api_key="phc_test_123")
    assert options.api_key == "phc_test_123"
    assert options.capture_tts_metrics is True

    step = FunnelStep(name="Checkout", visitors=1000, conversion_rate=50.0)
    assert step.name == "Checkout"

    card = VisualFunnelCard(title="Test Funnel", steps=[step])
    assert card.title == "Test Funnel"
    assert len(card.steps) == 1


def test_feature_flags_evaluation():
    mock_client = MagicMock()
    mock_client.feature_enabled.return_value = True
    mock_client.get_feature_flag.return_value = "variant-b"
    mock_client.get_feature_flag_payload.return_value = "You are an analytical assistant."

    flags = PostHogFeatureFlags(client=mock_client, default_distinct_id="session-user-1")

    # Boolean flag
    assert flags.is_feature_enabled("beta_funnel_tool") is True
    mock_client.feature_enabled.assert_called_with(
        key="beta_funnel_tool",
        distinct_id="session-user-1",
        person_properties=None,
    )

    # Multivariate flag
    assert flags.get_feature_flag("llm_model_choice") == "variant-b"

    # Prompt variant payload
    prompt = flags.get_prompt_variant("system_prompt_experiment", fallback_prompt="Default prompt")
    assert prompt == "You are an analytical assistant."


def test_feature_flags_offline_fallback():
    # Null client / network failure
    flags = PostHogFeatureFlags(client=None)

    assert flags.is_feature_enabled("any_flag", default=False) is False
    assert flags.is_feature_enabled("any_flag", default=True) is True
    assert flags.get_prompt_variant("prompt_exp", fallback_prompt="Default fallback") == "Default fallback"
