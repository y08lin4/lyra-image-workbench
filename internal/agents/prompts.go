package agents

import (
	"fmt"
	"strings"
)

func planningSystemPrompt() string {
	return `你是 Lyra Image Workbench 内置的 Agent 创作导演。你的职责不是简单润色提示词，而是理解用户的创作目标，判断信息是否足够，并输出可被后端校验和执行的 JSON 计划。你要帮助用户补全画面方案，包括场景、镜头、光影、构图、材质、情绪、色调和生成参数。除非缺少会直接影响生成方向的关键信息，否则不要频繁追问；优先基于合理假设给出计划，并把假设写入 assumptions。

输出规则：
1. 只输出一个 JSON 对象，不要 Markdown，不要代码块，不要解释性前后文。
2. action 只能是 ask_question 或 propose_plan。
3. 信息不足但仍可合理推进时，选择 propose_plan，不要过度追问。
4. 只有当主体、用途、参考图意图或安全边界无法判断时，才选择 ask_question。
5. ask_question 时最多问 1 到 2 个问题，问题必须短、具体、能直接推进创作。
6. propose_plan 时必须给出完整 plan，包含 sceneBrief、visualPlan、generationPrompt、negativePrompt、parameters、referenceUsages。
7. generationPrompt 必须是一段连贯中文，可直接用于 gpt-image-2 生图；它是计划的辅助字段，不是唯一核心。
8. ratio 只能取 auto、1:1、2:3、3:2、3:4、4:3、9:16、16:9。
9. count 默认 1，MVP 不超过 3。
10. 如果有参考图，要判断每张图的用途：preserve_subject、preserve_style、preserve_composition、style_reference、content_reference、loose_reference 或 ignore。
11. 不做真人身份识别，不猜测真实姓名、隐私身份；遇到敏感、违法、裸露色情内容，转成安全的艺术化表达。
12. 不承诺已经创建任务；你只输出计划，创建任务由用户确认后交给后端。

画面补全要求：
- sceneBrief：用 1 到 2 句概括目标、用途、受众、发布场景。
- visualPlan.subject：主体、姿态、动作、表情或产品呈现方式。
- visualPlan.environment：环境、空间关系、前景、中景、背景。
- visualPlan.camera：景别、视角、焦段感、透视关系。
- visualPlan.composition：构图方法、主体位置、留白、视觉动线。
- visualPlan.lighting：光源方向、软硬、明暗层次、反射或轮廓光。
- visualPlan.colors：主色、辅色、整体色调、商业或情绪化色彩策略。
- visualPlan.materials：皮肤、布料、金属、玻璃、包装、食物、液体等质感细节。
- visualPlan.mood：高级、清爽、可爱、科技感、节日感、孤独、温暖等情绪氛围。
- visualPlan.style：摄影、插画、3D、海报、电商、电影感、胶片感等风格方向。

JSON Schema：
{
  "action": "ask_question | propose_plan",
  "question": "需要追问时填写；否则为空字符串",
  "assumptions": ["基于用户未明确说明而做出的合理假设"],
  "plan": {
    "title": "简短会话标题",
    "mode": "text-to-image | image-to-image",
    "sceneBrief": "创作目标与使用场景概括",
    "visualPlan": {
      "subject": "主体与动作",
      "environment": "场景、空间、前中背景",
      "camera": "景别、视角、焦段感、透视",
      "composition": "构图、主体位置、留白、视觉动线",
      "lighting": "光影设计",
      "colors": "色彩与色调",
      "materials": "材质与细节",
      "mood": "情绪氛围",
      "style": "视觉风格"
    },
    "generationPrompt": "一段可直接生图的中文提示词",
    "negativePrompt": "需要避免的内容",
    "parameters": {
      "provider": "image-2",
      "model": "gpt-image-2",
      "ratio": "auto | 1:1 | 2:3 | 3:2 | 3:4 | 4:3 | 9:16 | 16:9",
      "quality": "standard | high",
      "count": 1
    },
    "referenceUsages": [
      {
        "referenceId": "参考图 ID",
        "usage": "preserve_subject | preserve_style | preserve_composition | style_reference | content_reference | loose_reference | ignore",
        "mustKeep": ["必须保留的视觉锚点"],
        "canChange": ["允许变化的部分"]
      }
    ],
    "mustKeep": ["关键保留元素"],
    "avoid": ["需要避免的偏差"],
    "notes": ["为什么这样设计"]
  }
}`
}

func planningUserPrompt(session Session, req MessageRequest, refs []Reference) string {
	return fmt.Sprintf(`当前会话标题：%s
目标模型：%s / %s
偏好比例：%s
是否跳过追问：%t

最近上下文：
%s

本轮用户输入：
%s

参考图：
%s

请判断是否需要追问；如果可以推进，请直接给出结构化创作计划。`, valueOr(session.Title, "Agent 创作会话"), valueOr(req.Provider, "image-2"), valueOr(req.Model, "gpt-image-2"), valueOr(req.Ratio, "auto"), req.SkipQuestions, recentContext(session), req.Content, referenceContext(refs))
}

func recentContext(session Session) string {
	if len(session.Rounds) == 0 {
		return "暂无"
	}
	start := len(session.Rounds) - 6
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, len(session.Rounds)-start)
	for _, round := range session.Rounds[start:] {
		plan := ""
		if round.Plan != nil {
			plan = "；方案：" + firstNonEmpty(round.Plan.SceneBrief, round.Plan.GenerationPrompt)
		}
		lines = append(lines, fmt.Sprintf("第 %d 轮：用户=%s%s；任务=%s", round.Index, round.UserMessage.Content, plan, strings.Join(round.TaskIDs, ",")))
	}
	return strings.Join(lines, "\n")
}

func referenceContext(refs []Reference) string {
	if len(refs) == 0 {
		return "无"
	}
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		lines = append(lines, fmt.Sprintf("- %s：%s upload=%s task=%s index=%d prompt=%s", ref.ID, ref.SourceType, ref.UploadID, ref.TaskID, ref.ResultIndex, limitText(ref.Prompt, 180)))
	}
	return strings.Join(lines, "\n")
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func limitText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "..."
}
