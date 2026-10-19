import math
import os
from PIL import Image, ImageDraw, ImageFont

WIDTH, HEIGHT = 920, 560
BG_COLOR = (9, 13, 22)
PANEL_BG = (17, 24, 39)
CARD_BG = (13, 19, 32)
BORDER_COLOR = (31, 41, 55)
TEXT_WHITE = (249, 250, 251)
TEXT_MUTED = (156, 163, 175)
ACCENT_BLUE = (59, 130, 246)
ACCENT_INDIGO = (99, 102, 241)
ACCENT_ORANGE = (245, 78, 0)
SUCCESS_GREEN = (16, 185, 129)
DANGER_RED = (239, 68, 68)

def get_font(size, bold=False):
    # Try Helvetica.ttc on macOS (index 1 is Bold, index 0 is Regular)
    font_path = "/System/Library/Fonts/Helvetica.ttc"
    if os.path.exists(font_path):
        try:
            return ImageFont.truetype(font_path, size, index=1 if bold else 0)
        except Exception:
            pass
    # Fallback system font paths
    candidates = [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold else "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/System/Library/Fonts/Geneva.dfont",
    ]
    for c in candidates:
        if os.path.exists(c):
            try:
                return ImageFont.truetype(c, size)
            except Exception:
                continue
    return ImageFont.load_default()

font_title = get_font(18, bold=True)
font_sub = get_font(11, bold=False)
font_label = get_font(14, bold=True)
font_badge = get_font(11, bold=True)
font_code = get_font(12, bold=True)
font_banner = get_font(11, bold=True)
font_text = get_font(13, bold=False)

def draw_rounded_rect(draw, bbox, radius, fill, outline=None):
    draw.rounded_rectangle(bbox, radius=radius, fill=fill, outline=outline, width=1)

def create_base_canvas(status_text="LiveKit Connected", dc_active=True, cards_count=1):
    img = Image.new("RGB", (WIDTH, HEIGHT), BG_COLOR)
    draw = ImageDraw.Draw(img)

    # 1. Header Bar
    draw.rectangle([(0, 0), (WIDTH, 52)], fill=(17, 24, 39))
    draw.line([(0, 52), (WIDTH, 52)], fill=BORDER_COLOR, width=1)

    # Logo prism icon
    draw.polygon([(25, 38), (42, 14), (59, 38)], fill=ACCENT_BLUE)
    draw.text((70, 16), "PRISM", font=font_title, fill=TEXT_WHITE)
    draw.text((145, 20), "|  Optical WebRTC Multimodal Gateway", font=font_sub, fill=TEXT_MUTED)

    # Status Pill
    draw_rounded_rect(draw, (WIDTH - 195, 12, WIDTH - 20, 40), 14, fill=(15, 23, 42), outline=(37, 99, 235))
    draw.ellipse((WIDTH - 180, 21, WIDTH - 170, 31), fill=SUCCESS_GREEN)
    draw.text((WIDTH - 160, 17), status_text, font=font_sub, fill=TEXT_WHITE)

    # 2. Left Column: Voice Agent Session Panel
    draw_rounded_rect(draw, (20, 68, 335, 465), 10, fill=PANEL_BG, outline=BORDER_COLOR)
    draw.text((35, 82), "Voice Agent Session", font=font_label, fill=TEXT_WHITE)

    # Left Column metrics box (Showcasing core features + newly added Feature Flags)
    draw_rounded_rect(draw, (35, 220, 320, 450), 8, fill=(11, 17, 28), outline=BORDER_COLOR)
    metrics = [
        ("Room", "prism-demo"),
        ("Data Channel", "Active (Ready)" if dc_active else "Inactive"),
        ("Cards Received", str(cards_count)),
        ("Turn Latency", "1.18s"),
        ("TTFT", "180ms"),
        ("PostHog Telemetry", "$ai_generation"),
        ("Feature Flags", "Active (v2_funnel)"),
    ]
    y_m = 230
    for label, val in metrics:
        draw.text((45, y_m), label, font=font_sub, fill=TEXT_MUTED)
        draw.text((185, y_m), val, font=font_code, fill=TEXT_WHITE)
        draw.line([(45, y_m + 22), (310, y_m + 22)], fill=(25, 35, 50), width=1)
        y_m += 30

    # 3. Right Column: Visual Spectrum Panel
    draw_rounded_rect(draw, (350, 68, WIDTH - 20, 465), 10, fill=PANEL_BG, outline=BORDER_COLOR)
    draw.text((370, 82), "Visual Spectrum  (WebRTC Data Channel Card)", font=font_label, fill=TEXT_WHITE)
    draw.text((WIDTH - 180, 84), "Topic: posthog-viz-card", font=font_sub, fill=TEXT_MUTED)

    return img, draw

