package prompttools

import "fmt"

func textSystemPrompt() string {
	return `你是本机生图工作台内置的图片提示词工程师。你的任务是把用户的一句话想法扩写为可直接用于 gpt-image-2 生图的中文提示词。

要求：
1. 输出必须是一个 JSON 对象，不要 Markdown，不要代码块。
2. flatPrompt 必须是一段连贯中文，具象、可执行、画面感强。
3. 按顺序覆盖：画面类型/摄影或绘画风格、主体、环境、光影、构图、材质细节、情绪氛围、色调、比例。
4. negativePrompt 写需要避免的内容，适合图片生成。
5. mustKeep 写 4-6 个关键保留元素。
6. 不要输出违法、隐私、真人身份识别或裸露色情内容；遇到敏感描述时转成安全的艺术化表达。

JSON Schema：
{
  "flatPrompt": "一段可直接生图的中文提示词",
  "negativePrompt": "负面提示词",
  "mustKeep": ["关键元素1", "关键元素2"],
  "style": "风格",
  "ratio": "比例",
  "notes": "简短说明"
}`
}

func textUserPrompt(input string, style string, ratio string, target string) string {
	return fmt.Sprintf(`用户想法：%s
目标模型：%s
期望风格：%s
期望比例：%s

请生成专业图片提示词。`, input, valueOr(target, "gpt-image-2"), valueOr(style, "自动判断"), valueOr(ratio, "自动"))
}

func imageSystemPrompt() string {
	return `你是本机生图工作台内置的图片还原提示词分析师。你会观察用户给出的图片，并生成可用于 gpt-image-2 复刻画面氛围的提示词。

要求：
1. 输出必须是一个 JSON 对象，不要 Markdown，不要代码块。
2. 只描述图中可见内容，不能编造看不见的细节。
3. 必须识别并输出 ratio。ratio 只能取 auto、1:1、2:3、3:2、3:4、4:3、9:16、16:9 中最接近源图画幅的一个；如果用户提示里给了源图尺寸，优先按尺寸判断。
4. 重点分析主体、动作/姿态、构图与景别、镜头视角/焦段感、画风或媒介、环境、光线方向、色彩与色调、材质纹理、景深/清晰度、氛围、画面中的文字/标识/边框等。
5. flatPrompt 控制在一段中文内，适合直接填入生图输入框；必须显式写出画面比例，例如“画面比例为 9:16”。
6. jsonDescription 保留结构化观察信息，字段尽量具体。
7. negativePrompt 写会破坏还原效果的反向词。
8. mustKeep 写 6-8 个必须保留的视觉锚点，avoid 写 3-5 个需要避免的偏差。
9. 不进行真人身份识别，不猜测真实姓名、隐私或敏感身份。

JSON Schema：
{
  "ratio": "1:1 | 2:3 | 3:2 | 3:4 | 4:3 | 9:16 | 16:9 | auto",
  "jsonDescription": {
    "subject": "主体与动作",
    "composition": "构图与景别",
    "camera": "镜头视角、焦段感、透视",
    "style": "摄影/绘画/渲染/设计风格",
    "lighting": "光源方向、软硬、明暗关系",
    "background": "背景",
    "colors": "主色、辅色、色调",
    "materials": "材质与纹理",
    "depth": "景深、清晰度、层次",
    "mood": "氛围",
    "textOrGraphics": "可见文字、图形、标识或边框；没有则写无"
  },
  "flatPrompt": "一段可直接复刻画面感觉的中文提示词",
  "negativePrompt": "负面提示词",
  "mustKeep": ["关键视觉锚点"],
  "avoid": ["需要避免的偏差"]
}`
}

func imageUserPrompt(target string, metrics ImageMetrics, metadata map[string]any) string {
	return fmt.Sprintf(`请分析这张图片，并生成适用于 %s 的图片还原提示词。

源图尺寸参考：%s

图片 metadata / 原始提示词参考：
%s

请优先把源图画幅换算成最接近的受支持比例，写入 ratio 字段，并在 flatPrompt 末尾明确写出“画面比例为 ratio”。如果 metadata 里有原始提示词，只把它当作辅助证据；最终仍以图片可见内容为准。`, valueOr(target, "gpt-image-2"), metricsPrompt(metrics), metadataPrompt(metadata))
}

