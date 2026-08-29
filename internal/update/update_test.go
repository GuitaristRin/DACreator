package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"3.0.0", "3.0.0", 0},
		{"3.0.0", "3.0.1", -1},
		{"3.1.0", "3.0.9", 1},
		{"v3.0.0", "3.0.0", 0},
		{"V4.0.0", "3.9.9", 1},
		{"3.0", "3.0.0", 0},
		{"3", "3.0.0", 0},
		{"3.10.0", "3.9.0", 1},
		{"dev", "3.0.0", -1}, // 非数字段按 0
		{"", "1.0.0", -1},
	}
	for _, tt := range tests {
		if got := CompareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

const releaseFixture = `{
  "tag_name": "v3.1.0",
  "body": "更新说明正文",
  "assets": [
    {"name": "DACreator_v3.1.0_x64.exe", "size": 1024, "browser_download_url": "https://example.com/dl"}
  ]
}`

func TestCheckUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(releaseFixture))
	}))
	defer srv.Close()

	orig := apiBase
	apiBase = srv.URL
	defer func() { apiBase = orig }()

	rel, has, err := CheckUpdate(context.Background(), "3.0.0")
	if err != nil {
		t.Fatalf("检查更新失败：%v", err)
	}
	if !has || rel.TagName != "v3.1.0" || rel.Body != "更新说明正文" || len(rel.Assets) != 1 {
		t.Errorf("结果不符：%+v has=%v", rel, has)
	}

	_, has, err = CheckUpdate(context.Background(), "3.1.0")
	if err != nil || has {
		t.Errorf("同版本不应提示更新：has=%v err=%v", has, err)
	}
}

func TestDownloadAsset(t *testing.T) {
	payload := []byte("fake-installer-bytes-1234567890")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.Write(payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "setup.exe")
	var lastPct int
	err := DownloadAsset(context.Background(), srv.URL, dest, func(pct int, _, _ int64) { lastPct = pct })
	if err != nil {
		t.Fatalf("下载失败：%v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("内容不符：%q", got)
	}
	if lastPct != 100 {
		t.Errorf("进度应到 100，实际 %d", lastPct)
	}
}
