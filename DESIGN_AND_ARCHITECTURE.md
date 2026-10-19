# System Design & Architecture: Project Prism (`livekit-prism`)
## Dual-Channel Multimodal Tool Gateway & Agent Plugin for LiveKit & PostHog

> **Core Sub-Components:** `prism-gateway` (Distributed Go Core), `livekit-plugins-posthog` (In-Process Python Plugin), `prism-web` (FastAPI Harness & Chart.js Client)  

---

## 1. Executive Summary & Vision

Autonomous voice agents built on WebRTC (such as LiveKit Agents) face a critical architectural failure when interacting with analytical or enterprise APIs (such as PostHog or ClickHouse): **The Voice-to-API Mismatch**.

Traditional APIs return dense, multi-line JSON datasets or markdown tables. Feeding these bulky objects back into a real-time LLM voice pipeline introduces 4 to 6 seconds of dead air, degrades TTS synthesis into reading raw table pipes and floating-point numbers out loud, and wastes the user's screen.

**Project Prism** introduces an **Optical Dual-Channel Routing Invariant**:
- **Visual Spectrum:** Heavy visualization JSON is pushed directly across LiveKit WebRTC Data Channels (`posthog-viz-card`), rendering interactive PostHog charts on the client display without entering the LLM context.
- **Audio Spectrum:** An ultra-compact speech-ready summary (`< 30` tokens, natural language numbers, zero markdown) is injected into the voice model with sub-1.2s conversational turn latency.

