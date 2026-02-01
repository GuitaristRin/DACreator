import os
import pandas as pd
from PIL import Image, ImageDraw, ImageFont
import tkinter as tk
from tkinter import filedialog, messagebox

# Global_Config
CONFIG = {
    # Basic_Render_Config
    "font_size": 12,
    "header_height": 40,
    "row_height": 30,
    "col_widths": [80, 60, 60, 80, 100, 220, 90, 80],
    "bg_color": (255, 255, 255),
    "header_color": (44, 62, 80),
    "header_text_color": (255, 255, 255),
    "row_even_color": (245, 245, 245),
    "row_odd_color": (255, 255, 255),
    "text_color": (0, 0, 0),
    "border_color": (200, 200, 200),
    "scale": 2,
    
    # Rank_assets_Config
    "rank_img_root": r".\assets\rank",
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
    
    # Internal_Fonts
    "font_root": r".\assets\font",
    "font_files": {
        "header": "YuGothB.ttc",       # Title
        "special_cols": "consolab.ttf",
        "normal_cols": "msyhbd.ttc" 
    },
    "special_col_names": ["熟練度", "タイム", "記録日"]
}

def select_csv_file():
    """User choose csv"""
    root = tk.Tk()
    root.withdraw()
    file_path = filedialog.askopenfilename(
        title="选择DAC成绩CSV文件",
        filetypes=[("CSV文件", "*.csv"), ("所有文件", "*.*")]
    )
    if not file_path:
        messagebox.showwarning("提示", "未选择CSV文件，程序退出")
        exit()
    return file_path

def select_save_dir():
    """User choose save route"""
    root = tk.Tk()
    root.withdraw()
    save_dir = filedialog.askdirectory(title="选择图片保存目录")
    if not save_dir:
        messagebox.showwarning("提示", "未选择保存目录，程序退出")
        exit()
    return save_dir

def load_csv_data(csv_path):
    """read csv and return to df"""
    try:
        df = pd.read_csv(csv_path, encoding="utf-8")
        required_cols = ["コース", "ルート", "熟練度", "タイム", "タイム評価", "記録車種", "全国順位", "記録日"]
        if not all(col in df.columns for col in required_cols):
            raise ValueError(f"CSV文件缺少必要列，需包含：{required_cols}")
        return df
    except Exception as e:
        messagebox.showerror("错误", f"读取CSV失败：{str(e)}")
        exit()

def load_rank_image(rank_text, target_height):
    """load and scale rank png file"""
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

def load_font(font_type):
    """load hd fonts"""
    font_file = CONFIG["font_files"][font_type]
    font_path = os.path.join(CONFIG["font_root"], font_file)
    
    # check fonts exist
    if not os.path.exists(font_path):
        messagebox.showerror("错误", f"内置字体文件缺失：{font_path}")
        exit()
    
    # load fonts
    try:
        # process ttc
        if font_file.endswith(".ttc"):
            font = ImageFont.truetype(font_path, CONFIG["font_size"] * CONFIG["scale"], index=0)
        else:
            font = ImageFont.truetype(font_path, CONFIG["font_size"] * CONFIG["scale"])
    except Exception as e:
        messagebox.showerror("错误", f"加载字体失败：{font_path}，{str(e)}")
        font = ImageFont.load_default(size=CONFIG["font_size"] * CONFIG["scale"])
    return font

def create_table_image(df):
    """output"""
    # preload fonts
    header_font = load_font("header")
    special_font = load_font("special_cols")
    normal_font = load_font("normal_cols")
    
    # calc picture pixel
    total_width = (sum(CONFIG["col_widths"]) + 20) * CONFIG["scale"]
    total_height = (CONFIG["header_height"] + (len(df) * CONFIG["row_height"]) + 20) * CONFIG["scale"]
    
    # create picture
    img = Image.new("RGB", (total_width, total_height), CONFIG["bg_color"])
    draw = ImageDraw.Draw(img)
    
    # create title
    x = 10 * CONFIG["scale"]
    y = 10 * CONFIG["scale"]
    # title background
    draw.rectangle(
        [x, y, total_width-10*CONFIG["scale"], y+CONFIG["header_height"]*CONFIG["scale"]],
        fill=CONFIG["header_color"],
        outline=CONFIG["border_color"]
    )
    # title text
    headers = df.columns.tolist()
    for i, header in enumerate(headers):
        draw.text(
            (x + 5*CONFIG["scale"], y + (CONFIG["header_height"]*CONFIG["scale"])/2 - (CONFIG["font_size"]*CONFIG["scale"])/2),
            header,
            fill=CONFIG["header_text_color"],
            font=header_font
        )
        x += CONFIG["col_widths"][i] * CONFIG["scale"]
    
    y += CONFIG["header_height"] * CONFIG["scale"]
    eval_col_idx = headers.index("タイム評価")
    
    for idx, (_, row) in enumerate(df.iterrows()):
        row_bg = CONFIG["row_even_color"] if idx%2 == 0 else CONFIG["row_odd_color"]
        draw.rectangle(
            [10*CONFIG["scale"], y, total_width-10*CONFIG["scale"], y+CONFIG["row_height"]*CONFIG["scale"]],
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
                        (x + 5*CONFIG["scale"], y + (CONFIG["row_height"]*CONFIG["scale"])/2 - (CONFIG["font_size"]*CONFIG["scale"])/2),
                        text,
                        fill=CONFIG["text_color"],
                        font=normal_font
                    )
            elif col in CONFIG["special_col_names"]:
                draw.text(
                    (x + 5*CONFIG["scale"], y + (CONFIG["row_height"]*CONFIG["scale"])/2 - (CONFIG["font_size"]*CONFIG["scale"])/2),
                    text,
                    fill=CONFIG["text_color"],
                    font=special_font
                )
            else:
                draw.text(
                    (x + 5*CONFIG["scale"], y + (CONFIG["row_height"]*CONFIG["scale"])/2 - (CONFIG["font_size"]*CONFIG["scale"])/2),
                    text,
                    fill=CONFIG["text_color"],
                    font=normal_font
                )
            
            x += CONFIG["col_widths"][i] * CONFIG["scale"]
        y += CONFIG["row_height"] * CONFIG["scale"]
    
    # return to native
    img = img.resize(
        (total_width//CONFIG["scale"], total_height//CONFIG["scale"]),
        Image.Resampling.LANCZOS
    )
    return img

def main():
    # 校验字体目录和等级图片目录
    if not os.path.exists(CONFIG["font_root"]):
        messagebox.showerror("错误", f"内置字体目录不存在：{CONFIG['font_root']}")
        exit()
    if not os.path.exists(CONFIG["rank_img_root"]):
        messagebox.showerror("错误", f"等级图片目录不存在：{CONFIG['rank_img_root']}")
        exit()
    
    # select csv
    csv_path = select_csv_file()
    # read csv
    df = load_csv_data(csv_path)
    # create table
    table_img = create_table_image(df)
    # save directory
    save_dir = select_save_dir()
    # save
    save_path = os.path.join(save_dir, "DAC_Record.png")
    try:
        table_img.save(save_path, "PNG")
        messagebox.showinfo("成功", f"图片已保存至：{save_path}")
    except Exception as e:
        messagebox.showerror("错误", f"保存图片失败：{str(e)}")

if __name__ == "__main__":
    print("若提示模块不存在，请执行：pip install -r requirements.txt")
    main()

