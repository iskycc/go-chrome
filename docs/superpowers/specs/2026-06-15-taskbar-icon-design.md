# Taskbar Icon Design Spec

## Background

`go-chrome` is a Windows GUI automation tool built with Fyne. The current application icon is `assets/icon.png` (256×256, blue background with a white letter "C"). Users report that the application has **no visible icon in the Windows taskbar**, which makes it hard to identify when pinned or running.

Fyne sets the application icon via `fyneApp.SetIcon()` in `internal/ui/main_window.go`. While Fyne accepts a PNG resource, Windows taskbar icons are best rendered from a multi-size `.ico` file. This spec describes a new icon and the integration changes needed to make it visible on the taskbar.

## Goals

1. Provide a clear, recognizable taskbar icon for `go-chrome`.
2. Ensure the icon remains readable at small sizes (16×16, 32×32, 48×48).
3. Avoid direct reuse of Chrome trademarks; instead communicate "browser automation" visually.
4. Deliver both a high-resolution PNG and a Windows `.ico` with multiple sizes.

## Non-Goals

- Redesigning in-app icons or toolbar glyphs.
- Changing the application name or branding beyond the icon.
- Supporting macOS/Linux `.icns` formats (out of scope for this task).

## Design

### Visual Concept

**Browser window + automation flow nodes.**

- A rounded-square canvas with a subtle blue-to-cyan gradient background.
- A simplified browser-window frame (rounded rectangle with a thin title/address bar line at the top).
- Inside the window, three circular nodes connected by a polyline path, flowing left-to-right, representing an automated workflow running inside the browser.
- The final node is slightly brighter to suggest execution completion.

### Color Palette

| Element | Color | Hex |
|---|---|---|
| Background gradient start | Deep blue | `#2563EB` |
| Background gradient end | Cyan | `#0EA5E9` |
| Window frame / flow lines | White | `#FFFFFF` |
| Primary nodes | White | `#FFFFFF` |
| Endpoint highlight | Bright cyan | `#67E8F9` |

### Size Variants

| Size | Usage |
|---|---|
| 16×16 | Windows taskbar (small icons) |
| 32×32 | Taskbar default, alt-tab |
| 48×48 | Start menu, file explorer |
| 256×256 | Fyne app icon, high-DPI displays |

## Files to Change

1. **`assets/icon.png`** — replace with the new 256×256 PNG.
2. **`assets/icon.ico`** — new multi-size Windows icon (16/32/48/256).
3. **`assets/embed.go`** — embed `icon.ico` in addition to `icon.png`.
4. **`internal/ui/main_window.go`** — on Windows, call `SetIcon()` with the `.ico` resource; fall back to PNG on other platforms.
5. **`docs/superpowers/specs/2026-06-15-taskbar-icon-design.md`** — this document.

## Implementation Notes

- The `.ico` file should be generated from the same source artwork to keep sizes consistent.
- The PNG should remain as a cross-platform fallback and for documentation.
- Because `go-chrome` is Windows-focused, the `.ico` should be used as the primary application icon.
- The icon graphic must use simple strokes and large enough shapes so that the 16×16 version is still distinguishable.

## Acceptance Criteria

- [ ] `assets/icon.ico` exists and contains 16, 32, 48, and 256 pixel sizes.
- [ ] `assets/icon.png` is replaced with the new design at 256×256.
- [ ] Windows build sets the `.ico` resource as the application icon.
- [ ] The icon is visible in the Windows taskbar when the application is running.
- [ ] PNG fallback remains functional on non-Windows builds.
