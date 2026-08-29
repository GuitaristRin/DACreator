package az

// EvalIDRanks 服务器返回的 eval_id → 成绩等级映射。
// 来源：ArcadeZone 网页前端 calculateHyokaid 的图标编号（旧版已实测校准）。
// 等级判定唯一以此为准，禁止使用本地 rank.csv 推算。
var EvalIDRanks = map[int]string{
	1: "ROOKIE", 2: "ROOKIE", 3: "ROOKIE", 4: "ROOKIE",
	5: "REGULAR", 6: "REGULAR", 7: "REGULAR", 8: "REGULAR",
	9: "SPECIALIST", 10: "SPECIALIST", 11: "SPECIALIST", 12: "SPECIALIST",
	13: "EXPERT", 14: "EXPERT", 15: "EXPERT", 16: "EXPERT",
	17: "PROFESSIONAL", 18: "PROFESSIONAL", 19: "PROFESSIONAL", 20: "PROFESSIONAL",
	21: "MASTER", 22: "MASTER", 23: "MASTER", 24: "MASTER",
	25: "MASTER+", 26: "MASTER+", 27: "MASTER+", 28: "MASTER+",
	29: "LEGEND",
}

// RankName 返回 eval_id 对应等级，未知 ID 返回 "未知评价"。
func RankName(evalID int) string {
	if r, ok := EvalIDRanks[evalID]; ok {
		return r
	}
	return "未知评价"
}
