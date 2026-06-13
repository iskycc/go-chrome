package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"go-chrome/internal/flow"
)

// showTemplatePickerDialog presents the built-in template catalog. The user
// picks a template, the factory produces a fresh flow, the flow is saved,
// and the user is taken to the new flow in the editor.
//
// The dialog runs entirely offline; it never touches the network.
func (a *App) showTemplatePickerDialog() {
	templates := flow.ListBuiltinTemplates()
	if len(templates) == 0 {
		dialog.ShowInformation("提示", "没有可用的内置模板", a.mainWin)
		return
	}

	// Track the current selection in a closure-shared variable since
	// fyne's widget.List does not expose a SelectedID() getter.
	selectedID := -1
	previewLines := []string{}

	// Right side: detail panel that updates when the user picks a template.
	descLabel := widget.NewLabel("请在左侧选择一个模板")
	descLabel.Wrapping = fyne.TextWrapWord
	previewHeader := widget.NewLabelWithStyle("步骤预览", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	previewList := widget.NewList(
		func() int { return len(previewLines) },
		func() fyne.CanvasObject { return newTruncatingLabel("") },
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(previewLines) {
				return
			}
			item.(*widget.Label).SetText(previewLines[id])
		},
	)

	detail := container.NewVBox(
		widget.NewLabelWithStyle("说明", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		descLabel,
		previewHeader,
		previewList,
	)

	// Left side: template list.
	list := widget.NewList(
		func() int { return len(templates) },
		func() fyne.CanvasObject { return newTruncatingLabel("") },
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(templates) {
				return
			}
			t := templates[id]
			label := t.Name
			if len(t.Tags) > 0 {
				label += " [" + strings.Join(t.Tags, ", ") + "]"
			}
			item.(*widget.Label).SetText(label)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(templates) {
			return
		}
		selectedID = int(id)
		t := templates[id]
		descLabel.SetText(fmt.Sprintf("%s\n\n标签：%s", t.Description, strings.Join(t.Tags, ", ")))
		lines := make([]string, 0)
		if t.Factory != nil {
			sample := t.Factory()
			for i, s := range sample.Steps {
				lines = append(lines, fmt.Sprintf("%d. %s (%s)", i+1, s.Name, s.Type))
			}
		}
		previewLines = lines
		previewList.Refresh()
	}
	list.OnUnselected = func(id widget.ListItemID) {
		if int(id) == selectedID {
			selectedID = -1
		}
	}

	leftPanel := container.NewBorder(
		widget.NewLabelWithStyle("内置模板", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		list,
	)
	split := container.NewHSplit(leftPanel, detail)
	split.SetOffset(0.4)

	createBtn := widget.NewButtonWithIcon("创建流程", theme.ContentAddIcon(), nil)
	createBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton("取消", nil)

	d := dialog.NewCustomWithoutButtons("从模板创建", split, a.mainWin)
	d.Resize(fyne.NewSize(720, 420))
	d.SetButtons([]fyne.CanvasObject{cancelBtn, createBtn})

	createBtn.OnTapped = func() {
		if selectedID < 0 || selectedID >= len(templates) {
			dialog.ShowInformation("提示", "请先选择一个模板", a.mainWin)
			return
		}
		t := templates[selectedID]
		if t.Factory == nil {
			dialog.ShowError(fmt.Errorf("模板 [%s] 没有定义工厂函数", t.Name), a.mainWin)
			return
		}
		newFlow := t.Factory()
		if err := a.flowStore.Save(newFlow); err != nil {
			dialog.ShowError(err, a.mainWin)
			return
		}
		d.Hide()
		a.refreshFlowList()
		a.flowLibrary.selectFlow(newFlow.ID)
	}
	cancelBtn.OnTapped = func() { d.Hide() }

	d.Show()
}
