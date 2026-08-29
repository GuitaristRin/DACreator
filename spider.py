import json
import re
import unicodedata
import requests
from bs4 import BeautifulSoup
from typing import Optional

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
    }
}

# 服务器返回的 eval_id 到成绩等级的映射（来自网页前端 calculateHyokaid 的图标编号）
EVAL_ID_RANKS = {
    1: "ROOKIE",
    2: "ROOKIE",
    3: "ROOKIE",
    4: "ROOKIE",
    5: "REGULAR",
    6: "REGULAR",
    7: "REGULAR",
    8: "REGULAR",
    9: "SPECIALIST",
    10: "SPECIALIST",
    11: "SPECIALIST",
    12: "SPECIALIST",
    13: "EXPERT",
    14: "EXPERT",
    15: "EXPERT",
    16: "EXPERT",
    17: "PROFESSIONAL",
    18: "PROFESSIONAL",
    19: "PROFESSIONAL",
    20: "PROFESSIONAL",
    21: "MASTER",
    22: "MASTER",
    23: "MASTER",
    24: "MASTER",
    25: "MASTER+",
    26: "MASTER+",
    27: "MASTER+",
    28: "MASTER+",
    29: "LEGEND"
}


class ArcadeZoneCrawler:
    def __init__(self):
        self.headers = CONFIG["headers"].copy()
        self.api_url = CONFIG["api_url"]
        self.base_web_url = CONFIG["base_web_url"]
        # 从配置文件加载赛季
        self.season = self._load_season()
        self.target_username = self._load_target_username()
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
        """从配置文件加载目标用户名（NFKC 归一化以兼容全角字符）"""
        try:
            with open(CONFIG["player_id_path"], "r", encoding="utf-8") as f:
                lines = f.readlines()

            for line in lines:
                line = line.strip()
                if line.startswith("ID = "):
                    username = unicodedata.normalize("NFKC", line.split("=")[1].strip())
                    print(f"✅ 成功加载目标用户：{username}")
                    return username

            raise ValueError("配置文件中未找到 ID 行")

        except FileNotFoundError:
            raise Exception(f"❌ 未找到配置文件：{CONFIG['player_id_path']}")
        except Exception as e:
            raise Exception(f"❌ 读取配置文件失败：{str(e)}")

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

    def _parse_time(self, ms: int) -> str:
        minutes = ms // 60000
        seconds = (ms % 60000) // 1000
        millis = ms % 1000
        return f"{minutes}:{seconds:02d}.{millis:03d}"
