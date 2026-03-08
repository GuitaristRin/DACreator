import json
import pandas as pd
from typing import List, Dict, Optional

# 复用原 spider 的配置和基础类
from spider import CONFIG, ArcadeZoneCrawler

class ArcadeZoneSearchCrawler(ArcadeZoneCrawler):
    """
    通过用户名搜索获取成绩的爬虫（结果不含全国排名）
    继承原爬虫的基础方法（等级判断、时间格式化、CSRF获取等）
    """

    def _search_request(self, payload: dict) -> Optional[dict]:
        """带重试的搜索请求 - 修复编码问题"""
        for retry in range(CONFIG["max_retry"]):
            try:
                # 关键修复：将数据编码为UTF-8
                response = self.session.post(
                    url=self.api_url,
                    headers=self.headers,
                    data=json.dumps(payload, ensure_ascii=False).encode('utf-8'),
                    timeout=CONFIG["timeout"]
                )
                response.raise_for_status()
                return response.json()
            except Exception as e:
                print(f"⚠️ 搜索请求失败（第{retry+1}次）：{e}")
                if retry == CONFIG["max_retry"] - 1:
                    print("❌ 搜索请求最终失败")
                    return None

    def search_by_name(self, name: str, course_id: Optional[int] = None) -> List[Dict]:
        """
        在指定赛道搜索用户的所有成绩
        返回记录列表，每条记录包含：赛道、路线、时间、等级、车型、日期
        """
        all_records = []
        page = 1

        while True:
            payload = {
                "page": page,
                "name": name,
                "season": self.season
            }
            if course_id is not None:
                payload["course"] = course_id

            data = self._search_request(payload)
            if not data:
                break

            page_records = self._parse_search_result(data)
            all_records.extend(page_records)

            pagination = data.get("pagination", {})
            if page >= pagination.get("last_page", 1):
                break
            page += 1

        return all_records

    def _parse_search_result(self, data: dict) -> List[dict]:
        """解析搜索结果，不包含排名信息"""
        result = []
        rank_list = data.get("list", [])
        car_styles_map = data.get("carStyles", {})

        for item in rank_list:
            course_id = item.get("course_id")
            course_name = CONFIG["course_name_map"].get(course_id, "未知赛道")
            direction = CONFIG["course_direction_map"].get(course_id, "未知方向")

            car_id = str(item.get("style_car_id", ""))
            car_name = car_styles_map.get(car_id, "未知车型")

            goal_time_ms = item.get("goal_time", 0)
            time_str = self._parse_time(goal_time_ms)
            play_time = item.get("play_dt", "").split(" ")[0]

            # 复用等级判断
            time_eval = self._judge_rank(course_name, direction, goal_time_ms)

            record = {
                "コース": course_name,
                "ルート": direction,
                "タイム": time_str,
                "タイム評価": time_eval,
                "記録車種": car_name,
                "記録日": play_time
                # 没有“全国順位”
            }
            result.append(record)

        return result

    def crawl_all_courses_by_search(self, name: str) -> List[Dict]:
        """遍历所有赛道，用搜索方式获取用户在每个赛道的成绩"""
        # 复用原爬虫中的赛道列表
        target_courses = [
            0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30,
            32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60,
            62, 64, 66, 68, 70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94
        ]
        all_records = []
        for cid in target_courses:
            print(f"🔍 正在搜索赛道 {cid} ...")
            records = self.search_by_name(name, course_id=cid)
            if records:
                print(f"   ✅ 找到 {len(records)} 条记录")
                all_records.extend(records)
            else:
                print(f"   ⏺️ 无记录")
        return all_records


# 对外暴露的爬取函数（供core调用）
def crawl_data_by_search(username: str = None) -> pd.DataFrame:
    """
    通过用户名搜索爬取成绩（无排名）
    若 username 为 None，则从 Player_ID.dat 读取
    """
    if username is None:
        try:
            with open(CONFIG["player_id_path"], "r", encoding="utf-8") as f:
                lines = f.readlines()
                for line in lines:
                    line = line.strip()
                    if line.startswith("ID = "):
                        username = line.split("=")[1].strip()
                        break
                if username is None:
                    raise ValueError("配置文件中未找到 ID 行")
            print(f"✅ 从配置文件读取用户名：{username}")
        except Exception as e:
            print(f"❌ 读取配置文件失败：{e}")
            return pd.DataFrame()

    try:
        crawler = ArcadeZoneSearchCrawler()
        print(f"🔍 开始搜索用户 {username} 在所有赛道的成绩...")
        records = crawler.crawl_all_courses_by_search(username)

        if not records:
            print("❌ 未找到任何成绩记录")
            return pd.DataFrame()

        df = pd.DataFrame(records)
        print(f"✅ 搜索完成，共找到 {len(df)} 条成绩记录")
        return df

    except Exception as e:
        print(f"❌ 搜索爬虫执行失败：{e}")
        import traceback
        traceback.print_exc()
        return pd.DataFrame()


if __name__ == "__main__":
    # 独立测试
    df = crawl_data_by_search()
    if not df.empty:
        print(df.to_string(index=False))