# DACreator
为头文字D：激斗设计的python程序，用于利用整理好的csv文件快速导出成图片。
# 快速开始
## 环境要求
- Python 3.7 +
- 依赖库：Pandas,Pillow
## 原数据文件
需要符合以下格式，并保存为csv格式
```csv
コース,ルート,熟練度,タイム,タイム評価,記録車種,全国順位,記録日
秋名湖,左周り,2,2'27"760,EXPERT,CIVIC TYPE R (FL5) [HC],255位,2026/01/19
秋名湖,右周り,1,2'28"702,EXPERT,CIVIC TYPE R (FL5) [HC],121位,2025/12/21
...
```
### 克隆仓库
```shell
https://github.com/GuitaristRin/DACreator.git
```
```shell
cd DACreator
```
### 安装依赖
```shell
pip install -r requirements.txt
```
### 执行程序
```shell
python main.py
```
通过弹出窗口选择准备好的csv文件和欲存放的目录
