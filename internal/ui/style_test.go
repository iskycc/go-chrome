package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func TestNewSectionHeader_NoActions(t *testing.T) {
	obj := newSectionHeader("标题")
	if obj == nil {
		t.Fatal("expected non-nil object")
	}
	label, ok := obj.(*widget.Label)
	if !ok {
		t.Fatalf("expected *widget.Label without actions, got %T", obj)
	}
	if label.Text != "标题" {
		t.Fatalf("expected text '标题', got %q", label.Text)
	}
}

func TestNewSectionHeader_WithActions(t *testing.T) {
	btn := widget.NewButton("test", nil)
	obj := newSectionHeader("标题", btn)
	if obj == nil {
		t.Fatal("expected non-nil object")
	}
	// With actions the result is wrapped in a padded border layout.
	padded, ok := obj.(*fyne.Container)
	if !ok {
		t.Fatalf("expected padded container, got %T", obj)
	}
	if len(padded.Objects) == 0 {
		t.Fatal("expected at least one object in padded container")
	}
}

func TestNewToolbarCard(t *testing.T) {
	content := widget.NewLabel("content")
	card := newToolbarCard(content)
	if card == nil {
		t.Fatal("expected non-nil card")
	}
	stack, ok := card.(*fyne.Container)
	if !ok {
		t.Fatalf("expected stack container, got %T", card)
	}
	if len(stack.Objects) < 3 {
		t.Fatalf("expected background, border and content, got %d objects", len(stack.Objects))
	}
}

func TestNewStatusBarCard(t *testing.T) {
	content := widget.NewLabel("content")
	card := newStatusBarCard(content)
	if card == nil {
		t.Fatal("expected non-nil card")
	}
	stack, ok := card.(*fyne.Container)
	if !ok {
		t.Fatalf("expected stack container, got %T", card)
	}
	if len(stack.Objects) < 3 {
		t.Fatalf("expected background, border and content, got %d objects", len(stack.Objects))
	}
}

func TestNewVerticalSeparator(t *testing.T) {
	sep := newVerticalSeparator()
	if sep == nil {
		t.Fatal("expected non-nil separator")
	}
	min := sep.MinSize()
	if min.Width != 1 {
		t.Fatalf("expected separator width 1, got %f", min.Width)
	}
	if min.Height < 1 {
		t.Fatalf("expected separator height > 0, got %f", min.Height)
	}
}

func TestNewInlineToolbarGroup(t *testing.T) {
	btn := widget.NewButton("btn", nil)
	obj := newInlineToolbarGroup("组", btn)
	if obj == nil {
		t.Fatal("expected non-nil object")
	}
	center, ok := obj.(*fyne.Container)
	if !ok {
		t.Fatalf("expected center container, got %T", obj)
	}
	if len(center.Objects) == 0 {
		t.Fatal("expected inner container in center")
	}
}

func TestNewStatusBadge(t *testing.T) {
	badge := newStatusBadge("ok", statusSuccess)
	if badge == nil {
		t.Fatal("expected non-nil badge")
	}
	_, ok := badge.(*fyne.Container)
	if !ok {
		t.Fatalf("expected container badge, got %T", badge)
	}
}

func TestNewStatusDot(t *testing.T) {
	dot := newStatusDot(statusInfo)
	if dot == nil {
		t.Fatal("expected non-nil dot")
	}
}
