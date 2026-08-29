#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
DACreator 回合排名模块
获取用户的回合分数和排名
"""

import os
import sys
import json
from typing import Optional, Dict, Any

# 将项目根目录添加到路径
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, BASE_DIR)

from method.spider import ArcadeZoneCrawler, CONFIG

# 回合ID映射（根据实际HTML中的value值）
ROUND_ID_MAPPING = {
    1: 5,   # Season5 Round 1
    2: 7,   # Season5 Round 2
    3: 8,   # Season5 Round 3
    4: 9,   # Season5 Round 4
}


class RoundCrawler(ArcadeZoneCrawler):
    """回合排名爬虫类"""
    
    def __init__(self, username: str = None, season: int = None, round_num: int = None, callback=None):
        super().__init__(username, season, callback)
        self.api_url = "https://arcadezone.cn/ranking/round"
        
        if round_num is not None:
            self.round_seq = round_num
        else:
            self.round_seq = self._load_round_seq()
        
        # 转换为API需要的round_id
        self.round_id = ROUND_ID_MAPPING.get(self.round_seq, 9)
        
        self._log(f"初始化回合排名爬虫，回合序号：{self.round_seq} -> API round_id：{self.round_id}")
    
    def _load_round_seq(self) -> int:
        """从配置文件加载回合序号（1-4）"""
        try:
            with open(CONFIG["player_id_path"], "r", encoding="utf-8") as f:
                lines = f.readlines()
            for line in lines:
                line = line.strip()
                if line.startswith("ROUND = "):
                    round_seq = int(line.split("=")[1].strip())
                    self._log(f"加载回合配置：第 {round_seq} 回合")
                    return round_seq
            self._log("未找到 ROUND 配置，使用默认回合 4", "warning")
            return 4
        except Exception as e:
            self._log(f"读取回合配置失败：{e}，使用默认回合 4", "warning")
            return 4
    
    def _post_json(self, url: str, payload: Dict[str, Any]) -> Optional[Dict]:
        """发送POST JSON请求"""
        self.stats["total_requests"] += 1
        
        for retry in range(CONFIG["max_retry"]):
            try:
                response = self.session.post(
                    url,
                    headers=self.headers,
                    data=json.dumps(payload, ensure_ascii=False),
                    timeout=CONFIG["timeout"]
                )
                response.raise_for_status()
                self.stats["successful_requests"] += 1
                return response.json()
            except Exception as e:
                self.stats["failed_requests"] += 1
                if retry == CONFIG["max_retry"] - 1:
                    self._log(f"请求失败：{e}", "error")
                    return None
                continue
    
    def fetch(self) -> str:
        """获取用户的回合分数和排名"""
        page = 1
        per_page = 15
        
        self._log(f"开始搜索用户 '{self.target_username}' 在回合 {self.round_seq} (round_id={self.round_id}) 的排名")
        
        # 获取第一页数据
        first_page_data = self._post_json(self.api_url, {"page": page, "round_id": self.round_id})
        if not first_page_data:
            return f"无法获取回合 {self.round_seq} 的数据"
        
        # 解析第一页
        rank_list = first_page_data.get("list", [])
        pagination = first_page_data.get("pagination", {})
        per_page = pagination.get("per_page", 15)
        total_pages = pagination.get("last_page", 1)
        
        self._log(f"回合 {self.round_seq} 总页数：{total_pages}")
        
        # 检查第一页
        for idx, item in enumerate(rank_list):
            userinfo = item.get("userinfo", {})
            username = userinfo.get("username", "")
            if username == self.target_username:
                rank = (page - 1) * per_page + idx + 1
                point = item.get("point", 0)
                self._log(f"✅ 找到用户：{username}，排名 {rank}，回合分数 {point}")
                return f"回合分数：{point}，排名：{rank}"
        
        # 遍历剩余页面
        for page in range(2, total_pages + 1):
            self._log(f"请求第 {page}/{total_pages} 页")
            page_data = self._post_json(self.api_url, {"page": page, "round_id": self.round_id})
            if not page_data:
                continue
            
            rank_list = page_data.get("list", [])
            for idx, item in enumerate(rank_list):
                userinfo = item.get("userinfo", {})
                username = userinfo.get("username", "")
                if username == self.target_username:
                    rank = (page - 1) * per_page + idx + 1
                    point = item.get("point", 0)
                    self._log(f"✅ 找到用户：{username}，排名 {rank}，回合分数 {point}")
                    return f"回合分数：{point}，排名：{rank}"
        
        self._log(f"未找到用户 '{self.target_username}' 在回合 {self.round_seq} 的数据", "warning")
        return f"未找到用户 '{self.target_username}' 在回合 {self.round_seq} 的数据"


def get_round_info(username: str = None, season: int = None, round_num: int = None, callback=None) -> str:
    """获取用户的回合分数和排名"""
    try:
        crawler = RoundCrawler(username, season, round_num, callback)
        result = crawler.fetch()
        return result
    except Exception as e:
        error_msg = f"获取回合信息失败：{str(e)}"
        if callback:
            callback(error_msg, "error")
        else:
            print(f"❌ {error_msg}")
        return error_msg


# CLI入口
if __name__ == "__main__":
    print("=" * 60)
    print("DACreator 回合排名模块 - 命令行版本")
    print("=" * 60)
    
    try:
        result = get_round_info()
        print(f"\n📊 {result}")
    except KeyboardInterrupt:
        print("\n\n⚠️ 用户中断")
    except Exception as e:
        print(f"\n❌ 程序出错：{str(e)}")
    
    input("\n按回车键退出...")