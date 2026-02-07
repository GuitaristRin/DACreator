# DACreator
为头文字D：激斗设计的python程序，用于利用整理好的csv文件快速导出成图片。

B站演示视频<https://www.bilibili.com/video/BV13SFWzTEnv/>
# 快速开始
## 环境要求
- Python 3.7 +
## 原数据文件
如果需要用本地文件生成图片，需要符合以下格式，并保存为csv格式
```csv
コース,ルート,タイム,タイム評価,記録車種,全国順位,記録日
秋名湖,左周り,2'27"760,EXPERT,CIVIC TYPE R (FL5) [HC],255位,2026/01/19
秋名湖,右周り,2'28"702,EXPERT,CIVIC TYPE R (FL5) [HC],121位,2025/12/21
...
```
### 克隆仓库
```shell
https://github.com/GuitaristRin/DACreator.git
```
```shell
cd DACreator
```
### 配置用户数据
用文本编辑器编辑目录下的***Player_id.dat***，输入自己的信息。
### 安装依赖
```shell
pip install -r requirements.txt
```
### 执行程序
```shell
python core.py
```
### 选择功能
进入程序后，输入1并回车以利用爬虫爬取网页数据，在此期间请耐心等待并保持网络畅通，输入2则为本地csv版本。

# 注意事项
## 1.关于rank数据库
由于信息缺乏，用于判断某一记录等级的 ./assets/rank.csv 文件中，有一部分是由ai推算出来的，与真实值有出入，若有准确数值，欢迎上传！
## 2.免责声明
此程序仅供学习参考，严禁用于商业用途！
