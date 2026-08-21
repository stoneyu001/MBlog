package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setAuthEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("ADMIN_USER", "owner")
	t.Setenv("ADMIN_PASSWORD", "a-strong-admin-password")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("COOKIE_SECURE", "true")
}

func TestGenerateAndParseToken(t *testing.T) {
	setAuthEnvironment(t)

	token, err := GenerateToken("owner")
	if err != nil {
		t.Fatalf("GenerateToken() 返回错误: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() 返回错误: %v", err)
	}
	if claims.Username != "owner" {
		t.Fatalf("用户名 = %q，期望 owner", claims.Username)
	}
}

func TestLoginSetsSecureCookie(t *testing.T) {
	setAuthEnvironment(t)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"owner","password":"a-strong-admin-password"}`))
	request.Header.Set("Content-Type", "application/json")
	context.Request = request

	Login(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", recorder.Code)
	}
	cookie := recorder.Header().Get("Set-Cookie")
	for _, attribute := range []string{"HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(cookie, attribute) {
			t.Fatalf("Cookie 缺少 %s: %s", attribute, cookie)
		}
	}
}
