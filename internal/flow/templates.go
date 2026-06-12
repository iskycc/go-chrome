package flow

// FlowTemplate describes a built-in flow template the user can create from
// the UI. It is intentionally kept simple and offline-friendly; remote
// template catalogs are not part of the built-in set.
type FlowTemplate struct {
	ID          string
	Name        string
	Description string
	Tags        []string
	// Factory returns a fresh flow (with new IDs and timestamps) every
	// time it is invoked, so the same template can be reused without
	// clobbering prior creations.
	Factory func() *Flow
}

// builtinTemplates is the registry of templates available out of the box.
// New entries are added here as the catalog grows.
var builtinTemplates = []FlowTemplate{
	{
		ID:          "login-basic",
		Name:        "登录测试",
		Description: "打开登录页，输入用户名密码并断言欢迎文案（需要本地演示服务器 http://localhost:18080）",
		Tags:        []string{"登录", "示例"},
		Factory:     newExampleLoginFlowFactory,
	},
	{
		ID:          "blank",
		Name:        "空白流程",
		Description: "仅含一个空白的打开网址步骤，其余步骤自行添加",
		Tags:        []string{"入门"},
		Factory:     newBlankFlowFactory,
	},
	{
		ID:          "form-fill",
		Name:        "表单填写",
		Description: "导航到一个 URL，依次填入两个文本框，最后截图",
		Tags:        []string{"表单"},
		Factory:     newFormFillFlowFactory,
	},
	{
		ID:          "text-assertion",
		Name:        "页面文本断言",
		Description: "导航到一个 URL，断言页面包含指定文本，并截图留档",
		Tags:        []string{"断言"},
		Factory:     newTextAssertionFlowFactory,
	},
}

// ListBuiltinTemplates returns a defensive copy of the built-in template
// catalog so callers can iterate it without affecting the registry.
func ListBuiltinTemplates() []FlowTemplate {
	out := make([]FlowTemplate, len(builtinTemplates))
	copy(out, builtinTemplates)
	return out
}

// FindBuiltinTemplate returns the template with the given ID, or false if
// the ID is not part of the built-in catalog.
func FindBuiltinTemplate(id string) (FlowTemplate, bool) {
	for _, t := range builtinTemplates {
		if t.ID == id {
			return t, true
		}
	}
	return FlowTemplate{}, false
}

func newExampleLoginFlowFactory() *Flow {
	return NewExampleLoginFlow()
}

func newBlankFlowFactory() *Flow {
	f := NewFlow("新建流程（空白模板）")
	f.Tags = []string{"空白"}
	s := NewStep("打开网址", StepNavigate)
	s.Target = Target{Strategy: TargetXPath, Value: "https://example.com"}
	f.Steps = []Step{s}
	return f
}

func newFormFillFlowFactory() *Flow {
	f := NewFlow("新建流程（表单填写模板）")
	f.Tags = []string{"表单"}
	f.Steps = []Step{
		NewStep("打开网址", StepNavigate),
		NewStep("输入字段 1", StepInput),
		NewStep("输入字段 2", StepInput),
		NewStep("页面截图", StepScreenshot),
	}
	f.Steps[0].Target = Target{Strategy: TargetXPath, Value: "https://example.com/form"}
	f.Steps[1].Target = Target{Strategy: TargetXPath, Value: "//input[@name='field1']"}
	f.Steps[1].Input = Input{Mode: InputLiteral, Text: "field1-value"}
	f.Steps[2].Target = Target{Strategy: TargetXPath, Value: "//input[@name='field2']"}
	f.Steps[2].Input = Input{Mode: InputLiteral, Text: "field2-value"}
	return f
}

func newTextAssertionFlowFactory() *Flow {
	f := NewFlow("新建流程（页面文本断言模板）")
	f.Tags = []string{"断言"}
	f.Steps = []Step{
		NewStep("打开网址", StepNavigate),
		NewStep("断言页面文本", StepAssertText),
		NewStep("页面截图", StepScreenshot),
	}
	f.Steps[0].Target = Target{Strategy: TargetXPath, Value: "https://example.com"}
	f.Steps[1].Target = Target{Strategy: TargetXPath, Value: "//body"}
	return f
}
