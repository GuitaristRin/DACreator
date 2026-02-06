import os
import pandas as pd
from PIL import Image, ImageDraw, ImageFont
import tkinter as tk
from tkinter import filedialog, messagebox
import subprocess
import platform
import sys
from typing import Optional

# 导入spider模块的爬取函数
try:
    import spider
except ImportError:
    messagebox.showerror("错误", "未找到spider.py文件，请确保该文件与core.py在同一目录下")
    sys.exit(1)

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

def ping_arcadezone() -> bool:
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
    root = tk.Tk()
    root.withdraw()
    save_dir = filedialog.askdirectory(title="选择图片保存目录")
    if not save_dir:
        messagebox.showwarning("提示", "未选择保存目录，程序退出")
        sys.exit(1)
    return save_dir

def load_csv_data(csv_path: str) -> pd.DataFrame:
    try:
        df = pd.read_csv(csv_path, encoding="utf-8-sig")
        required_cols = ["コース", "ルート", "タイム", "タイム評価", "記録車種", "全国順位", "記録日"]
        if not all(col in df.columns for col in required_cols):
            raise ValueError(f"CSV文件缺少必要列，需包含：{required_cols}")
        return df
    except Exception as e:
        messagebox.showerror("错误", f"读取CSV失败：{str(e)}")
        sys.exit(1)

def load_rank_image(rank_text: str, target_height: int) -> Optional[Image.Image]:
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
    header_font = load_font("header")
    special_font = load_font("special_cols")
    normal_font = load_font("normal_cols")
    
    total_width = (sum(CONFIG["col_widths"]) + 20) * CONFIG["scale"]
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
        x += CONFIG["col_widths"][i] * CONFIG["scale"]
    
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
                    img_x = x + (CONFIG["col_widths"][i] * CONFIG["scale"] - rank_img.width) // 2
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
            
            x += CONFIG["col_widths"][i] * CONFIG["scale"]
        y += CONFIG["row_height"] * CONFIG["scale"]
    
    img = img.resize(
        (total_width // CONFIG["scale"], total_height // CONFIG["scale"]),
        Image.Resampling.LANCZOS
    )
    return img

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
    print("1. 爬取ArcadeZone用户数据并生成可视化表格图片")
    print("2. 本地CSV文件生成可视化表格图片")
    choice = input("请选择功能（1/2）：").strip()
    
    root = tk.Tk()
    root.withdraw()
    if choice not in ["1", "2"]:
        messagebox.showerror("错误", "无效选择，程序退出")
        sys.exit(1)
    
    if choice == "1":
        if not ping_arcadezone():
            messagebox.showerror("错误", "网络连接异常，无法访问ArcadeZone，请检查网络")
            sys.exit(1)
        
        print("\n📡 开始执行爬虫任务...")
        df = spider.crawl_data()
        
        if df.empty:
            messagebox.showerror("错误", "未爬取到任何成绩数据")
            sys.exit(1)
        
        save_dir = select_save_dir()
        
        try:
            with open("Player_ID.dat", "r", encoding="utf-8") as f:
                line = f.readline().strip()
                username = line.split("ID = ")[1].strip() if line.startswith("ID = ") else "未知用户"
            csv_filename = f"DAC_{username}_成绩表.csv"
            csv_path = os.path.join(save_dir, csv_filename)
            df.to_csv(csv_path, index=False, encoding="utf-8-sig")
            print(f"✅ CSV文件已保存至：{csv_path}")
        except Exception as e:
            messagebox.showwarning("提示", f"CSV保存失败：{str(e)}，继续生成图片")
        
        try:
            print("🎨 开始生成可视化表格图片...")
            table_img = create_table_image(df)
            img_path = os.path.join(save_dir, "DAC成绩表_可视化.png")
            table_img.save(img_path, "PNG", dpi=(300, 300))
            messagebox.showinfo("成功", f"""
✅ 任务完成！
- 爬取到{len(df)}条成绩数据
- CSV文件路径：{csv_path if 'csv_path' in locals() else '未保存'}
- 图片文件路径：{img_path}
            """)
        except Exception as e:
            messagebox.showerror("错误", f"生成图片失败：{str(e)}")
            sys.exit(1)
    
    elif choice == "2":
        csv_path = select_csv_file()
        df = load_csv_data(csv_path)
        table_img = create_table_image(df)
        save_dir = select_save_dir()
        img_path = os.path.join(save_dir, "DAC成绩表_可视化.png")
        try:
            table_img.save(img_path, "PNG", dpi=(300, 300))
            messagebox.showinfo("成功", f"表格图片已保存至：\n{img_path}")
        except Exception as e:
            messagebox.showerror("错误", f"保存图片失败：{str(e)}")
            sys.exit(1)

if __name__ == "__main__":
    print("若提示模块不存在，请执行：pip install -r requirements.txt")
    main()
