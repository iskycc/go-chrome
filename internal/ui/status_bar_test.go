package ui

import (
	"image/color"
	"testing"
)

func TestNewStatusItem(t *testing.T) {
	si := newStatusItem("字段", "默认值", statusSuccess, 100)
	if si == nil {
		t.Fatal("expected non-nil statusItem")
	}
	if si.row == nil {
		t.Fatal("expected non-nil row")
	}
	if si.value.Text != "默认值" {
		t.Fatalf("expected default value '默认值', got %q", si.value.Text)
	}
	if si.dot == nil {
		t.Fatal("expected dot circle")
	}
}

func TestStatusItemSetValue(t *testing.T) {
	si := newStatusItem("字段", "默认值", statusSuccess, 100)
	si.setValue("新值")
	if si.value.Text != "新值" {
		t.Fatalf("expected value '新值', got %q", si.value.Text)
	}

	// Setting the same value again should be a no-op.
	si.setValue("新值")
	if si.value.Text != "新值" {
		t.Fatalf("expected value unchanged, got %q", si.value.Text)
	}
}

func TestStatusItemSetKind(t *testing.T) {
	si := newStatusItem("字段", "值", statusSuccess, 100)
	oldDot := si.dot
	si.setKind(statusDanger)

	if si.dot != oldDot {
		t.Fatal("expected dot circle to be reused, not replaced")
	}
	if si.dot.FillColor != uiColorForStatus(statusDanger) {
		t.Fatalf("expected dot color updated to danger, got %v", si.dot.FillColor)
	}
}

func TestStatusItemSetColor(t *testing.T) {
	si := newStatusItem("字段", "值", statusSuccess, 100)
	newColor := color.NRGBA{R: 0xff, A: 0xff}
	si.setColor(newColor)
	if si.dot.FillColor != newColor {
		t.Fatalf("expected dot color %v, got %v", newColor, si.dot.FillColor)
	}
}

func TestNewStatusBar(t *testing.T) {
	app := &App{}
	sb := newStatusBar(app)
	if sb == nil {
		t.Fatal("expected non-nil statusBar")
	}
	if sb.widget == nil {
		t.Fatal("expected non-nil widget")
	}
	if sb.save == nil || sb.chrome == nil || sb.run == nil {
		t.Fatal("expected status items to be initialized")
	}
}

func TestStatusBarSetFlow(t *testing.T) {
	app := &App{}
	sb := newStatusBar(app)
	sb.setFlow("测试流程")
	if sb.flow.value.Text != "测试流程" {
		t.Fatalf("expected flow value '测试流程', got %q", sb.flow.value.Text)
	}
}

func TestStatusBarSetSave(t *testing.T) {
	app := &App{}
	sb := newStatusBar(app)
	sb.setSave(SaveDirty)
	if sb.save.value.Text != "有未保存修改" {
		t.Fatalf("expected save value '有未保存修改', got %q", sb.save.value.Text)
	}
}

func TestStatusBarSetRun(t *testing.T) {
	app := &App{}
	sb := newStatusBar(app)
	sb.setRun(RunRunning, 2, 5, "")
	if sb.run.value.Text != "运行中 2/5" {
		t.Fatalf("expected run value '运行中 2/5', got %q", sb.run.value.Text)
	}
}

func TestStatusItemSetKind_NoDot(t *testing.T) {
	si := &statusItem{dot: nil}
	// Should not panic when dot is nil.
	si.setKind(statusSuccess)
}

func TestStatusItemSetColor_NoDot(t *testing.T) {
	si := &statusItem{dot: nil}
	// Should not panic when dot is nil.
	si.setColor(color.NRGBA{R: 0xff, A: 0xff})
}

func TestStatusItemSetValue_Unchanged(t *testing.T) {
	si := newStatusItem("字段", "值", statusSuccess, 100)
	// setValue should skip when text matches.
	si.setValue("值")
	if si.value.Text != "值" {
		t.Fatalf("expected unchanged value, got %q", si.value.Text)
	}
}
