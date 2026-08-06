package rancher

import (
	"testing"
)

func TestParseNginxConfig_ServerWrapped(t *testing.T) {
	// 模拟标准 nginx 配置：server {} 包裹 location {}
	config := `
server {
    listen 80;
    server_name dev.ymygz.com;

    location /lushanxihai2-integration/admin/ {
        proxy_pass http://web-lsxh-admin.dev-lushanxihai2-integration:80/;
    }
}`

	entries, err := ParseNginxConfig("https://dev.ymygz.com", config)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("没有解析出任何 ConfigEntry，server {} 包裹导致 location 被跳过")
	}

	e := entries[0]
	if e.ServerName != "web-lsxh-admin" {
		t.Errorf("ServerName = %q, want %q", e.ServerName, "web-lsxh-admin")
	}
	if e.Domain != "dev-lushanxihai2-integration" {
		t.Errorf("Domain = %q, want %q", e.Domain, "dev-lushanxihai2-integration")
	}
	if e.LocationPath != "/lushanxihai2-integration/admin/" {
		t.Errorf("LocationPath = %q, want %q", e.LocationPath, "/lushanxihai2-integration/admin/")
	}
	if e.BaseURL != "https://dev.ymygz.com" {
		t.Errorf("BaseURL = %q, want %q", e.BaseURL, "https://dev.ymygz.com")
	}

	t.Logf("解析成功: %+v", e)
}

func TestParseNginxConfig_NestedLocation(t *testing.T) {
	// 验证嵌套 location 仍然能正确解析
	config := `
server {
    listen 80;
    location /api {
        proxy_pass http://api-svc.default:8080/;
        location /api/v2 {
            proxy_pass http://api-v2-svc.default:8080/;
        }
    }
}`

	entries, err := ParseNginxConfig("https://example.com", config)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("期望 2 个 ConfigEntry，实际得到 %d", len(entries))
	}

	// 内层 location 应该也被找到
	found := false
	for _, e := range entries {
		if e.ServerName == "api-v2-svc" {
			found = true
		}
	}
	if !found {
		t.Error("未找到嵌套的 location /api/v2")
	}
}
