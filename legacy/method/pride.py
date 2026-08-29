#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
DACreator 名声排名模块
获取用户的名声值和排名
可作为独立CLI运行，也可被GUI导入
"""

import os
import sys
import json
import requests
from typing import Optional, Dict, Any

# 将项目根目录添加到路径
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, BASE_DIR)

# 直接导入所需的基类和配置，避免循环导入
from method.spider import ArcadeZoneCrawler, CONFIG


class PrideCrawler(ArcadeZoneCrawler):
    """名声排名爬虫类"""
    
    def __init__(self, username: str = None, season: int = None, callback=None):
        """
        初始化名声排名爬虫
        :param username: 用户名，如果为None则从配置文件读取
        :param season: 赛季，如果为None则从配置文件读取
        :param callback: 回调函数，用于GUI进度显示
        """
        super().__init__(username, season, callback)
        # 设置API端点
        self.api_url = "https://arcadezone.cn/ranking/pride"
        self._log("初始化名声排名爬虫")
    
    def _post_json(self, url: str, payload: Dict[str, Any]) -> Optional[Dict]:
        """
        发送POST JSON请求
        :param url: 请求URL
        :param payload: 请求数据
        :return: 响应JSON或None
        """
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
        """
        获取用户的名声值和排名
        :return: 格式化的字符串
        """
        page = 1
        per_page = 15  # 默认值，实际会从响应中获取
        
        self._log(f"开始搜索用户 {self.target_username} 的名声排名")
        
        while True:
            payload = {"page": page, "season": self.season}
            self._log(f"请求第 {page} 页数据")
            
            data = self._post_json(self.api_url, payload)
            if not data:
                break
            
            rank_list = data.get("list", [])
            pagination = data.get("pagination", {})
            per_page = pagination.get("per_page", 15)
            
            # 查找目标用户
            for idx_in_page, item in enumerate(rank_list):
                userinfo = item.get("userinfo", {})
                if userinfo.get("username") == self.target_username:
                    rank = (page - 1) * per_page + idx_in_page + 1
                    pride_point = item.get("pride_point", 0)
                    self._log(f"找到用户：排名 {rank}，名声值 {pride_point}")
                    return f"名声值：{pride_point}，排名：{rank}"
            
            # 检查是否还有下一页
            last_page = pagination.get("last_page", 1)
            if page >= last_page:
                break
            
            page += 1
        
        self._log(f"未找到用户 {self.target_username} 的名声数据", "warning")
        return f"未找到用户 {self.target_username} 的名声数据"


def get_pride_info(username: str = None, season: int = None, callback=None) -> str:
    """
    获取用户的名声值和排名
    :param username: 用户名，为None时从配置文件读取
    :param season: 赛季，为None时从配置文件读取
    :param callback: 回调函数，用于GUI进度显示
    :return: 格式化的字符串
    """
    try:
        crawler = PrideCrawler(username, season, callback)
        result = crawler.fetch()
        return result
    except Exception as e:
        error_msg = f"获取名声信息失败：{str(e)}"
        if callback:
            callback(error_msg, "error")
        else:
            print(f"❌ {error_msg}")
        return error_msg


# CLI入口
if __name__ == "__main__":
    print("=" * 60)
    print("DACreator 名声排名模块 - 命令行版本")
    print("=" * 60)
    
    try:
        result = get_pride_info()
        print(f"\n📊 {result}")
    except KeyboardInterrupt:
        print("\n\n⚠️ 用户中断")
    except Exception as e:
        print(f"\n❌ 程序出错：{str(e)}")
    
    input("\n按回车键退出...")