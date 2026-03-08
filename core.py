import os
import pandas as pd
from PIL import Image, ImageDraw, ImageFont
import tkinter as tk
from tkinter import filedialog, messagebox
import subprocess
import platform
import sys
from typing import Optional
from datetime import datetime
import time

# 导入spider模块的爬取函数
try:
    import spider
except ImportError:
    messagebox.showerror("错误", "未找到spider.py文件，请确保该文件与core.py在同一目录下")
    sys.exit(1)

# 导入新搜索模块（可选）
try:
    import spider_search
    SEARCH_MODULE_AVAILABLE = True
except ImportError:
    SEARCH_MODULE_AVAILABLE = False
    print("⚠️ 未找到 spider_search.py，搜索功能不可用")

# 全局配置
CONFIG = {
    "font_size": 12,
    "header_height": 40,
    "row_height": 30,
    "col_widths": [80, 60, 80, 100, 280, 90, 80],
    "bg_color": (255, 255, 255),
    "header_color": (44, 62, 80),
    "header_text_color": (255, 255, 255),
    "row_even_color": (245, 245, 245),
    "row_odd_color": (255, 255, 255),
    "text_color": (0, 0, 0),
    "border_color": (200, 200, 200),
    "scale": 2,
    
    "rank_img_root": r"./assets/rank",
    "rank_img_scale": 0.8,
    "rank_mapping": {
        "ROOKIE": "rookie.png",
        "REGULAR": "regular.png",
        "SPECIALIST": "specialist.png",
        "EXPERT": "expert.png",
        "PROFESSIONAL": "professional.png",
        "MASTER": "master.png",
        "MASTER+": "masterp.png",
        "LEGEND": "legend.png"
        },
    
    "font_root": r"./assets/font",
    "font_files": {
        "header": "YuGothB.ttc",
        "special_cols": "consolab.ttf",
        "normal_cols": "msyhbd.ttc"
    },
    "special_col_names": ["タイム", "記録日"],
}

