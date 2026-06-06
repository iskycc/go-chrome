// Package ui i18n support.
// This file provides a lightweight i18n interface for future Chinese/English expansion.
// All UI strings currently use Chinese as the default language.
// To add a new language, implement the Translator interface and set it via SetTranslator.

package ui

// Translator provides localized strings for the UI.
type Translator interface {
	T(key string) string
}

// defaultTranslator returns Chinese strings (current default).
type defaultTranslator struct{}

func (t *defaultTranslator) T(key string) string {
	if v, ok := zhCN[key]; ok {
		return v
	}
	return key
}

// zhCN holds the default Chinese translations.
var zhCN = map[string]string{
	"app.title":               "Chrome 自动化编排工具",
	"status.no_flow":          "未选择流程",
	"status.unmodified":       "未修改",
	"status.dirty":            "有未保存修改",
	"status.saving":           "保存中",
	"status.saved":            "已保存",
	"status.save_failed":      "保存失败",
	"status.chrome.not_installed": "未安装",
	"status.chrome.installed":     "已安装",
	"status.chrome.downloading":   "下载中",
	"status.chrome.starting":      "启动中",
	"status.chrome.running":       "已启动",
	"status.chrome.start_failed":  "启动失败",
	"status.run.idle":         "空闲",
	"status.run.running":      "运行中",
	"status.run.completed":    "已完成",
	"status.run.failed":       "失败",
	"flow.new":                "新建流程",
	"flow.save":               "保存",
	"flow.import":             "导入",
	"flow.export":             "导出",
	"flow.clone":              "复制",
	"flow.delete":             "删除",
	"step.new":                "新增步骤",
	"step.copy":               "复制步骤",
	"step.delete":             "删除步骤",
	"step.move_up":            "上移",
	"step.move_down":          "下移",
	"step.apply":              "应用到当前步骤",
	"run.start_browser":       "启动浏览器",
	"run.run_flow":            "运行整个流程",
	"run.step":                "单步执行",
	"run.stop":                "停止",
	"dialog.confirm_delete":   "确认删除",
	"dialog.unsaved_changes":  "未保存的修改",
	"dialog.save_before_switch": "当前流程 [%s] 有未保存的修改，是否保存？",
}

var currentTranslator Translator = &defaultTranslator{}

// SetTranslator sets the global translator.
func SetTranslator(t Translator) {
	currentTranslator = t
}

// Tr returns the localized string for the given key.
func Tr(key string) string {
	return currentTranslator.T(key)
}
