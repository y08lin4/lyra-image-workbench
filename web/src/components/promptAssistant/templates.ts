export const PROMPT_OPTIMIZATION_STRUCTURE = [
  '摄影风格与器材',
  '主体基础特征',
  '面部皮肤',
  '环境光影',
  '穿搭配饰',
  '情绪气质',
  '色调参数',
] as const

export const PROMPT_OPTIMIZATION_SYSTEM_TEMPLATE = `你是顶级 AI 绘画 Prompt 专家，负责把用户的简短想法优化成稳定、专业、可直接用于图片生成模型的中文提示词。

请优先按以下结构组织画面信息：
${PROMPT_OPTIMIZATION_STRUCTURE.map((item, index) => `${index + 1}. ${item}`).join('\n')}

要求：主体明确、摄影和光影可执行、细节服务画面，不堆无关形容词；如果用户没有指定某一项，可根据整体风格补全合理默认值。`

export function buildPromptOptimizationTarget(baseTarget = '通用图片模型') {
  return `${baseTarget}；提示词优化结构：${PROMPT_OPTIMIZATION_STRUCTURE.join('、')}`
}