def format_time(seconds: float) -> str:
    """格式化时间为 分'秒"毫秒 格式"""
    minutes = int(seconds // 60)
    seconds_remainder = seconds % 60
    whole_seconds = int(seconds_remainder)
    milliseconds = int((seconds_remainder - whole_seconds) * 1000)
    
    if minutes > 0:
        return f"{minutes}'{whole_seconds:02d}\"{milliseconds:03d}"
    else:
        return f"{whole_seconds}\"{milliseconds:03d}"

def get_timestamp() -> str:
    """获取当前时间戳，格式：YYYYMMDD_HHMMSS"""
    return datetime.now().strftime("%Y%m%d_%H%M%S")

def ping_arcadezone() -> bool:
    """检查网络连接"""
    host = "arcadezone.cn"
    param = "-n" if platform.system().lower() == "windows" else "-c"
    command = ["ping", param, "4", host]
    
    try:
        output = subprocess.run(
            command,
            capture_output=True,
            text=True,
            timeout=10
        )
        if (platform.system().lower() == "windows" and "TTL=" in output.stdout) or \
           (platform.system().lower() != "windows" and "0% packet loss" in output.stdout):
            return True
        else:
            return False
    except Exception:
        return False

def select_csv_file() -> str:
    """选择CSV文件"""
    root = tk.Tk()
    root.withdraw()
    file_path = filedialog.askopenfilename(
        title="选择DAC成绩CSV文件",
        filetypes=[("CSV文件", "*.csv"), ("所有文件", "*.*")]
    )
    if not file_path:
        messagebox.showwarning("提示", "未选择CSV文件，程序退出")
        sys.exit(1)
    return file_path

def select_save_dir() -> str:
    """选择保存目录"""
    root = tk.Tk()
    root.withdraw()
    save_dir = filedialog.askdirectory(title="选择图片保存目录")
    if not save_dir:
        messagebox.showwarning("提示", "未选择保存目录，程序退出")
        sys.exit(1)
    return save_dir

def load_csv_data(csv_path: str) -> pd.DataFrame:
    """加载CSV数据（标准格式，含排名列）"""
    try:
        df = pd.read_csv(csv_path, encoding="utf-8-sig")
        required_cols = ["コース", "ルート", "タイム", "タイム評価", "記録車種", "全国順位", "記録日"]
        if not all(col in df.columns for col in required_cols):
            raise ValueError(f"CSV文件缺少必要列，需包含：{required_cols}")
        return df
    except Exception as e:
        messagebox.showerror("错误", f"读取CSV失败：{str(e)}")
        sys.exit(1)

def load_csv_data_no_rank(csv_path: str) -> pd.DataFrame:
    """加载无排名列的CSV数据（搜索模式专用）"""
    try:
        df = pd.read_csv(csv_path, encoding="utf-8-sig")
        required_cols = ["コース", "ルート", "タイム", "タイム評価", "記録車種", "記録日"]
        if not all(col in df.columns for col in required_cols):
            raise ValueError(f"CSV文件缺少必要列，需包含：{required_cols}")
        return df
    except Exception as e:
        messagebox.showerror("错误", f"读取CSV失败：{str(e)}")
        sys.exit(1)

def load_rank_image(rank_text: str, target_height: int) -> Optional[Image.Image]:
    """加载等级图片"""
    rank_text_upper = rank_text.strip().upper()
    if rank_text_upper not in CONFIG["rank_mapping"]:
        return None
    
    img_name = CONFIG["rank_mapping"][rank_text_upper]
    img_path = os.path.join(CONFIG["rank_img_root"], img_name)
    if not os.path.exists(img_path):
        messagebox.showwarning("提示", f"等级图片不存在：{img_path}")
        return None
    
    img = Image.open(img_path).convert("RGBA")
    original_w, original_h = img.size
    final_row_height = CONFIG["row_height"] * CONFIG["rank_img_scale"]
    scale_ratio = final_row_height / original_h
    new_w = int(original_w * scale_ratio * CONFIG["scale"])
    new_h = int(original_h * scale_ratio * CONFIG["scale"])
    
    img_resized = img.resize((new_w, new_h), Image.Resampling.LANCZOS)
    return img_resized

def load_font(font_type: str) -> ImageFont.FreeTypeFont:
    """加载字体"""
    font_file = CONFIG["font_files"][font_type]
    font_path = os.path.join(CONFIG["font_root"], font_file)
    
    if not os.path.exists(font_path):
        messagebox.showerror("错误", f"内置字体文件缺失：{font_path}")
        sys.exit(1)
    
    try:
        if font_file.endswith(".ttc"):
            font = ImageFont.truetype(font_path, CONFIG["font_size"] * CONFIG["scale"], index=0)
        else:
            font = ImageFont.truetype(font_path, CONFIG["font_size"] * CONFIG["scale"])
    except Exception as e:
        messagebox.showerror("错误", f"加载字体失败：{font_path}，{str(e)}")
        font = ImageFont.load_default(size=CONFIG["font_size"] * CONFIG["scale"])
    return font

def create_table_image(df: pd.DataFrame) -> Image.Image:
    """创建表格图片（兼容有无排名列两种情况）"""
    header_font = load_font("header")
    special_font = load_font("special_cols")
    normal_font = load_font("normal_cols")
    
    # 根据实际列数调整列宽
    actual_cols = len(df.columns)
    if actual_cols == 6:  # 无排名列
        col_widths = CONFIG["col_widths"][:5] + [CONFIG["col_widths"][6]]  # 去掉排名列宽度
    else:  # 7列（含排名）
        col_widths = CONFIG["col_widths"]
    
    total_width = (sum(col_widths) + 20) * CONFIG["scale"]
    total_height = (CONFIG["header_height"] + (len(df) * CONFIG["row_height"]) + 20) * CONFIG["scale"]
    
    img = Image.new("RGB", (total_width, total_height), CONFIG["bg_color"])
    draw = ImageDraw.Draw(img)
    
    # 绘制表头
    x = 10 * CONFIG["scale"]
    y = 10 * CONFIG["scale"]
    draw.rectangle(
        [x, y, total_width - 10 * CONFIG["scale"], y + CONFIG["header_height"] * CONFIG["scale"]],
        fill=CONFIG["header_color"],
        outline=CONFIG["border_color"]
    )
    headers = df.columns.tolist()
    for i, header in enumerate(headers):
        draw.text(
            (x + 5 * CONFIG["scale"], y + (CONFIG["header_height"] * CONFIG["scale"]) / 2 - (CONFIG["font_size"] * CONFIG["scale"]) / 2),
            header,
            fill=CONFIG["header_text_color"],
            font=header_font
        )
        x += col_widths[i] * CONFIG["scale"]
    
    # 绘制数据行
    y += CONFIG["header_height"] * CONFIG["scale"]
    eval_col_idx = headers.index("タイム評価") if "タイム評価" in headers else -1
    
    for idx, (_, row) in enumerate(df.iterrows()):
        row_bg = CONFIG["row_even_color"] if idx % 2 == 0 else CONFIG["row_odd_color"]
        draw.rectangle(
            [10 * CONFIG["scale"], y, total_width - 10 * CONFIG["scale"], y + CONFIG["row_height"] * CONFIG["scale"]],
            fill=row_bg,
            outline=CONFIG["border_color"]
        )
        
        x = 10 * CONFIG["scale"]
        for i, col in enumerate(headers):
            text = str(row[col]) if pd.notna(row[col]) else ""
            
            if i == eval_col_idx:
                rank_img = load_rank_image(text, 0)
                if rank_img:
                    img_x = x + (col_widths[i] * CONFIG["scale"] - rank_img.width) // 2
                    img_y = y + (CONFIG["row_height"] * CONFIG["scale"] - rank_img.height) // 2
                    img.paste(rank_img, (img_x, img_y), mask=rank_img)
                else:
                    draw.text(
                        (x + 5 * CONFIG["scale"], y + (CONFIG["row_height"] * CONFIG["scale"]) / 2 - (CONFIG["font_size"] * CONFIG["scale"]) / 2),
                        text,
                        fill=CONFIG["text_color"],
                        font=normal_font
                    )
            elif col in CONFIG["special_col_names"]:
                draw.text(
                    (x + 5 * CONFIG["scale"], y + (CONFIG["row_height"] * CONFIG["scale"]) / 2 - (CONFIG["font_size"] * CONFIG["scale"]) / 2),
                    text,
                    fill=CONFIG["text_color"],
                    font=special_font
                )
            else:
                draw.text(
                    (x + 5 * CONFIG["scale"], y + (CONFIG["row_height"] * CONFIG["scale"]) / 2 - (CONFIG["font_size"] * CONFIG["scale"]) / 2),
                    text,
                    fill=CONFIG["text_color"],
                    font=normal_font
                )
            
            x += col_widths[i] * CONFIG["scale"]
        y += CONFIG["row_height"] * CONFIG["scale"]
    
    # 缩小回正常尺寸
    img = img.resize(
        (total_width // CONFIG["scale"], total_height // CONFIG["scale"]),
        Image.Resampling.LANCZOS
    )
    return img

def get_username_from_file() -> str:
    """从Player_ID.dat获取用户名"""
    try:
        with open("Player_ID.dat", "r", encoding="utf-8") as f:
            lines = f.readlines()
            for line in lines:
                line = line.strip()
                if line.startswith("ID = "):
                    return line.split("=")[1].strip()
    except:
        pass
    return "未知用户"

# 主函数
def main():
    # 校验资源目录
    if not os.path.exists(CONFIG["font_root"]):
        messagebox.showerror("错误", f"内置字体目录不存在：{CONFIG['font_root']}")
        sys.exit(1)
    if not os.path.exists(CONFIG["rank_img_root"]):
        messagebox.showerror("错误", f"等级图片目录不存在：{CONFIG['rank_img_root']}")
        sys.exit(1)
    
    # 功能选择
    print("\n===== DAC成绩表生成工具 =====")
    print("1. 爬取ArcadeZone用户数据并生成可视化表格图片（含排名）")
    print("2. 本地CSV文件生成可视化表格图片")
    if SEARCH_MODULE_AVAILABLE:
        print("3. 通过用户名搜索生成无排名表格图片（快速模式）")
    choice = input(f"请选择功能（1/2{'/3' if SEARCH_MODULE_AVAILABLE else ''}）：").strip()
    
    root = tk.Tk()
    root.withdraw()
    
    valid_choices = ["1", "2"]
    if SEARCH_MODULE_AVAILABLE:
        valid_choices.append("3")
    
    if choice not in valid_choices:
        messagebox.showerror("错误", f"无效选择，程序退出（有效选项：{', '.join(valid_choices)}）")
        sys.exit(1)
    
    # 获取时间戳（统一命名使用）
    timestamp = get_timestamp()
    
    # 记录开始时间（在用户选择保存目录之前）
    start_time = time.time()
    
    # 功能1：传统爬虫模式（含排名）
    if choice == "1":
        if not ping_arcadezone():
            messagebox.showerror("错误", "网络连接异常，无法访问ArcadeZone，请检查网络")
            sys.exit(1)
        
        print("\n📡 开始执行爬虫任务（遍历所有赛道）...")
        df = spider.crawl_data()
        
        if df.empty:
            messagebox.showerror("错误", "未爬取到任何成绩数据")
            sys.exit(1)
        
        # 计算爬虫耗时（在选择保存目录之前）
        crawl_time = time.time() - start_time
        print(f"⏱️ 数据爬取完成，耗时 {format_time(crawl_time)}")
        
        save_dir = select_save_dir()
        
        # 统一命名：DAC成绩表_时间戳
        base_filename = f"DAC成绩表_{timestamp}"
        
        # 保存CSV
        try:
            csv_filename = f"{base_filename}.csv"
            csv_path = os.path.join(save_dir, csv_filename)
            df.to_csv(csv_path, index=False, encoding="utf-8-sig")
            print(f"✅ CSV文件已保存至：{csv_path}")
        except Exception as e:
            messagebox.showwarning("提示", f"CSV保存失败：{str(e)}，继续生成图片")
            csv_path = None
        
        # 生成图片
        try:
            print("🎨 开始生成可视化表格图片...")
            table_img = create_table_image(df)
            img_filename = f"{base_filename}.png"
            img_path = os.path.join(save_dir, img_filename)
            table_img.save(img_path, "PNG", dpi=(300, 300))
            
            # 计算总耗时
            total_time = time.time() - start_time
            print(f"✅ 完成！总耗时 {format_time(total_time)}")
            
            messagebox.showinfo("成功", f"""
✅ 任务完成！
- 爬取到 {len(df)} 条成绩数据
- CSV文件路径：{csv_path if 'csv_path' in locals() else '未保存'}
- 图片文件路径：{img_path}
- 总耗时：{format_time(total_time)}
            """)
        except Exception as e:
            messagebox.showerror("错误", f"生成图片失败：{str(e)}")
            sys.exit(1)
    
    # 功能2：本地CSV模式（标准格式，含排名）
    elif choice == "2":
        csv_path = select_csv_file()
        df = load_csv_data(csv_path)
        
        # 计算读取CSV耗时
        read_time = time.time() - start_time
        print(f"⏱️ CSV文件读取完成，耗时 {format_time(read_time)}")
        
        save_dir = select_save_dir()
        
        # 统一命名：DAC成绩表_时间戳.png
        img_filename = f"DAC成绩表_{timestamp}.png"
        img_path = os.path.join(save_dir, img_filename)
        
        try:
            print("🎨 开始生成可视化表格图片...")
            table_img = create_table_image(df)
            table_img.save(img_path, "PNG", dpi=(300, 300))
            
            # 计算总耗时
            total_time = time.time() - start_time
            print(f"✅ 完成！总耗时 {format_time(total_time)}")
            
            messagebox.showinfo("成功", f"""
✅ 任务完成！
- 表格图片已保存至：{img_path}
- 总耗时：{format_time(total_time)}
            """)
        except Exception as e:
            messagebox.showerror("错误", f"保存图片失败：{str(e)}")
            sys.exit(1)
    
    # 功能3：搜索模式（无排名）
    elif choice == "3" and SEARCH_MODULE_AVAILABLE:
        if not ping_arcadezone():
            messagebox.showerror("错误", "网络连接异常，无法访问ArcadeZone")
            sys.exit(1)
        
        print("\n📡 开始执行搜索爬虫任务（快速模式）...")
        df = spider_search.crawl_data_by_search()
        
        if df.empty:
            messagebox.showerror("错误", "未搜索到任何成绩数据")
            sys.exit(1)
        
        # 计算搜索耗时（在选择保存目录之前）
        search_time = time.time() - start_time
        print(f"⏱️ 数据搜索完成，耗时 {format_time(search_time)}")
        
        save_dir = select_save_dir()
        
        # 统一命名：DAC成绩表_时间戳
        base_filename = f"DAC成绩表_{timestamp}"
        
        # 保存CSV（无排名列）
        try:
            csv_filename = f"{base_filename}.csv"
            csv_path = os.path.join(save_dir, csv_filename)
            df.to_csv(csv_path, index=False, encoding="utf-8-sig")
            print(f"✅ CSV文件已保存至：{csv_path}")
        except Exception as e:
            messagebox.showwarning("提示", f"CSV保存失败：{str(e)}，继续生成图片")
            csv_path = None
        
        # 生成图片
        try:
            print("🎨 开始生成可视化表格图片...")
            table_img = create_table_image(df)
            img_filename = f"{base_filename}.png"
            img_path = os.path.join(save_dir, img_filename)
            table_img.save(img_path, "PNG", dpi=(300, 300))
            
            # 计算总耗时
            total_time = time.time() - start_time
            print(f"✅ 完成！总耗时 {format_time(total_time)}")
            
            messagebox.showinfo("成功", f"""
✅ 任务完成！
- 搜索到 {len(df)} 条成绩数据
- CSV文件路径：{csv_path if 'csv_path' in locals() else '未保存'}
- 图片文件路径：{img_path}
- 总耗时：{format_time(total_time)}
            """)
        except Exception as e:
            messagebox.showerror("错误", f"生成图片失败：{str(e)}")
            sys.exit(1)

if __name__ == "__main__":
    print("若提示模块不存在，请执行：pip install -r requirements.txt")
    print("提示：如需使用搜索功能，请确保 spider_search.py 在同一目录下")
    main()