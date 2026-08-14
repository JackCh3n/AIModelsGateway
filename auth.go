package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 管理后台可选登录保护
//
// 服务器默认监听 0.0.0.0（局域网可访问），而 /admin/api/* 会返回
// 全部上游 API Key 等敏感信息。未设置密码时保持原有行为（无鉴权）；
// 在「设置」页配置管理员密码后，/admin/api/* 需要携带有效会话 Cookie，
// 前端在收到 401 时弹出登录框。会话为 HMAC 签名令牌，有效期 12 小时。
// ============================================================

const (
	adminSessionCookie = "aim_admin_session"
	adminSessionTTL    = 12 * time.Hour
)

// hashAdminPassword 计算密码的 SHA-256 十六进制摘要（存储用）
func hashAdminPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

// adminPasswordEnabled 是否启用了管理员密码保护
func adminPasswordEnabled() bool {
	return loadConfig().Settings.AdminPasswordHash != ""
}

// adminTokenValue 生成会话令牌：expiry.hmac(expiry)，密钥为密码摘要
func adminTokenValue() string {
	cfg := loadConfig()
	hash := cfg.Settings.AdminPasswordHash
	exp := time.Now().Add(adminSessionTTL).Unix()
	expStr := strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, []byte(hash))
	mac.Write([]byte(expStr))
	return expStr + "." + hex.EncodeToString(mac.Sum(nil))
}

// adminTokenValid 校验会话令牌：有效期 + HMAC 签名一致
func adminTokenValid(token string) bool {
	cfg := loadConfig()
	hash := cfg.Settings.AdminPasswordHash
	if hash == "" {
		return true // 未启用保护时视为通过
	}
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	mac := hmac.New(sha256.New, []byte(hash))
	mac.Write([]byte(parts[0]))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(parts[1]), []byte(want))
}

// adminAuthed 判断当前请求是否已通过管理后台鉴权
func adminAuthed(r *http.Request) bool {
	if !adminPasswordEnabled() {
		return true
	}
	c, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return false
	}
	return adminTokenValid(c.Value)
}

// adminAuth 管理后台 API 鉴权中间件：仅拦截 /admin/api/*（login/logout 除外）
func adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/admin/api/") && p != "/admin/api/login" && p != "/admin/api/logout" {
			if !adminAuthed(r) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// adminLoginHandler POST /admin/api/login {password}
func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	cfg := loadConfig()
	if cfg.Settings.AdminPasswordHash == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "未设置管理员密码"})
		return
	}
	if !hmac.Equal([]byte(hashAdminPassword(body.Password)), []byte(cfg.Settings.AdminPasswordHash)) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "密码错误"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    adminTokenValue(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(adminSessionTTL.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// adminLogoutHandler POST /admin/api/logout 清除会话
func adminLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