frames = []

# Funnel step definitions
steps_data = [
    ("Cart Viewed", 14200, "100.0%", ACCENT_BLUE, 0.90),
    ("Billing Entered", 9100, "64.1%", ACCENT_INDIGO, 0.58),
    ("Payment Confirmed", 3200, "22.5%", ACCENT_ORANGE, 0.20),
]

# Phase 1: User Speaking (4 frames)
for f_idx in range(4):
    img, draw = create_base_canvas(status_text="LiveKit Connected", dc_active=True, cards_count=0)

    # Audio Wave Container (User speaking - Electric Blue)
    c_x1, c_y1, c_x2, c_y2 = 35, 110, 320, 212
    draw_rounded_rect(draw, (c_x1, c_y1, c_x2, c_y2), 8, fill=(11, 17, 28), outline=BORDER_COLOR)
    wave_x = 75
    for i in range(11):
        h = int(12 + 20 * abs(math.sin(f_idx * 0.9 + i * 0.8)))
        draw.rounded_rectangle([(wave_x, 146 - h // 2), (wave_x + 8, 146 + h // 2)], radius=4, fill=ACCENT_BLUE)
        wave_x += 16

    draw.text(((c_x1 + c_x2) / 2, 188), "User speaking: 'Check conversion funnel...'", font=font_sub, fill=TEXT_MUTED, anchor="mm")

    # Empty card container
    draw_rounded_rect(draw, (370, 115, WIDTH - 40, 450), 8, fill=(11, 17, 28), outline=BORDER_COLOR)
    draw.text((475, 260), "Awaiting WebRTC Data Channel Packet...", font=font_sub, fill=TEXT_MUTED)

    # Subtitle bar
    draw.rectangle([(0, HEIGHT - 70), (WIDTH, HEIGHT)], fill=(13, 19, 30))
    draw.line([(0, HEIGHT - 70), (WIDTH, HEIGHT - 70)], fill=BORDER_COLOR, width=1)
    draw.text((30, HEIGHT - 55), "USER (Voice Track):", font=font_label, fill=ACCENT_BLUE)
    draw.text((30, HEIGHT - 32), "\"Can you check the checkout conversion funnel?\"", font=font_text, fill=TEXT_WHITE)

    frames.append(img)

# Phase 2: Gateway Optical Split & Card Rendering Animation (7 frames)
for progress in [0.15, 0.35, 0.55, 0.75, 0.90, 1.0, 1.0]:
    img, draw = create_base_canvas(status_text="LiveKit Connected", dc_active=True, cards_count=1)

    # Audio Wave (Agent speaking waveform - Vibrant Orange)
    c_x1, c_y1, c_x2, c_y2 = 35, 110, 320, 212
    draw_rounded_rect(draw, (c_x1, c_y1, c_x2, c_y2), 8, fill=(11, 17, 28), outline=BORDER_COLOR)
    wave_x = 75
    for i in range(11):
        h = int(10 + 22 * abs(math.cos(progress * 4 + i * 0.7)))
        draw.rounded_rectangle([(wave_x, 146 - h // 2), (wave_x + 8, 146 + h // 2)], radius=4, fill=ACCENT_ORANGE)
        wave_x += 16
    draw.text(((c_x1 + c_x2) / 2, 188), "Agent replying verbally (<30 tokens)...", font=font_sub, fill=ACCENT_ORANGE, anchor="mm")

    # Active Visual Funnel Card
    draw_rounded_rect(draw, (370, 115, WIDTH - 40, 450), 8, fill=CARD_BG, outline=BORDER_COLOR)

    # Place 1: Badge Box (Guaranteed centered inside outer border)
    badge_text = "PostHog Funnel"
    bbox_b = font_badge.getbbox(badge_text)
    bw = bbox_b[2] - bbox_b[0]
    bx1, by1 = 390, 130
    bx2, by2 = bx1 + bw + 24, by1 + 24
    draw_rounded_rect(draw, (bx1, by1, bx2, by2), 4, fill=(45, 20, 12), outline=ACCENT_ORANGE)
    draw.text(((bx1 + bx2) / 2, (by1 + by2) / 2), badge_text, font=font_badge, fill=ACCENT_ORANGE, anchor="mm")

    # Card Title
    draw.text((390, 166), "Checkout Conversion Dropoff", font=font_label, fill=TEXT_WHITE)

    # Bars area
    chart_y_base = 352
    chart_h_max = 130
    col_w = 95
    spacing = 150
    start_x = 405

    for idx, (s_name, visitors, cr_str, color, norm_h) in enumerate(steps_data):
        bx = start_x + idx * spacing
        bar_h = int(chart_h_max * norm_h * progress)
        by = chart_y_base - bar_h

        # Bar
        draw_rounded_rect(draw, (bx, by, bx + col_w, chart_y_base), 6, fill=color)

        # Value labels cleanly positioned above bars (NO OVERLAP!)
        if progress > 0.4:
            draw.text((bx + 14, by - 32), f"{visitors:,}", font=font_code, fill=TEXT_WHITE)
            draw.text((bx + 18, by - 16), f"({cr_str})", font=font_sub, fill=TEXT_MUTED)

        # X-axis step label
        draw.text((bx - 2, chart_y_base + 8), s_name, font=font_sub, fill=TEXT_WHITE)

    # X-axis baseline
    draw.line([(385, chart_y_base), (WIDTH - 55, chart_y_base)], fill=(45, 55, 72), width=1)

    # Place 2: Critical Dropoff Alert Banner (Centered with generous margin for Payment Confirmed step)
    if progress >= 0.75:
        banner_text = "! Critical Dropoff: Largest exit detected at Payment Confirmed step (22.5%)"
        ban_x1, ban_y1 = 385, 385
        ban_x2 = WIDTH - 55
        ban_y2 = ban_y1 + 42
        draw_rounded_rect(draw, (ban_x1, ban_y1, ban_x2, ban_y2), 6, fill=(45, 14, 18), outline=DANGER_RED)
        draw.text(((ban_x1 + ban_x2) / 2, (ban_y1 + ban_y2) / 2), banner_text, font=font_banner, fill=(254, 202, 202), anchor="mm")

    # Subtitle bar at bottom
    draw.rectangle([(0, HEIGHT - 70), (WIDTH, HEIGHT)], fill=(13, 19, 30))
    draw.line([(0, HEIGHT - 70), (WIDTH, HEIGHT - 70)], fill=BORDER_COLOR, width=1)
    draw.text((30, HEIGHT - 55), "AGENT (Audio Spectrum Delta, 24 tokens):", font=font_label, fill=SUCCESS_GREEN)
    draw.text((30, HEIGHT - 32), "\"The checkout funnel has a 22.5% conversion rate. The main drop-off is at payment confirmation.\"", font=font_text, fill=TEXT_WHITE)

    frames.append(img)

# Hold final frame for 2.5 seconds (9 frames) so viewers can inspect the details
for _ in range(9):
    frames.append(frames[-1])

out_path = os.path.join(os.path.dirname(__file__), "assets", "demo.gif")
os.makedirs(os.path.dirname(out_path), exist_ok=True)
frames[0].save(
    out_path,
    save_all=True,
    append_images=frames[1:],
    duration=280,
    loop=0,
    optimize=True,
)
print(f"Successfully generated new demo gif at: {out_path} ({os.path.getsize(out_path)} bytes)")
