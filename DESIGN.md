---
name: V2rayDash
description: 私有代理服务管理面板，莫兰迪色彩体系，温润精致的工具美学
colors:
  dusty-rose: "#C9A9A6"
  dusty-rose-hover: "#B89591"
  sage: "#A8B5A0"
  sky: "#9DB4C0"
  lavender: "#B4A7C7"
  sand: "#D4C5B5"
  cream: "#F5F0E8"
  terracotta: "#C4836A"
  error-red: "#D68B8B"
  warm-gray: "#9E9A93"
  bg-primary: "#FAF8F5"
  bg-secondary: "#F0EDE8"
  bg-card: "#FFFFFF"
  bg-sidebar: "#2C2A28"
  text-primary: "#3D3A36"
  text-secondary: "#6B6760"
  text-muted: "#9E9A93"
  border: "#E5E0D8"
typography:
  heading-page:
    fontFamily: "'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "24px"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "normal"
  heading-section:
    fontFamily: "'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "20px"
    fontWeight: 500
    lineHeight: 1.4
  heading-card:
    fontFamily: "'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "16px"
    fontWeight: 500
    lineHeight: 1.5
  body:
    fontFamily: "'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: "'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  sm: "4px"
  md: "10px"
  lg: "12px"
  xl: "16px"
  full: "50%"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "20px"
  xl: "28px"
  xxl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.dusty-rose}"
    textColor: "#FFFFFF"
    rounded: "{rounded.md}"
    padding: "0 16px"
    height: "40px"
  button-primary-hover:
    backgroundColor: "{colors.dusty-rose-hover}"
    textColor: "#FFFFFF"
    rounded: "{rounded.md}"
  button-default:
    backgroundColor: "{colors.bg-card}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.md}"
    padding: "0 16px"
    height: "40px"
  card:
    backgroundColor: "{colors.bg-card}"
    rounded: "{rounded.lg}"
    padding: "{spacing.lg}"
  tag:
    backgroundColor: "{colors.bg-secondary}"
    textColor: "{colors.text-secondary}"
    rounded: "{rounded.sm}"
    padding: "4px 10px"
  input:
    backgroundColor: "{colors.bg-card}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.md}"
    height: "40px"
---

# Design System: V2rayDash

## 1. Overview

**Creative North Star: "石间清泉"**

V2rayDash 的设计语言源于莫兰迪的色彩哲学——低饱和、高自持，用克制的色彩叙述而非高声呼喊传达信息。就像山石间渗出的清泉，安静、清澈、有温度，这个界面选择"不打扰"作为最高设计原则。用户打开面板是为了完成任务，不是为了欣赏界面；于是这套系统让信息自己说话，让色彩成为结构而非装饰。

主体以暖白（#FAF8F5）为底，用深棕近黑的侧边栏（#2C2A28）形成稳定的空间锚点，暮玫粉（#C9A9A6）作为唯一互动色，在表单、按钮、激活状态中出现，其余色彩——山岚绿、烟霭蓝、淡砾灰——只用于语义标注（成功、信息、状态）。整个调色板像一幅褪色的水彩，笔触浅淡，但层次清晰。

这套设计系统明确拒绝：过度营销感的强渐变与大对比；游戏/赛博朋克的霓虹发光与炫技暗色；企业 ERP 的蓝灰密集；过于可爱的消费级 App 粉色。

**Key Characteristics:**
- 低饱和莫兰迪色系，整体视觉噪音极低
- 深色侧边栏 + 暖白内容区双层空间结构（未来可探索全亮主题）
- 暮玫粉单一互动色——稀少即力量
- Noto Sans SC 单字族承载所有层级，克制优雅
- 信息密度适中：工具性优先，留白不奢侈也不拥挤

## 2. Colors: 石间五色

莫兰迪调色不是"灰"，而是所有颜色都往中灰靠拢后各自保留的那一点色相记忆。

