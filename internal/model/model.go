// Package model 定义 DACreator 的核心数据类型与 CSV 编解码。
// CSV schema 与旧版 Python 实现完全一致，保证用户既有成绩文件与 raw/ 历史数据可用。
package model

// CSV 列名（日文，与旧版及 ArcadeZone 数据文件保持一致）。
const (
	ColCourse    = "コース"
	ColDirection = "ルート"
	ColTime      = "タイム"
	ColRank      = "タイム評価"
	ColCar       = "記録車種"
	ColNational  = "全国順位"
	ColDate      = "記録日"
)

// Columns 是 canonical 的 7 列表头顺序。
var Columns = []string{ColCourse, ColDirection, ColTime, ColRank, ColCar, ColNational, ColDate}

// Record 是单条计时赛成绩。
type Record struct {
	Course    string // 赛道名，如 "秋名湖"
	Direction string // 路线方向，如 "下坡"
	TimeMs    int    // 成绩毫秒数（内部唯一规范表示）
	Rank      string // 等级评价，来自服务器 eval_id 映射（如 "EXPERT"）
	Car       string // 记录车辆
	National  string // 全国排名，如 "255位"；旧格式文件可能缺省为空
	Date      string // 记录日期，YYYY/MM/DD
}