![Prism Multimodal Voice & Telemetry Demo](file:///Users/chung/sandbox/posthog/livekit-prism/assets/demo.gif)

---

## 2. System Architecture & Component Topology

```text
 ┌─────────────────────────────────────────────────────────┐
 │               Browser Client (prism-web)                │
 │  - LiveKit WebRTC UMD Client                            │
 │  - Chart.js Dynamic Card Renderer                       │
 └──────────────┬────────────────────────────▲─────────────┘
                │                            │
   WebRTC Audio │                            │ WebRTC Data Channel
  (Mic In/Agent)│                            │ Topic: "posthog-viz-card"
                ▼                            │
 ┌─────────────────────────────┐             │
 │     LiveKit SFU Server      │             │
 │   (WebRTC Media Engine)     │             │
 └──────────────┬──────────────┘             │
                │                            │
   WebRTC Track │ Dispatch                   │
                ▼                            │
 ┌─────────────────────────────┐             │
 │  LiveKit Voice Agent Worker │             │
 │         (agent.py)          │             │
 │                             │             │
 │  ┌───────────────────────┐  │             │
 │  │livekit-plugins-posthog│  │             │
 │  │- PostHogAgentObserver │──┼─── PostHog AI Telemetry ($ai_generation)
 │  │- PostHogDualChannel   │  │             │
 │  └───────────┬───────────┘  │             │
 └──────────────┼──────────────┘             │
                │ Tool Call (HTTP/RPC)       │
                ▼                            │
 ┌───────────────────────────────────────────┴─────────────┐
 │               Prism Go Edge Gateway                     │
 │  - cmd/prism-gateway/router.go (Optical Split)          │
 │  - cmd/prism-gateway/livekit_client.go (gRPC SendData)  │
 │  - cmd/prism-gateway/telemetry.go (ClickHouse Ingestion)│
 └──────────────┬──────────────────────────────────────────┘
                │
         Batch Ingestion
                ▼
 ┌─────────────────────────────┐
 │      ClickHouse Server      │
 │  (livekit_telemetry DB)     │
 └─────────────────────────────┘
```

---

## 3. Core Component Breakdown

### A. The Go Distributed Edge Gateway (`cmd/prism-gateway`)
- **Location:** [`cmd/prism-gateway/`](file:///Users/chung/sandbox/posthog/livekit-prism/cmd/prism-gateway/)
- **Binary:** [`bin/prism-gateway`](file:///Users/chung/sandbox/posthog/livekit-prism/bin/prism-gateway)
- **Role:** High-performance, low-latency edge service written in Go.
- **Key Modules:**
  1. [`main.go`](file:///Users/chung/sandbox/posthog/livekit-prism/cmd/prism-gateway/main.go): Server lifecycle, HTTP mux, signal trapping, and connection pooling.
  2. [`router.go`](file:///Users/chung/sandbox/posthog/livekit-prism/cmd/prism-gateway/router.go): Implements the Optical Prism split:
     - Formats Contract A visual cards (`posthog-viz-card`).
     - Dispatches visual cards asynchronously to LiveKit via gRPC.
     - Generates Contract B Speech-Ready Deltas (`< 30` tokens) for the agent LLM.
     - Exposes conversational filler phrases (`/api/tools/filler`) to mask query latency.
  3. [`livekit_client.go`](file:///Users/chung/sandbox/posthog/livekit-prism/cmd/prism-gateway/livekit_client.go): Wraps LiveKit's gRPC `RoomServiceClient.SendData()` with connection pooling.
  4. [`telemetry.go`](file:///Users/chung/sandbox/posthog/livekit-prism/cmd/prism-gateway/telemetry.go): Asynchronous, non-blocking ClickHouse batch recorder using native TCP protocol (`clickhouse-go/v2`).

### B. Python Agent Plugin (`livekit-plugins-posthog`)
- **Location:** [`plugins/livekit-plugins-posthog/`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/)
- **Role:** Upstream-compliant in-process Python plugin for LiveKit Agents.
- **Key Modules:**
  1. [`observer.py`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/observer.py): [`PostHogAgentObserver`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/observer.py#L11-L24) listens to `session.on("metrics_collected")`. Maps full conversational latency waterfalls: `LLMMetrics` (`$ai_generation`), `InterruptionMetrics` (`voice_agent_interrupted`), `TTSMetrics` (`voice_agent_tts_latency`), `STTMetrics` (`voice_agent_stt_latency`), and `EOUMetrics` (`voice_agent_endpointing_delay`).
  2. [`tools.py`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/tools.py): [`PostHogDualChannelTools`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/tools.py#L17-L29) wraps analytical endpoints using LiveKit 1.x `@function_tool` with local WebRTC data channel fallback.
  3. [`feature_flags.py`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/feature_flags.py): [`PostHogFeatureFlags`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/feature_flags.py#L11-L24) evaluates boolean feature gates, multivariate model A/B testing, and dynamic prompt experiment overrides.
  4. [`models.py`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/models.py): Dataclass schemas for plugin configuration ([`PostHogPluginOptions`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/models.py#L11-L21)), conversion funnels ([`VisualFunnelCard`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/models.py#L31-L40)), and latency waterfalls ([`LatencyWaterfallEvent`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/models.py#L42-L52)).
  5. [`__init__.py`](file:///Users/chung/sandbox/posthog/livekit-prism/plugins/livekit-plugins-posthog/livekit/plugins/posthog/__init__.py): Registers `PostHogPlugin(Plugin)` with `Plugin.register_plugin()` and exports sub-modules.

### C. Client Harness & WebRTC Token Server (`prism-web`)
- **Location:** [`server.py`](file:///Users/chung/sandbox/posthog/livekit-prism/server.py), [`templates/index.html`](file:///Users/chung/sandbox/posthog/livekit-prism/templates/index.html)
- **Role:** Zero-build FastAPI and HTML5 WebRTC testing harness.
- **Features:**
  - Mints LiveKit WebRTC AccessTokens with data channel publishing grants (`/api/token`).
  - Browser UI imports LiveKit Client SDK via UMD CDN and Chart.js.
  - Subscribes to `RoomEvent.DataReceived` on topic `"posthog-viz-card"` and dynamically renders responsive funnel bar charts.

---

## 4. Interface Contracts

### Contract A: WebRTC Data Channel Visual Cards (`posthog-viz-card`)
Broadcast directly to the browser over WebRTC data channels for real-time interactive rendering:

1. **Conversion Funnel (`funnel_viz`):**
```json
{
  "type": "funnel_viz",
  "title": "Checkout Conversion Dropoff",
  "funnel_id": "checkout_v2",
  "steps": [
    { "name": "Cart Viewed", "visitors": 14200, "conversion_rate": 100.0 },
    { "name": "Billing Entered", "visitors": 9100, "conversion_rate": 64.1 },
    { "name": "Payment Confirmed", "visitors": 3200, "conversion_rate": 22.5 }
  ],
  "primary_drop_step": "Payment Confirmed",
  "timestamp": 1729123200
}
```

2. **API Error Waterfall (`error_waterfall`):**
```json
{
  "type": "error_waterfall",
  "title": "API Error Waterfall (24h)",
  "timeframe": "24h",
  "total_errors": 42,
  "primary_root_cause": "PaymentGatewayTimeout (504)",
  "errors": [
    { "error_type": "PaymentGatewayTimeout (504)", "count": 22, "percentage": 52.4, "status_code": 504 },
    { "error_type": "StripeCardDeclined (400)", "count": 13, "percentage": 30.9, "status_code": 400 },
    { "error_type": "DatabaseDeadlock (500)", "count": 7, "percentage": 16.7, "status_code": 500 }
  ],
  "timestamp": 1729123200
}
```

3. **Cohort Retention Trend (`retention_trend`):**
```json
{
  "type": "retention_trend",
  "title": "8-Week Cohort Retention Trend",
  "cohort_name": "Weekly Active Users",
  "terminal_retention": 19.5,
  "intervals": [
    { "interval": "Week 1", "retention_rate": 100.0, "users": 8400 },
    { "interval": "Week 2", "retention_rate": 44.2, "users": 3712 },
    { "interval": "Week 8", "retention_rate": 19.5, "users": 1638 }
  ],
  "timestamp": 1729123200
}
```

4. **HogQL Analytical Query Table (`hogql_viz`):**
```json
{
  "type": "hogql_viz",
  "title": "HogQL Analytical Query Result",
  "query": "SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event ORDER BY count() DESC LIMIT 5",
  "columns": ["event", "count()"],
  "rows": [
    ["$pageview", 48200],
    ["user_signed_up", 1240],
    ["checkout_step_completed", 890]
  ],
  "timestamp": 1729123200
}
```

### Contract B: Voice LLM Speech-Ready Deltas
Injected back into the agent's function-calling context (strictly $< 30$ tokens, natural phrasing, no raw markdown):
- **Funnel:** `"The checkout funnel has a 22.5 percent conversion rate. The main drop-off is at the payment confirmation step. I've projected the conversion funnel onto your screen."`
- **Errors:** `"We detected 42 server errors in the last 24 hours. The highest failure rate is Payment Gateway Timeouts at 52 percent. The error waterfall is on your screen."`
- **Retention:** `"Your eight-week user retention stands at 19.5 percent, beating the industry average of 15 percent. The retention curve is displayed on your screen."`
- **HogQL:** `"Your HogQL query completed successfully across 48,000 pageviews and 1,240 signups. I've projected the analytical results onto your screen."`

### Contract C: PostHog AI Observability Schema (`$ai_generation`)
Emitted by `PostHogAgentObserver` on `metrics_collected`:
```json
{
  "event": "$ai_generation",
  "properties": {
    "$session_id": "RM_prism_1729123200",
    "$ai_trace_id": "req_xyz789",
    "$ai_model": "gpt-4o-mini",
    "$ai_provider": "openai",
    "$ai_input_tokens": 142,
    "$ai_output_tokens": 28,
    "$ai_latency": 0.42,
    "$ai_time_to_first_token": 0.18,
    "tokens_per_second": 66.7,
    "cancelled": false
  }
}
```

### Contract D: PostHog Session Replay Linking Schema (`$session_replay_link`)
Cross-links real-time LiveKit WebRTC voice sessions directly into PostHog Session Replay recordings:
```json
{
  "event": "$session_replay_link",
  "properties": {
    "$session_id": "RM_prism_1729123200",
    "livekit_room": "prism-demo",
    "participant_identity": "user-alex",
    "replay_url": "https://us.posthog.com/project/12345/replay/RM_prism_1729123200",
    "webrtc_stream_linked": true,
    "timestamp": 1729123200
  }
}
```

---

## 5. End-to-End Execution Flow

```text
User                  Web Client                 Agent (agent.py)             Prism Gateway               ClickHouse
 │                        │                             │                           │                          │
 │ 1. "Check funnel"      │                             │                           │                          │
 ├───────────────────────►│                             │                           │                          │
 │ (WebRTC Audio)         ├────────────────────────────►│                           │                          │
 │                        │                             │ 2. Tool Invocation        │                          │
 │                        │                             │    (query_funnel)         │                          │
 │                        │                             ├──────────────────────────►│                          │
 │                        │                             │                           │ 3. Optical Split         │
 │                        │                             │                           │    - Visual JSON         │
 │                        │                             │                           │    - Speech Delta        │
 │                        │                             │                           ├─────────────────────────►│
 │                        │                             │                           │    Async Batch Insert    │
 │                        │ 4. WebRTC Data Channel Card │                           │                          │
 │                        │    ("posthog-viz-card")     │                           │                          │
 │                        │◄────────────────────────────┼───────────────────────────┤                          │
 │                        │ (Chart renders in browser)  │                           │                          │
 │                        │                             │ 5. Speech Delta (<30 toks)│                          │
 │                        │                             │◄──────────────────────────┤                          │
 │ 6. Audio Response      │                             │                           │                          │
 │    ("22.5% rate...")   │                             │                           │                          │
 │◄───────────────────────┴─────────────────────────────┤                           │                          │
 │                        │                             │                           │                          │
```

---

## 6. Performance & Latency Budgets

Using Python notation for the conversational latency budget:

```python
turn_latency = wire_overhead + llm_ttft + tts_ttfb
assert turn_latency < 1.2  # Conversational pause budget in seconds
```

- **Wire Overhead:** Go Edge Gateway optical split executes in `< 1.5ms`.
- **LLM Context Budget:** Speech-ready delta is strictly capped at `< 30 tokens`, preventing LLM completion latency from exceeding `400ms`.
- **Database Telemetry:** ClickHouse batch worker flushes in chunks of `50` records or every `1.0s` without blocking the request-response thread.

---

## 7. Execution Runbook

All startup scripts are located in [`livekit-prism/bin/`](file:///Users/chung/sandbox/posthog/livekit-prism/bin/):

```bash
cd /Users/chung/sandbox/posthog/livekit-prism

# 1. Local Infrastructure (ClickHouse & NATS)
./bin/run-infra.sh

# 2. LiveKit SFU Server
./bin/run-livekit.sh

# 3. Prism Go Gateway
./bin/run-gateway.sh

# 4. FastAPI WebRTC Client Harness
./bin/run-web.sh

# 5. LiveKit Voice Agent Worker
./bin/run-agent.sh

# Stop all background processes
./bin/stop-all.sh
```
