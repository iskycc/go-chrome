# Taskbar Icon Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current `assets/icon.png` with a new browser-window + automation-flow icon, add a multi-size `assets/icon.ico`, and make the Windows build use the `.ico` for the taskbar while keeping PNG fallback for other platforms.

**Architecture:** A Python/Pillow generator script (`scripts/generate-icon.py`) produces both the 256×256 PNG and the multi-size ICO from the same vector-like drawing code. The `assets` package exposes platform-specific `AppIcon()` (`.ico` on Windows, `.png` elsewhere) via build-tag files. `internal/ui/main_window.go` switches from `assets.Icon()` to `assets.AppIcon()`.

**Tech Stack:** Go, Fyne v2, Python 3, Pillow, ImageMagick (fallback).

---

## Task 1: Create the Icon Generation Script

**Files:**
- Create: `scripts/generate-icon.py`

Generate the icon in four sizes (16, 32, 48, 256) with the design from `docs/superpowers/specs/2026-06-15-taskbar-icon-design.md`.

- [ ] **Step 1: Write `scripts/generate-icon.py`**

```python
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
    images[0].save(
        OUTPUT_ICO,
        format="ICO",
        append_images=images[1:],
    )

    print(f"Generated: {OUTPUT_PNG}")
    print(f"Generated: {OUTPUT_ICO}")


if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Make the script executable**

```bash
chmod +x scripts/generate-icon.py
```

- [ ] **Step 3: Commit the script**

```bash
git add scripts/generate-icon.py
git commit -m "chore: add icon generation script"
```

---

## Task 2: Generate the Icon Assets

**Files:**
- Create/Overwrite: `assets/icon.png`
- Create: `assets/icon.ico`

- [ ] **Step 1: Run the generator**

```bash
python3 scripts/generate-icon.py
```

Expected output:

```
Generated: /opt/go-chrome/assets/icon.png
Generated: /opt/go-chrome/assets/icon.ico
```

- [ ] **Step 2: Verify the ICO contains all sizes**

```bash
python3 - <<'PY'
from PIL import Image
ico = Image.open('assets/icon.ico')
print('ICO sizes:', [s for s in Image.open('assets/icon.ico').info.get('sizes', [])])
PY
```

Expected: `ICO sizes: [(16, 16), (32, 32), (48, 48), (256, 256)]` (order may vary).

- [ ] **Step 3: Inspect the 256×256 PNG**

Open `assets/icon.png` to confirm the design matches the spec: blue-cyan rounded-square background, white browser frame, three connected nodes, last node cyan.

- [ ] **Step 4: Commit the generated assets**

```bash
git add assets/icon.png assets/icon.ico
git commit -m "assets: generate new browser-flow taskbar icon"
```

---

## Task 3: Expose Platform-Specific App Icon

**Files:**
- Modify: `assets/embed.go`
- Create: `assets/appicon_windows.go`
- Create: `assets/appicon_other.go`

- [ ] **Step 1: Update `assets/embed.go` to embed `icon.ico` and expose `IconICO()`**

```go
package assets

import (
	"embed"
	"sync"

	"fyne.io/fyne/v2"
)

//go:embed icon.png icon.ico fonts/MapleMono-CN-Regular.ttf
var assetFS embed.FS

var (
	iconOnce    sync.Once
	iconRes     fyne.Resource
	iconICOOnce sync.Once
	iconICORes  fyne.Resource

	fontOnce sync.Once
	fontRes  fyne.Resource
)

// Icon returns the PNG application icon.
func Icon() fyne.Resource {
	iconOnce.Do(func() {
		data, err := assetFS.ReadFile("icon.png")
		if err != nil {
			return
		}
		iconRes = fyne.NewStaticResource("icon.png", data)
	})
	return iconRes
}

// IconICO returns the Windows ICO application icon.
func IconICO() fyne.Resource {
	iconICOOnce.Do(func() {
		data, err := assetFS.ReadFile("icon.ico")
		if err != nil {
			return
		}
		iconICORes = fyne.NewStaticResource("icon.ico", data)
	})
	return iconICORes
}

func AppUIFontRegular() fyne.Resource {
	fontOnce.Do(func() {
		data, err := assetFS.ReadFile("fonts/MapleMono-CN-Regular.ttf")
		if err != nil {
			return
		}
		fontRes = fyne.NewStaticResource("MapleMono-CN-Regular.ttf", data)
	})
	return fontRes
}
```

- [ ] **Step 2: Create `assets/appicon_windows.go`**

```go
//go:build windows

package assets

import "fyne.io/fyne/v2"

// AppIcon returns the ICO resource on Windows for best taskbar rendering.
func AppIcon() fyne.Resource {
	return IconICO()
}
```

- [ ] **Step 3: Create `assets/appicon_other.go`**

```go
//go:build !windows

package assets

import "fyne.io/fyne/v2"

// AppIcon returns the PNG resource on non-Windows platforms.
func AppIcon() fyne.Resource {
	return Icon()
}
```

- [ ] **Step 4: Build the assets package**

```bash
go build ./assets
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add assets/embed.go assets/appicon_windows.go assets/appicon_other.go
git commit -m "feat(assets): expose platform-specific AppIcon"
```

---

## Task 4: Use `assets.AppIcon()` in the Main Window

**Files:**
- Modify: `internal/ui/main_window.go:168`

- [ ] **Step 1: Replace the icon assignment**

Old code:

```go
	if ico := assets.Icon(); ico != nil {
		a.fyneApp.SetIcon(ico)
	}
```

New code:

```go
	if ico := assets.AppIcon(); ico != nil {
		a.fyneApp.SetIcon(ico)
	}
```

- [ ] **Step 2: Verify the change compiles**

```bash
go build ./internal/ui
```

Note: On a headless Linux environment this may fail due to missing Fyne/GLFW system libraries. If it fails for that reason, run `go build ./assets` and `go vet ./internal/ui` instead to catch Go-level errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/main_window.go
git commit -m "feat(ui): use platform-specific AppIcon for taskbar"
```

---

## Task 5: Final Verification and Push

- [ ] **Step 1: Verify generated files are present**

```bash
ls -lh assets/icon.png assets/icon.ico
```

Expected: both files exist and are non-empty.

- [ ] **Step 2: Check git status**

```bash
git status --short
```

Expected: no unexpected untracked files (e.g., no screenshot/HTML artifacts from tests).

- [ ] **Step 3: Push**

```bash
git push
```

---

## Spec Coverage Check

| Spec Requirement | Implementing Task |
|---|---|
| New 256×256 PNG with browser + flow design | Task 2 |
| Multi-size `.ico` (16/32/48/256) | Task 2 |
| Embed `.ico` in Go binary | Task 3 |
| Windows build uses `.ico` | Task 3 + Task 4 |
| Non-Windows fallback to PNG | Task 3 |
| Icon visible in Windows taskbar | Task 4 (verified on Windows) |

## Placeholder Scan

No TBD/TODO placeholders. All code blocks are complete and runnable.
