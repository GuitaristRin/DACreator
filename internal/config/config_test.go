package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportLegacyDat(t *testing.T) {
	tests := []struct {
		name string
		dat  string
		want Config
	}{
		{
			name: "GUI 写出的标准格式",
			dat: "ID = 高橋リンタ\nREGION = 関東\nCITY = 東京\nSTORE = ゲームセンター\nTEAM = Project D\nSEASON = 5\nROUND = 4\nVERSION = 2.1.2\n",
			want: Config{ID: "高橋リンタ", Region: "関東", City: "東京", Store: "ゲームセンター", Team: "Project D", Season: 5, Round: 4},
		},
		{
			name: "旧模板 LOCALE 键与无空格等号",
			dat: "ID = 你的ID\nTEAM = 你的车队名\nSTORE = 你的店铺名\nLOCALE = 店铺所在地区\nCITY = 店铺所在城市\nSEASON = 5\nROUND = 4\nVERSION =2.1.1\n",
			want: Config{ID: "你的ID", Region: "店铺所在地区", City: "店铺所在城市", Store: "你的店铺名", Team: "你的车队名", Season: 5, Round: 4},
		},
		{
			name: "全角用户名 NFKC 归一化",
			dat:  "ID = ＡＢＣ１２３\nSEASON = 7\n",
			want: Config{ID: "ABC123", Region: "", Season: 7, Round: 1},
		},
		{
			name: "缺失字段回落默认值",
			dat:  "ID = rin\n",
			want: Config{ID: "rin", Season: 5, Round: 1},
		},
		{
			name: "非法赛季回落默认值",
			dat:  "ID = rin\nSEASON = abc\n",
			want: Config{ID: "rin", Season: 5, Round: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "Player_ID.dat")
			if err := os.WriteFile(path, []byte(tt.dat), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := ImportLegacyDat(path)
			if err != nil {
				t.Fatalf("导入失败：%v", err)
			}
			if got != tt.want {
				t.Errorf("结果不符：got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestImportLegacyDatFileMissing(t *testing.T) {
	_, err := ImportLegacyDat(filepath.Join(t.TempDir(), "不存在.dat"))
	if err == nil {
		t.Fatal("文件缺失应报错")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())          // Windows
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())  // Linux
	t.Setenv("HOME", t.TempDir())             // macOS 兜底

	want := Config{ID: "rin", Region: "関東", City: "東京", Store: "店", Team: "D", Season: 6, Round: 2}
	if err := Save(want); err != nil {
		t.Fatalf("保存失败：%v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if got != want {
		t.Errorf("roundtrip 不符：got %+v want %+v", got, want)
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	got, err := Load()
	if err != nil {
		t.Fatalf("缺失配置文件不应报错：%v", err)
	}
	if got != Default() {
		t.Errorf("应返回默认配置：got %+v", got)
	}
}
