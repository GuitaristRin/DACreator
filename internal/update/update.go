// Package update 实现基于 GitHub Releases 的自更新检查与下载。
// v3 不再使用镜像站 hack：直接走 GitHub API，网络不可达时由用户手动更新。
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Owner = "GuitaristRin"
	Repo  = "DACreator"
)

// Asset 是 Release 附件。
type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Release 是 Release 信息。
type Release struct {
	TagName string  `json:"tag_name"`
	Body    string  `json:"body"`
	Assets  []Asset `json:"assets"`
}

// ProgressFn 下载进度回调：pct 0-100，downloaded/total 为字节数（total 可能未知为 0）。
type ProgressFn func(pct int, downloaded, total int64)

// apiBase 供测试注入替换。
var apiBase = "https://api.github.com"

// Latest 拉取最新 Release。
func Latest(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, Owner, Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DACreator-Engine")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("访问 GitHub Releases 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("访问 GitHub Releases 失败：HTTP %d", resp.StatusCode)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析 Release 信息: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("Release 信息缺少版本号")
	}
	return &rel, nil
}

// CheckUpdate 比较当前版本与最新 Release。
// 返回 (最新 Release, 是否有更新, error)。
func CheckUpdate(ctx context.Context, current string) (*Release, bool, error) {
	rel, err := Latest(ctx)
	if err != nil {
		return nil, false, err
	}
	return rel, CompareVersions(current, rel.TagName) < 0, nil
}

// CompareVersions 语义化比较版本号，容忍可选的 v/V 前缀与不足三段。
// 返回 -1/0/1。解析失败的段按 0 处理。
func CompareVersions(a, b string) int {
	pa, pb := parseVersion(a), parseVersion(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			n = 0
		}
		out[i] = n
	}
	return out
}

// DownloadAsset 流式下载附件到 destPath，通过 onProgress 汇报进度。
func DownloadAsset(ctx context.Context, url, destPath string, onProgress ProgressFn) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DACreator-Engine")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败：HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建文件 %s: %w", destPath, err)
	}
	defer f.Close()

	var downloaded int64
	buf := make([]byte, 64<<10)
	lastPct := -1
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return fmt.Errorf("写入文件: %w", werr)
			}
			downloaded += int64(n)
			if onProgress != nil && total > 0 {
				pct := int(downloaded * 100 / total)
				if pct != lastPct {
					lastPct = pct
					onProgress(pct, downloaded, total)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("下载中断: %w", err)
		}
	}
	return nil
}
