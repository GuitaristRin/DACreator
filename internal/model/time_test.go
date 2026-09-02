package model

import "testing"

func TestFormatRaceTime(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{0, "0:00.000"},
		{760, "0:00.760"},
		{61000, "1:01.000"},
		{147760, "2:27.760"},
		{62345, "1:02.345"},
		{6039999, "100:39.999"}, // 分钟进位边界（99分99秒）
	}
	for _, tt := range tests {
		if got := FormatRaceTime(tt.ms); got != tt.want {
			t.Errorf("FormatRaceTime(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestParseRaceTime(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"2:27.760", 147760, false},  // 服务器格式
		{"2'27\"760", 147760, false}, // 旧版 CSV 格式
		{"1:02.345", 62345, false},
		{"1'02\"345", 62345, false},
		{"0:00.000", 0, false},
		{"0'00\"000", 0, false},
		{" 2:27.760 ", 147760, false}, // 容忍首尾空白
		{"", 0, true},
		{"abc", 0, true},
		{"2:27", 0, true}, // 缺毫秒
		{"2'27", 0, true},
		{"2:60.000", 0, true},   // 秒越界
		{"99'99\"999", 0, true}, // rank.csv 旧哨兵值：秒位 99 不合法，v3 以空单元格表示无标准
		{"2:27.1000", 0, true},  // 毫秒越界
		{"x2:27.760", 0, true},
		{"2:2a.760", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseRaceTime(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseRaceTime(%q) 期望报错，实际得到 %d", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRaceTime(%q) 意外出错：%v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRaceTime(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestRaceTimeRoundTrip(t *testing.T) {
	cases := []int{0, 1, 999, 147760, 62345, 3723456, 6039999}
	for _, ms := range cases {
		parsed, err := ParseRaceTime(FormatRaceTime(ms))
		if err != nil {
			t.Fatalf("roundtrip %d: %v", ms, err)
		}
		if parsed != ms {
			t.Errorf("roundtrip %d 得到 %d", ms, parsed)
		}
	}
}
