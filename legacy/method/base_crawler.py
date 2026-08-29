#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
爬虫基类模块
提供所有爬虫模块的公共功能
"""

import json
import os
import sys
from typing import Optional, Dict, Any

# 将项目根目录添加到路径
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from method.spider import ArcadeZoneCrawler, CONFIG


class BaseRankingCrawler(ArcadeZoneCrawler):
    """扩展的爬虫基类，支持任意API端点"""
    
    def __init__(self, username: str = None, season: int = None, callback=None):
        """
        初始化爬虫
        :param username: 用户名，如果为None则从配置文件读取
        :param season: 赛季，如果为None则从配置文件读取
        :param callback: 回调函数，用于GUI进度显示
        """
        super().__init__(username, season, callback)
        # 为子类预留API端点设置
        self.api_endpoint = None
    
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
    
    def _load_config_value(self, key: str, default: Any = None) -> Any:
        """
        从配置文件加载指定键的值
        :param key: 配置键名
        :param default: 默认值
        :return: 配置值
        """
        try:
            with open(CONFIG["player_id_path"], "r", encoding="utf-8") as f:
                lines = f.readlines()
            for line in lines:
                line = line.strip()
                if line.startswith(f"{key} = "):
                    value = line.split("=")[1].strip()
                    self._log(f"加载配置：{key} = {value}")
                    return value
            if default is not None:
                self._log(f"未找到配置项 {key}，使用默认值：{default}", "warning")
                return default
            raise ValueError(f"配置文件中未找到 {key} 行")
        except FileNotFoundError:
            raise Exception(f"未找到配置文件：{CONFIG['player_id_path']}")
        except Exception as e:
            raise Exception(f"读取配置文件失败：{str(e)}")
    
    def _load_round(self) -> int:
        """从配置文件加载回合数"""
        value = self._load_config_value("ROUND")
        return int(value) if value else 0
    
    def _load_team_name(self) -> str:
        """从配置文件加载车队名"""
        return self._load_config_value("TEAM", "")


class PaginationHelper:
    """分页辅助类"""
    
    @staticmethod
    def get_item_by_username(data_list: list, username: str, per_page: int, 
                            page: int, idx: int) -> Optional[tuple]:
        """
        从数据列表中查找指定用户名
        :param data_list: 数据列表
        :param username: 用户名
        :param per_page: 每页数量
        :param page: 当前页码
        :param idx: 当前索引
        :return: (排名, 数据项) 或 None
        """
        for idx_in_page, item in enumerate(data_list):
            userinfo = item.get("userinfo", {})
            if userinfo.get("username") == username:
                rank = (page - 1) * per_page + idx_in_page + 1
                return (rank, item)
        return None
    
    @staticmethod
    def get_item_by_team_name(data_list: list, team_name: str, per_page: int,
                             page: int, idx: int) -> Optional[tuple]:
        """
        从数据列表中查找指定车队名
        :param data_list: 数据列表
        :param team_name: 车队名
        :param per_page: 每页数量
        :param page: 当前页码
        :param idx: 当前索引
        :return: (排名, 数据项) 或 None
        """
        for idx_in_page, item in enumerate(data_list):
            teaminfo = item.get("teaminfo", {})
            if teaminfo.get("team_name") == team_name:
                rank = (page - 1) * per_page + idx_in_page + 1
                return (rank, item)
        return None