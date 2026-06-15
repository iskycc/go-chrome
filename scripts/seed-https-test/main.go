// seed-https-test inserts a flow that exercises the HTTPS self-signed
// login flow against the local test-server, and marks it as the
// most-recently-used flow so go-chrome auto-opens it on next launch.
//
// Usage:
//
//	go run ./scripts/seed-https-test
//
// Requires the test-server to be running on :18080/:18443.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go-chrome/internal/db"
	"go-chrome/internal/flow"
)

func main() {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	dbPath := filepath.Join(appData, "go-chrome", "go-chrome.db")

	sqliteDB, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	fs, err := db.NewFlowStore(sqliteDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "store:", err)
		os.Exit(1)
	}

	all, _ := fs.List()
	for _, f := range all {
		if f.Name == "HTTPS 自签证书登录测试" {
			_ = fs.Delete(f.ID)
		}
	}

	f := flow.NewFlow("HTTPS 自签证书登录测试")
	f.Steps = []flow.Step{
		func() flow.Step {
			s := flow.NewStep("打开 HTTPS 自签登录页", flow.StepNavigate)
			s.Input = flow.Input{
				Mode: flow.InputLiteral,
				Text: "https://localhost:18443/",
			}
			return s
		}(),
		func() flow.Step {
			s := flow.NewStep("输入用户名", flow.StepInput)
			s.Target = flow.Target{
				Strategy: flow.TargetXPath,
				Value:    "//input[@id='username']",
			}
			s.Input = flow.Input{
				Mode: flow.InputTemplate,
				Text: "SP${11000-11099}",
			}
			return s
		}(),
		func() flow.Step {
			s := flow.NewStep("输入密码", flow.StepInput)
			s.Target = flow.Target{
				Strategy: flow.TargetXPath,
				Value:    "//input[@id='password']",
			}
			s.Input = flow.Input{
				Mode: flow.InputLiteral,
				Text: "Password123",
			}
			return s
		}(),
		func() flow.Step {
			s := flow.NewStep("点击登录按钮", flow.StepClick)
			s.Target = flow.Target{
				Strategy: flow.TargetXPath,
				Value:    "//button[@type='submit']",
			}
			return s
		}(),
		func() flow.Step {
			s := flow.NewStep("断言欢迎文本", flow.StepAssertText)
			s.Target = flow.Target{
				Strategy: flow.TargetXPath,
				Value:    "//div[contains(text(),'欢迎')]",
			}
			s.Input = flow.Input{
				Mode: flow.InputLiteral,
				Text: "欢迎，SP",
			}
			return s
		}(),
	}

	if err := fs.Save(f); err != nil {
		fmt.Fprintln(os.Stderr, "save:", err)
		os.Exit(1)
	}
	fmt.Printf("Saved flow id=%s name=%q with %d steps\n", f.ID, f.Name, len(f.Steps))

	recent := db.NewRecentRepo(sqliteDB)
	if err := recent.Save([]string{f.ID}); err != nil {
		fmt.Fprintln(os.Stderr, "save recent:", err)
		os.Exit(1)
	}
	fmt.Println("Seeded recent_flows.")
}
