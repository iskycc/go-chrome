// test-server 是一个独立的演示用 HTTP 服务端，用于本地测试 go-chrome 的登录流程。
// 该文件完全独立，删除 cmd/test-server 目录即可移除所有测试代码，不影响主程序。
//
// 使用方法：
//   go run ./cmd/test-server
//   然后浏览器访问 http://localhost:8080
//
// 预置 XPath（与示例流程一致）：
//   用户名输入框：//input[@id='username']
//   密码输入框：  //input[@id='password']
//   登录按钮：    //button[@type='submit']
//   欢迎文本：    //div[contains(text(),'欢迎')]
//   登出按钮：    //button[@id='logout']

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

const addr = ":18080"

// 硬编码演示账号（任意用户名密码均可登录，演示用）
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleLoginPage)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/welcome", handleWelcome)
	mux.HandleFunc("/logout", handleLogout)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("[test-server] 启动于 http://localhost%s", addr)
	log.Printf("[test-server] 预置 XPath:")
	log.Printf("  用户名: //input[@id='username']")
	log.Printf("  密码:   //input[@id='password']")
	log.Printf("  登录:   //button[@type='submit']")
	log.Printf("  欢迎:   //div[contains(text(),'欢迎')]")
	log.Printf("  登出:   //button[@id='logout']")

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	loginTmpl.Execute(w, nil)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 演示服务器：任意非空用户名/密码均允许登录
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		http.Error(w, "用户名或密码不能为空", http.StatusBadRequest)
		return
	}

	// 设置登录 cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "demo-session-" + username,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600,
	})

	http.Redirect(w, r, "/welcome?user="+template.URLQueryEscaper(username), http.StatusSeeOther)
}

func handleWelcome(w http.ResponseWriter, r *http.Request) {
	// 简单检查 cookie（仅演示，不做真实鉴权）
	_, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	user := r.URL.Query().Get("user")
	if user == "" {
		user = "访客"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	welcomeTmpl.Execute(w, map[string]string{"User": user})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------- HTML 模板 ----------

var loginTmpl = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>演示登录页</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f2f5; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
  .card { background: #fff; padding: 40px 32px; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); width: 320px; }
  h2 { margin: 0 0 24px; text-align: center; color: #333; }
  .field { margin-bottom: 16px; }
  label { display: block; margin-bottom: 6px; font-size: 14px; color: #555; }
  input[type="text"], input[type="password"] { width: 100%; padding: 10px 12px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 14px; box-sizing: border-box; }
  input:focus { outline: none; border-color: #1a73e8; }
  button[type="submit"] { width: 100%; padding: 11px; background: #1a73e8; color: #fff; border: none; border-radius: 4px; font-size: 15px; cursor: pointer; }
  button[type="submit"]:hover { background: #1557b0; }
  .hint { margin-top: 16px; font-size: 12px; color: #888; text-align: center; }
</style>
</head>
<body>
<div class="card">
  <h2>后台登录</h2>
  <form action="/login" method="POST">
    <div class="field">
      <label for="username">用户名</label>
      <input type="text" id="username" name="username" placeholder="请输入用户名">
    </div>
    <div class="field">
      <label for="password">密码</label>
      <input type="password" id="password" name="password" placeholder="请输入密码">
    </div>
    <button type="submit">登录</button>
  </form>
  <div class="hint">演示服务器：任意非空用户名/密码均可登录</div>
</div>
</body>
</html>`))

var welcomeTmpl = template.Must(template.New("welcome").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>欢迎页</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f2f5; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
  .card { background: #fff; padding: 40px 32px; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); width: 360px; text-align: center; }
  h2 { margin: 0 0 16px; color: #333; }
  .welcome-text { font-size: 18px; color: #1a73e8; margin-bottom: 24px; }
  button#logout { padding: 10px 24px; background: #e53935; color: #fff; border: none; border-radius: 4px; font-size: 14px; cursor: pointer; }
  button#logout:hover { background: #c62828; }
</style>
</head>
<body>
<div class="card">
  <h2>登录成功</h2>
  <div class="welcome-text">欢迎，{{.User}}！</div>
  <form action="/logout" method="POST">
    <button id="logout" type="submit">退出登录</button>
  </form>
</div>
</body>
</html>`))

// 辅助函数（兼容旧版本 Go，template.URLQueryEscaper 在 Go 1.22+ 可用）
func init() {
	_ = fmt.Sprintf // 避免未使用 import 报错（实际由 template 使用）
}
