import json
import re
import requests
import pandas as pd
from bs4 import BeautifulSoup
from typing import List, Dict, Optional

CONFIG = {
    "base_web_url": "https://arcadezone.cn/ranking#timetrial",
    "api_url": "https://arcadezone.cn/ranking/timetrial",
    "season": 5,  # 默认值，会被配置文件覆盖
    "headers": {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
        "Content-Type": "application/json",
        "Accept": "application/json, text/plain, */*",
        "Referer": "https://arcadezone.cn/ranking#timetrial",
        "Origin": "https://arcadezone.cn",
    },
    "player_id_path": "Player_ID.dat",
    "standard_time_path": "./assets/rank.csv",  # 标准等级时间库路径
    "timeout": 30,
    "max_retry": 3,
    "course_name_map": {
        0: "秋名湖",
        2: "秋名湖",
        4: "妙義",
        6: "妙義",
        8: "赤城",
        10: "赤城",
        12: "秋名",
        14: "秋名",
        16: "伊吕波坂",
        18: "伊吕波坂",
        20: "筑波",
        22: "筑波",
        24: "八方原",
        26: "八方原",
        28: "长尾",
        30: "长尾",
        32: "椿线",
        34: "椿线",
        36: "碓冰",
        38: "碓冰",
        40: "定峰",
        42: "定峰",
        44: "土坂",
        46: "土坂",
        48: "秋名雪",
        50: "秋名雪",
        52: "箱根",
        54: "箱根",
        56: "枫树线",
        58: "枫树线",
        60: "七曲",
        62: "七曲",
        64: "群馬赛车场",
        66: "群馬赛车场",
        68: "小田原",
        70: "小田原",
        72: "筑波雪",
        74: "筑波雪",
        76: "矢矩",
        78: "矢矩",
        80: "土坂雪",
        82: "土坂雪",
        84: "真鹤",
        86: "真鹤",
        88: "碓冰雪",
        90: "碓冰雪",
        92: "秋名雨",
        94: "秋名雨"
    },
    "course_direction_map": {
        0: "逆时针",
        2: "顺时针",
        4: "下坡",
        6: "上坡",
        8: "下坡",
        10: "上坡",
        12: "下坡",
        14: "上坡",
        16: "下坡",
        18: "逆行",
        20: "去路",
        22: "归路",
        24: "去路",
        26: "归路",
        28: "下坡",
        30: "上坡",
        32: "下坡",
        34: "上坡",
        36: "逆时针",
        38: "顺时针",
        40: "下坡",
        42: "上坡",
        44: "去路",
        46: "归路",
        48: "下坡",
        50: "上坡",
        52: "下坡",
        54: "上坡",
        56: "下坡",
        58: "上坡",
        60: "下坡",
        62: "上坡",
        64: "去路",
        66: "归路",
        68: "顺行",
        70: "逆行",
        72: "去路",
        74: "归路",
        76: "下坡",
        78: "上坡",
        80: "去路",
        82: "归路",
        84: "顺行",
        86: "逆行",
        88: "逆时针",
        90: "顺时针",
        92: "下坡",
        94: "上坡"
    },
    "rank_priority": ["LEGEND", "MASTER+", "MASTER", "PROFESSIONAL", "EXPERT", "SPECIALIST", "REGULAR"]
}

