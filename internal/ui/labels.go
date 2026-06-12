package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
	"go-chrome/internal/textutil"
)

// truncate is a thin wrapper around textutil.Truncate for use inside the
// UI package. Kept as a local name to keep call sites short.
//
// Note: in table cells and narrow list rows we now prefer
// newTruncatingLabel() (which uses Fyne's TextTruncateEllipsis) over
// manual truncation, so the cut adapts to the actual widget width
// regardless of font/DPI. truncate() is still useful for fixed-width
// contexts (e.g. status bar fields) where the available width is
// deterministic.
func truncate(s string, max int) string {
	return textutil.Truncate(s, max)
}

// newTruncatingLabel returns a widget.Label that visually truncates with
// an ellipsis ("…") when its text exceeds the available width, instead
// of drawing past the cell boundary.
//
// This avoids the previous "tofu / overflow" bug where long Chinese
// step names, XPaths, or file paths rendered beyond the column or
// list-row width and either overlapped neighbouring cells or were
// clipped by the window edge.
//
// Fyne's Truncation field (Since: 2.4) handles the cut on the render
// thread, so the displayed substring is chosen to fit the actual
// measured cell width and the system font metrics.
func newTruncatingLabel(initial string) *widget.Label {
	l := widget.NewLabel(initial)
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

var stepTypeOptions = []string{
	"打开网址",
	"点击元素",
	"输入文本",
	"清空后输入",
	"等待元素出现",
	"等待元素可见",
	"固定等待",
	"获取元素文本",
	"断言元素存在",
	"断言文本包含",
	"页面截图",
}

var stepTypeToLabel = map[flow.StepType]string{
	flow.StepNavigate:      "打开网址",
	flow.StepClick:         "点击元素",
	flow.StepInput:         "输入文本",
	flow.StepClearAndInput: "清空后输入",
	flow.StepWaitPresent:   "等待元素出现",
	flow.StepWaitVisible:   "等待元素可见",
	flow.StepWaitFixed:     "固定等待",
	flow.StepGetText:       "获取元素文本",
	flow.StepAssertExists:  "断言元素存在",
	flow.StepAssertText:    "断言文本包含",
	flow.StepScreenshot:    "页面截图",
}

var labelToStepType = map[string]flow.StepType{
	"打开网址":   flow.StepNavigate,
	"点击元素":   flow.StepClick,
	"输入文本":   flow.StepInput,
	"清空后输入":  flow.StepClearAndInput,
	"等待元素出现": flow.StepWaitPresent,
	"等待元素可见": flow.StepWaitVisible,
	"固定等待":   flow.StepWaitFixed,
	"获取元素文本": flow.StepGetText,
	"断言元素存在": flow.StepAssertExists,
	"断言文本包含": flow.StepAssertText,
	"页面截图":   flow.StepScreenshot,
}

var errorPolicyOptions = []string{
	"失败后停止",
	"失败后继续",
	"失败后重试",
}

var errorPolicyToLabel = map[flow.ErrorPolicy]string{
	flow.ErrStop:     "失败后停止",
	flow.ErrContinue: "失败后继续",
	flow.ErrRetry:    "失败后重试",
}

var labelToErrorPolicy = map[string]flow.ErrorPolicy{
	"失败后停止": flow.ErrStop,
	"失败后继续": flow.ErrContinue,
	"失败后重试": flow.ErrRetry,
}

func stepTypeLabel(t flow.StepType) string {
	if label, ok := stepTypeToLabel[t]; ok {
		return label
	}
	return string(t)
}

func stepTypeFromLabel(label string) flow.StepType {
	if t, ok := labelToStepType[label]; ok {
		return t
	}
	return flow.StepType(label)
}

func errorPolicyLabel(p flow.ErrorPolicy) string {
	if label, ok := errorPolicyToLabel[p]; ok {
		return label
	}
	return string(p)
}

func errorPolicyFromLabel(label string) flow.ErrorPolicy {
	if p, ok := labelToErrorPolicy[label]; ok {
		return p
	}
	return flow.ErrorPolicy(label)
}
