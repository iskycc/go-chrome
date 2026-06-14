package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
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
}

func TestStatusItemSetValue(t *testing.T) {
	si := newStatusItem("字段", "默认值", statusSuccess, 100)
	si.setValue("新值")
	if si.value.Text != "新值" {
		t.Fatalf("expected value '新值', got %q", si.value.Text)
	}
}

func TestStatusItemSetKind(t *testing.T) {
	si := newStatusItem("字段", "值", statusSuccess, 100)
	si.setKind(statusDanger)

	center, ok := si.row.(*fyne.Container)
	if !ok || len(center.Objects) == 0 {
		t.Fatal("expected center container with hbox")
	}
	hbox, ok := center.Objects[0].(*fyne.Container)
	if !ok || len(hbox.Objects) == 0 {
		t.Fatal("expected hbox inside center")
	}
	// Dot is now the last object.
	last := hbox.Objects[len(hbox.Objects)-1]
	if last != si.dot {
		t.Fatal("expected last object to be the updated dot")
	}
}

func TestStatusItemSetColor(t *testing.T) {
	si := newStatusItem("字段", "值", statusSuccess, 100)
	newColor := color.NRGBA{R: 0xff, A: 0xff}
	si.setColor(newColor)

	center, ok := si.row.(*fyne.Container)
	if !ok || len(center.Objects) == 0 {
		t.Fatal("expected center container with hbox")
	}
	hbox, ok := center.Objects[0].(*fyne.Container)
	if !ok || len(hbox.Objects) == 0 {
		t.Fatal("expected hbox inside center")
	}
	last := hbox.Objects[len(hbox.Objects)-1]
	dotWrap, ok := last.(*fyne.Container)
	if !ok || len(dotWrap.Objects) == 0 {
		t.Fatal("expected dot wrapper container")
	}
	circle, ok := dotWrap.Objects[0].(*canvas.Circle)
	if !ok {
		t.Fatalf("expected canvas.Circle, got %T", dotWrap.Objects[0])
	}
	if circle.FillColor != newColor {
		t.Fatalf("expected dot color %v, got %v", newColor, circle.FillColor)
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

func TestStatusItemSetKind_InvalidRow(t *testing.T) {
	si := &statusItem{row: widget.NewLabel("not a center container")}
	// Should not panic on unexpected row structure.
	si.setKind(statusSuccess)
}

func TestStatusItemSetColor_InvalidRow(t *testing.T) {
	si := &statusItem{row: widget.NewLabel("not a center container")}
	// Should not panic on unexpected row structure.
	si.setColor(color.NRGBA{R: 0xff, A: 0xff})
}
