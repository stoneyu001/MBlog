package config

import "testing"

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PASSWORD", "database-password")
	t.Setenv("ADMIN_USER", "owner")
	t.Setenv("ADMIN_PASSWORD", "a-strong-admin-password")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("COOKIE_SECURE", "true")
}

func TestLoadConfigUsesDBName(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DB_NAME", "mblog_test")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() 返回错误: %v", err)
	}
	if cfg.Database.DBName != "mblog_test" {
		t.Fatalf("数据库名 = %q，期望 mblog_test", cfg.Database.DBName)
	}
}

func TestLoadConfigRejectsDefaultAdminPassword(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("ADMIN_PASSWORD", "admin123")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("默认管理员密码应被拒绝")
	}
}

func TestLoadConfigRejectsShortJWTSecret(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("JWT_SECRET", "too-short")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("过短的 JWT_SECRET 应被拒绝")
	}
}

func TestLoadConfigRejectsExampleValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "数据库密码", key: "DB_PASSWORD", value: "your_secure_password_here"},
		{name: "管理员密码", key: "ADMIN_PASSWORD", value: "replace_with_a_strong_password"},
		{name: "JWT 密钥", key: "JWT_SECRET", value: "replace_with_at_least_32_random_characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tt.key, tt.value)
			if _, err := LoadConfig(); err == nil {
				t.Fatalf("示例值 %s 应被拒绝", tt.key)
			}
		})
	}
}
