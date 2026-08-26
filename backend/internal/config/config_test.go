package config

import "testing"

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("DB_PASSWORD", "database-password")
	t.Setenv("ADMIN_USER", "owner")
	t.Setenv("ADMIN_PASSWORD", "password")
	t.Setenv("JWT_SECRET", "secret")
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

func TestLoadConfigAcceptsShortNonEmptyValues(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("ADMIN_PASSWORD", "1")
	t.Setenv("JWT_SECRET", "2")

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("非空配置不应被额外限制: %v", err)
	}
}

func TestLoadConfigRejectsEmptyRequiredValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "数据库密码", key: "DB_PASSWORD"},
		{name: "管理员密码", key: "ADMIN_PASSWORD"},
		{name: "JWT 密钥", key: "JWT_SECRET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tt.key, "")
			if _, err := LoadConfig(); err == nil {
				t.Fatalf("空配置 %s 应被拒绝", tt.key)
			}
		})
	}
}
