#!/usr/bin/env python3
"""Generate go-chrome taskbar icons from the project design spec."""

from pathlib import Path

from PIL import Image, ImageDraw

SIZE = 256
RADIUS = 48
BG_START = (37, 99, 235)   # #2563EB
BG_END = (14, 165, 233)    # #0EA5E9
WHITE = (255, 255, 255)
CYAN = (103, 232, 249)     # #67E8F9

PROJECT_ROOT = Path(__file__).resolve().parent.parent
OUTPUT_PNG = PROJECT_ROOT / "assets" / "icon.png"
OUTPUT_ICO = PROJECT_ROOT / "assets" / "icon.ico"


def gradient(size: int, c1, c2) -> Image.Image:
    base = Image.new("RGBA", (size, size))
    draw = ImageDraw.Draw(base)
    for y in range(size):
        ratio = y / (size - 1)
        r = int(c1[0] + (c2[0] - c1[0]) * ratio)
        g = int(c1[1] + (c2[1] - c1[1]) * ratio)
        b = int(c1[2] + (c2[2] - c1[2]) * ratio)
        draw.line([(0, y), (size, y)], fill=(r, g, b, 255))
    return base


def draw_icon(size: int) -> Image.Image:
    img = gradient(size, BG_START, BG_END)
    draw = ImageDraw.Draw(img)

    pad = max(size // 8, 4)
    frame_left = pad
    frame_top = int(pad * 1.2)
    frame_right = size - pad
    frame_bottom = size - pad
    corner_radius = max(size // 12, 2)
    stroke = max(size // 64, 1)

    # Browser window frame
    draw.rounded_rectangle(
        [(frame_left, frame_top), (frame_right, frame_bottom)],
        radius=corner_radius,
        outline=WHITE,
        width=stroke,
    )

    # Address/title bar line
    bar_y = frame_top + max(size // 16, 4)
    draw.line(
        [(frame_left + corner_radius, bar_y), (frame_right - corner_radius, bar_y)],
        fill=WHITE,
        width=stroke,
    )

    # Flow nodes and connecting line
    y_center = (bar_y + frame_bottom) // 2
    node_r = max(size // 22, 2)
    x_positions = [
        int(frame_left + size * 0.22),
        size // 2,
        int(frame_right - size * 0.22),
    ]

    draw.line(
        [(x_positions[0], y_center), (x_positions[1], y_center), (x_positions[2], y_center)],
        fill=WHITE,
        width=max(size // 48, 1),
    )

    for i, x in enumerate(x_positions):
        color = CYAN if i == len(x_positions) - 1 else WHITE
        draw.ellipse(
            [(x - node_r, y_center - node_r), (x + node_r, y_center + node_r)],
            fill=color,
        )

    return img


def main() -> None:
    sizes = [16, 32, 48, 256]
    images = [draw_icon(s) for s in sizes]

    # Save high-resolution PNG
    images[-1].save(OUTPUT_PNG, "PNG")

    # Save multi-size ICO (Windows taskbar needs 16/32/48)
    # Use the largest image as the save base so Pillow's default size
    # filter does not drop the larger frames.
    images[-1].save(
        OUTPUT_ICO,
        format="ICO",
        sizes=[(s, s) for s in sizes],
        append_images=images[:-1],
    )

    print(f"Generated: {OUTPUT_PNG}")
    print(f"Generated: {OUTPUT_ICO}")


if __name__ == "__main__":
    main()