### Primary
- **暮玫粉** (#C9A9A6): 唯一互动色。主按钮、激活导航项边框、表单前缀图标、品牌名称重音字。出现面积 ≤10%，其稀少性是其力量所在。
- **玫粉悬停** (#B89591): 暮玫粉的悬停态，明度下移约 5%，无色相偏移。

### Secondary
- **山岚灰绿** (#A8B5A0): 成功/在线状态。状态指示点、成功反馈。不作装饰用。
- **烟霭蓝** (#9DB4C0): 信息/中性状态。信息标签、部分图表轨道色。
- **陶燃橙** (#C4836A): 警告色。也是系统高负载时环形图的填充色阈值变化信号。

### Tertiary
- **暮红** (#D68B8B): 错误/离线状态。错误边框、错误文案、离线状态点。
- **淡紫** (#B4A7C7): 辅助标签色。用于区分标签组时作第四个语义色。

### Neutral
- **细沙米** (#D4C5B5): 边界辅助色，用于细节分隔。
- **浅棉白** (#F5F0E8): 卡片头部渐变起点，modal 标题区背景。
- **暖砾灰** (#9E9A93): 次要文本、占位符、滚动条 thumb、状态点静默色。
- **边框灰** (#E5E0D8): 表格分隔线、卡片边框、分割线。
- **内容底** (#FAF8F5): 页面主背景，1°暖调倾向。
- **次级底** (#F0EDE8): 表格标题背景、悬停行填色的极淡版本。
- **白面** (#FFFFFF): 卡片、Modal、输入框本体。
- **主文** (#3D3A36): 正文、标题，深棕近黑，避免纯黑过于锐利。
- **副文** (#6B6760): 表单 label、次级描述，与内容底对比度务必 ≥4.5:1。
- **侧边栏** (#2C2A28): 导航底色，近黑棕，为暮玫粉激活色提供深色背景烘托。

### Named Rules
**克制用色法则。** 暮玫粉在任一屏幕的出现面积不超过 10%。山岚绿、烟霭蓝只服务语义状态，不作装饰。如果不确定是否该用彩色，答案是否定的。

**暖灰对比警戒线。** 副文 #6B6760 对内容底 #FAF8F5 的对比度约为 4.6:1，刚刚达标。任何在此之上叠加背景色（卡片、次级底）的场景，都需要重新核验副文对比度，不允许降到 4.5:1 以下。

## 3. Typography

**Display/Body Font:** Noto Sans SC（主字体）
**Fallback Stack:** -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif

**Character:** 单字族配置，用字重（400/500/600）而非字体切换建立层级。Noto Sans SC 对中文字符有良好支持，在低饱和界面中提供稳定、有温度但不抢眼的质感。无衬线保证了数据密集型界面的扫描效率。

### Hierarchy
- **页面标题**（600, 24px, 行高 1.3）: 每页顶部 H1，告知用户所在。间距下方 8px 接副标题。
- **区块标题**（500, 20px, 行高 1.4）: 页面内模块/卡片组的主标题，如 header `<h2>`。
- **卡片/Modal 标题**（500, 16px, 行高 1.5）: Ant Design Card 的 `ant-card-head-title`，Modal 标题。
- **正文/表格数据**（400, 14px, 行高 1.6）: 表格单元格、表单描述、段落内容。
- **标签/辅助**（400, 12px, 行高 1.5）: Tag、小型说明文字、时间戳。

### Named Rules
**单族原则。** 不引入第二字体。标题与正文的区分依赖字重和尺寸，而非字体切换。额外字体 = 额外负载 + 视觉噪音，两者在私有工具中都不必要。

**中文行长克制。** 正文段落建议 30–40 个中文字符（约 60–80ch 等效），超出时使用 `text-wrap: pretty` 控制孤词。

## 4. Elevation

系统采用"克制分层"策略：大多数表面平铺无阴影，阴影只在响应状态变化（悬停、模态弹出）时出现，作为可交互性或浮动层级的信号，而非装饰。

### Shadow Vocabulary
- **软阴影 soft** (`0 2px 12px rgba(60, 55, 48, 0.08)`): 卡片默认态。极轻，几乎不可见，仅将卡片与背景区分。
- **提升阴影 elevated** (`0 8px 24px rgba(60, 55, 48, 0.12)`): 卡片悬停态 / Modal / 下拉面板。传达"我在更高一层"的层级感。

阴影颜色取自文字色系（暗棕 rgb(60,55,48)），而非黑色，保持与整体暖调的一致性。

### Named Rules
**静息无阴影法则。** 不悬停时，大多数容器以极轻的 soft 阴影存在或直接平铺（靠边框 `var(--border-color)` 区分）。阴影不是装饰，是状态的声音。

## 5. Components

### Buttons
- **Shape:** 圆角 10px（`{rounded.md}`），不用全圆或方角
- **Primary:** 暮玫粉底色 (#C9A9A6)、白色文字、高度 40px、内边距 0 16px；hover 时底色切换至 #B89591，0.3s ease 过渡
- **Default:** 白底、主文色字体、1px 边框 #E5E0D8；hover 时边框色倾向暮玫粉
- **危险/删除:** 使用 Ant Design danger 变体，暮红 (#D68B8B) 底色
- **尺寸:** `size="large"` 用于表单提交，`size="middle"` 用于表格操作列

### Cards / Containers
- **Corner Style:** 圆角 12px (`{rounded.lg}`)
- **Background:** 白面 (#FFFFFF)
- **Shadow Strategy:** 默认 soft 阴影，hover 切换至 elevated 并 `translateY(-2px)`；过渡 0.3s ease
- **Border:** 1px solid #E5E0D8
- **卡片头部:** `linear-gradient(135deg, #F5F0E8, #F0EDE8)` 渐变底，强化莫兰迪质感
- **Internal Padding:** 20px (body)，16px 20px (头部)

### Inputs / Fields
- **Style:** 白底，1px 边框，圆角 10px；中文占位符用暖砾灰 (#9E9A93)
- **Focus:** Ant Design 默认蓝色 focus ring 被暮玫粉调色取代（通过 CSS override）
- **Prefix Icon:** 暮玫粉 (#C9A9A6) 配色
- **Error:** 暮红 (#D68B8B) 边框 + 下方错误文案

### Navigation (Sidebar)
- **Background:** #2C2A28（深棕近黑）
- **Default item:** rgba(255,255,255,0.6) 文字，transparent 背景，3px solid transparent 左边框
- **Hover:** 白色文字，rgba(255,255,255,0.05) 背景
- **Active:** 浅棉白 (#F5F0E8) 文字，rgba(201,169,166,0.15) 背景，3px solid #C9A9A6 左边框
- **Logo:** "V2" + 暮玫粉品牌重音字，18px 500 weight，letter-spacing 1px
- **Icon size:** 20×20px，opacity 0.8 降低视觉重量

### Tags / Chips
- **Style:** 圆角 6px，padding 4px 10px，12px 字体
- **状态语义:** 绿色=在线/成功，暮红=错误，橙=警告，灰=未知
- **不用纯色块标签** 传达状态，而是配合 Tag 组件的边框模式

### Tables
- **Container:** 圆角 12px，overflow hidden，白面背景
- **Header:** 次级底 (#F0EDE8) 背景，主文色，500 weight
- **Row divider:** #E5E0D8 细线
- **Hover row:** rgba(201,169,166,0.05) 极淡玫粉填色
- **展开行:** `padding-inline: 0` 使图表填满全宽

### Metric Ring（环形指标卡）
系统监控页的标志性组件。SVG 圆弧 270°弧度，轨道色 rgba(60,55,48,0.10)，填充色根据负载动态切换：正常=山岚绿，中等=细沙米，高负载=陶燃橙。过渡用 `cubic-bezier(.4,0,.2,1)` 1s 缓动。

## 6. Do's and Don'ts

### Do:
- **Do** 用暮玫粉 (#C9A9A6) 作为唯一互动 / 品牌色，在一个屏幕内将其出现面积控制在 10% 以内
- **Do** 在副文 (#6B6760) 叠加到次级底 (#F0EDE8) 等浅色背景前核验对比度 ≥4.5:1
- **Do** 用字重 (400/500/600) 而非字体切换建立文字层级
- **Do** 用 `rgba(60,55,48,...)` 系暖棕透明阴影，而非黑色透明阴影
- **Do** 用语义色（山岚绿/暮红/陶燃橙）只标注状态，不作纯装饰用途
- **Do** 卡片悬停时配合 `translateY(-2px)` + elevated 阴影，提供微触感反馈
- **Do** 保持圆角一致性：12px 容器 / 10px 控件 / 6px 标签

### Don't:
- **Don't** 用强渐变或强对比色块——这是"过度营销感"的核心特征，与莫兰迪美学根本对立
- **Don't** 引入霓虹色、发光效果或高饱和暗色主题——游戏/赛博朋克风格在此处是明确的反面参考
- **Don't** 使用蓝灰主色调搭配密集无间距表格——ERP 风格，破坏工具的私人质感
- **Don't** 使用卡通图标、粉嫩配色、可爱插图——这是消费级 App 语言，不是私人工具的调性
- **Don't** 在激活/非激活状态使用重色彩填充——非激活元素保持低饱和或无色，彩色是行动信号
- **Don't** 用 `border-left` > 1px 彩色条纹装饰卡片或列表项——用背景色块或整圈边框代替
- **Don't** 用 `background-clip: text` 渐变字——彩虹文字是装饰语言，莫兰迪系统里没有这个位置
- **Don't** 用全饱和红/绿做状态色——用暮红 (#D68B8B)、山岚绿 (#A8B5A0) 等莫兰迪化后的语义色
- **Don't** 嵌套卡片——卡片中的卡片是永远错误的；用内边距、分隔线或列表行区分内容