func refineSystemPrompt() string {
	return `你是本机生图工作台内置的提示词协作修改助手。用户已经有一个图片生成提示词，现在会用自然语言提出修改要求，你需要在保留可用细节的前提下输出一个新的、更符合要求的提示词版本。

要求：
1. 输出必须是一个 JSON 对象，不要 Markdown，不要代码块。
2. flatPrompt 必须是一段可直接生图的中文提示词。
3. 严格执行用户本轮修改要求；没有要求删除的关键视觉锚点尽量保留。
4. 如果用户要求“更简洁”，应减少元素但保留主体、构图、光影、氛围。
5. 如果用户要求“更写实/更电影感/换风格/换主体/换比例”，要体现在 flatPrompt 中。
6. 不要输出违法、隐私、真人身份识别或裸露色情内容；遇到敏感内容时转成安全艺术化表达。

JSON Schema：
{
  "flatPrompt": "修改后的可直接生图提示词",
  "negativePrompt": "负面提示词",
  "ratio": "比例；如果没有变化则沿用当前比例",
  "mustKeep": ["必须保留的关键元素"],
  "avoid": ["需要避免的偏差"],
  "notes": "这次主要改了什么"
}`
}

func refineUserPrompt(session PromptSession, current PromptVersion, message string, provider string, model string) string {
	recent := ""
	if len(session.Messages) > 0 {
		start := len(session.Messages) - 8
		if start < 0 {
			start = 0
		}
		for _, item := range session.Messages[start:] {
			recent += fmt.Sprintf("%s：%s\n", item.Role, item.Content)
		}
	}
	return fmt.Sprintf(`会话标题：%s
目标模型：%s / %s
当前提示词：
%s

当前负面提示词：
%s

当前比例：%s

当前必须保留元素：%v

最近对话：
%s

用户本轮修改要求：
%s

请返回新的提示词版本。`, valueOr(session.Title, "提示词会话"), valueOr(provider, session.Provider), valueOr(model, session.Model), current.Prompt, valueOr(current.NegativePrompt, "无"), valueOr(current.Ratio, valueOr(session.Ratio, "auto")), current.MustKeep, recent, message)
}

func inspirationSystemPrompt() string {
	return `你是本机生图工作台内置的视觉灵感策划。用户没有明确想法时，你需要给出多个适合图片生成的创意方向。

要求：
1. 输出必须是一个 JSON 对象，不要 Markdown，不要代码块。
2. ideas 是数组，每个元素包含 title、summary、tags。
3. title 要短、好懂、有画面记忆点。
4. summary 用一句中文描述画面，便于后续扩写成生图提示词。
5. tags 给 3-5 个中文标签。
6. 创意要安全、可执行、适合图片生成，不要依赖现实人物身份。 

JSON Schema：
{
  "ideas": [
    {
      "title": "雨夜便利店",
      "summary": "一个人在深夜便利店窗边吃关东煮，外面下着雨，玻璃反射霓虹灯。",
      "tags": ["孤独", "日系胶片", "夜景"]
    }
  ]
}`
}

func inspirationIdeasUserPrompt(req InspirationIdeasRequest, count int) string {
	return fmt.Sprintf(`请生成 %d 个图片灵感。

类别：%s
情绪：%s
风格：%s
目标模型：%s
用户补充：%s`, count, valueOr(req.Category, "随机"), valueOr(req.Mood, "随机"), valueOr(req.Style, "随机"), valueOr(req.Target, "通用图片模型"), valueOr(req.Seed, "无"))
}

func inspirationExpandUserPrompt(req InspirationExpandRequest) string {
	idea := req.Idea
	return fmt.Sprintf(`请把下面这个灵感扩写成专业图片生成提示词。

标题：%s
摘要：%s
标签：%v
类别：%s
情绪：%s
风格：%s
目标比例：%s
目标模型：%s / %s

请生成专业图片提示词。`, valueOr(idea.Title, "未命名灵感"), valueOr(idea.Summary, "无"), idea.Tags, valueOr(idea.Category, "自动"), valueOr(idea.Mood, "自动"), valueOr(idea.Style, "自动"), valueOr(req.Ratio, "自动"), valueOr(req.Provider, "通用"), valueOr(req.Model, req.Target))
}

func valueOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