class ArcadeZoneCrawler:
    def __init__(self):
        self.headers = CONFIG["headers"].copy()
        self.api_url = CONFIG["api_url"]
        self.base_web_url = CONFIG["base_web_url"]
        # 从配置文件加载赛季
        self.season = self._load_season()
        self.target_username = self._load_target_username()
        self.standard_times = self._load_standard_times()
        self.session = requests.Session()
        self._get_csrf_token()

    def _load_season(self) -> int:
        """从配置文件加载赛季"""
        default_season = CONFIG["season"]
        
        try:
            with open(CONFIG["player_id_path"], "r", encoding="utf-8") as f:
                lines = f.readlines()
                
            for line in lines:
                line = line.strip()
                if line.startswith("SEASON = "):
                    try:
                        season = int(line.split("=")[1].strip())
                        print(f"✅ 加载赛季配置：第 {season} 赛季")
                        return season
                    except:
                        pass
            
            print(f"⚠️ 配置文件中未找到赛季设置，使用默认值：第 {default_season} 赛季")
            return default_season
            
        except Exception as e:
            print(f"⚠️ 读取配置文件失败，使用默认赛季：第 {default_season} 赛季")
            return default_season

    def _load_target_username(self) -> str:
        """从配置文件加载目标用户名"""
        try:
            with open(CONFIG["player_id_path"], "r", encoding="utf-8") as f:
                lines = f.readlines()
                
            for line in lines:
                line = line.strip()
                if line.startswith("ID = "):
                    username = line.split("=")[1].strip()
                    print(f"✅ 成功加载目标用户：{username}")
                    return username
                    
            raise ValueError("配置文件中未找到 ID 行")
            
        except FileNotFoundError:
            raise Exception(f"❌ 未找到配置文件：{CONFIG['player_id_path']}")
        except Exception as e:
            raise Exception(f"❌ 读取配置文件失败：{str(e)}")

    def _load_standard_times(self) -> pd.DataFrame:
        try:
            df = pd.read_csv(CONFIG["standard_time_path"], encoding="utf-8-sig")
            required_cols = ["Course", "Direction"] + CONFIG["rank_priority"]
            for col in required_cols:
                if col not in df.columns:
                    raise Exception(f"标准库缺少必填列：{col}，请检查CSV文件")
            for rank in CONFIG["rank_priority"]:
                df[rank] = df[rank].fillna("99'99\"999")
            print(f"✅ 成功加载等级标准库，共{len(df)}条赛道标准")
            return df
        except FileNotFoundError:
            raise Exception(f"❌ 未找到等级标准库：{CONFIG['standard_time_path']}")
        except Exception as e:
            raise Exception(f"❌ 读取等级标准库失败：{str(e)}")

    def _get_csrf_token(self):
        """获取CSRF Token"""
        try:
            response = self.session.get(
                self.base_web_url,
                headers=self.headers,
                timeout=CONFIG["timeout"]
            )
            response.raise_for_status()
            soup = BeautifulSoup(response.text, "html.parser")
            csrf_meta = soup.find("meta", attrs={"name": "csrf-token"})
            if csrf_meta:
                csrf_token = csrf_meta.get("content")
                self.headers["X-CSRF-TOKEN"] = csrf_token
                print(f"✅ 成功获取CSRF Token：{csrf_token[:10]}...")
            else:
                raise Exception("网页中未找到CSRF Token")
        except Exception as e:
            raise Exception(f"❌ 获取CSRF Token失败：{str(e)}")

    def _str_time_to_ms(self, time_str: str) -> int:
        try:
            if ":" in time_str and "." in time_str:
                minute, rest = time_str.split(":")
                second, ms = rest.split(".")
                return int(minute)*60000 + int(second)*1000 + int(ms)
            elif "'" in time_str and "\"" in time_str:
                minute, rest = time_str.split("'")
                second, ms = rest.split("\"")
                return int(minute)*60000 + int(second)*1000 + int(ms)
            else:
                return 99999999  # 异常格式返回极大值
        except:
            return 99999999

    def _judge_rank(self, course: str, direction: str, score_ms: int) -> str:
        mask = (self.standard_times["Course"] == course) & (self.standard_times["Direction"] == direction)
        if not mask.any():
            print(f"⚠️  未找到{course}-{direction}的等级标准，默认未知评价")
            return "未知评价"
        
        # 获取标准行数据
        standard_row = self.standard_times[mask].iloc[0]
        for rank in CONFIG["rank_priority"]:
            standard_ms = self._str_time_to_ms(str(standard_row[rank]))
            if score_ms <= standard_ms:
                return rank
        
        # 所有高级标准都不满足，判定为ROOKIE
        return "ROOKIE"

    def _request_api(self, page: int, course_id: int) -> Optional[Dict]:
        payload = {
            "page": page,
            "season": self.season,
            "course": course_id
        }
        for retry in range(CONFIG["max_retry"]):
            try:
                response = self.session.post(
                    url=self.api_url,
                    headers=self.headers,
                    data=json.dumps(payload, ensure_ascii=False),
                    timeout=CONFIG["timeout"]
                )
                response.raise_for_status()
                return response.json()
            except requests.exceptions.RequestException as e:
                print(f"⚠️  第{retry+1}次请求失败，赛道{course_id}第{page}页：{str(e)}")
                if retry == CONFIG["max_retry"] - 1:
                    print(f"❌ 赛道{course_id}第{page}页请求最终失败，跳过该页")
                    return None

    def _parse_time(self, ms: int) -> str:
        minutes = ms // 60000
        seconds = (ms % 60000) // 1000
        millis = ms % 1000
        return f"{minutes}:{seconds:02d}.{millis:03d}"

    def _parse_rank_data(self, data: Dict, course_id: int, current_page: int) -> List[Dict]:
        result = []
        rank_list = data.get("list", [])
        car_styles_map = data.get("carStyles", {})
        course_name = CONFIG["course_name_map"].get(course_id, "未知赛道")
        direction = CONFIG["course_direction_map"].get(course_id, "未知方向")

        per_page = data.get("pagination", {}).get("per_page", 15)

        for idx, item in enumerate(rank_list):
            user_info = item.get("userinfo", {})
            username = user_info.get("username", "")
            if username != self.target_username:
                continue

            national_rank = (current_page - 1) * per_page + idx + 1
            car_id = str(item.get("style_car_id", ""))
            car_name = car_styles_map.get(car_id, "未知车型")
            goal_time_ms = item.get("goal_time", 0)
            time_str = self._parse_time(goal_time_ms)
            play_time = item.get("play_dt", "").split(" ")[0]

            time_eval = self._judge_rank(course_name, direction, goal_time_ms)
            print(f"[判断] {course_name}({direction}) | 成绩：{time_str} → 等级：{time_eval}")

            rank_info = {
                "コース": course_name,
                "ルート": direction,
                "タイム": time_str,
                "タイム評価": time_eval,
                "記録車種": car_name,
                "全国順位": str(national_rank),
                "記録日": play_time
            }
            result.append(rank_info)
        return result

    def crawl_course(self, course_id: int) -> List[Dict]:
        course_name = CONFIG["course_name_map"].get(course_id, "未知赛道")
        direction = CONFIG["course_direction_map"].get(course_id, "未知方向")
        print(f"\n========== 开始爬取 赛道ID:{course_id}（{course_name}({direction})） ==========")
        all_matched_data = []

        first_page_data = self._request_api(page=1, course_id=course_id)
        if not first_page_data:
            return all_matched_data

        page1_data = self._parse_rank_data(first_page_data, course_id, current_page=1)
        all_matched_data.extend(page1_data)

        total_pages = first_page_data.get("pagination", {}).get("last_page", 1)
        print(f"✅ 赛道{course_id} 总页数：{total_pages}")

        for page in range(2, total_pages + 1):
            print(f"正在爬取 赛道{course_id} 第{page}/{total_pages}页...",end='\r')
            page_data = self._request_api(page=page, course_id=course_id)
            if not page_data:
                continue
            matched_data = self._parse_rank_data(page_data, course_id, current_page=page)
            all_matched_data.extend(matched_data)

        print(f"========== 赛道{course_id} 爬取完成，匹配到{len(all_matched_data)}条成绩 ==========\n")
        return all_matched_data

    def run(self, course_list: List[int], return_df: bool = False) -> Optional[pd.DataFrame]:
        final_result = []
        for course_id in course_list:
            data = self.crawl_course(course_id)
            final_result.extend(data)

        if not final_result:
            print(f"❌ 未匹配到{self.target_username}的任何成绩记录")
            if return_df:
                return pd.DataFrame()
            return None

        # 保持原有输出格式
        csv_columns = ["コース", "ルート", "タイム", "タイム評価", "記録車種", "全国順位", "記録日"]
        df = pd.DataFrame(final_result)[csv_columns]
        
        if not return_df:
            csv_filename = f"DAC_{self.target_username}_成绩表.csv"
            df.to_csv(csv_filename, index=False, encoding="utf-8-sig")

            # 控制台输出结果
            print("=" * 80)
            print(f"🎯 爬取任务全部完成！总计匹配到{len(final_result)}条成绩")
            print(f"✅ CSV文件已保存：{csv_filename}")
            print("=" * 80)
            print("\n【成绩预览】")
            print(df.to_string(index=False))
        
        return df if return_df else None

# 对外暴露的爬取函数（供core调用）
def crawl_data() -> pd.DataFrame:
    # 配置需要爬取的赛道ID列表
    TARGET_COURSES = [0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94]

    try:
        crawler = ArcadeZoneCrawler()
        df = crawler.run(TARGET_COURSES, return_df=True)
        return df
    except Exception as e:
        print(f"❌ 爬虫执行失败：{str(e)}")
        return pd.DataFrame()

if __name__ == "__main__":
    # 独立运行爬虫的逻辑
    TARGET_COURSES = [0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94]

    try:
        crawler = ArcadeZoneCrawler()
        crawler.run(TARGET_COURSES)
    except Exception as e:
        print(f"❌ 程序运行失败：{str(e)}")