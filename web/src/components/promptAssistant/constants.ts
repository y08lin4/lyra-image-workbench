export type Tab = 'text' | 'image' | 'inspiration' | 'history'
export type InspirationStepId = 'idea' | 'purpose' | 'style' | 'palette' | 'composition' | 'constraints'
export type InspirationSkipState = Record<InspirationStepId, boolean>

export type InspirationStep = {
  id: InspirationStepId
  label: string
  question: string
  placeholder: string
  quick: string[]
}

export const styleOptions = [
  { value: 'auto', label: '自动判断' },
  { value: 'cinematic', label: '电影感' },
  { value: 'photo', label: '写实摄影' },
  { value: 'poster', label: '海报设计' },
  { value: 'anime', label: '二次元' },
  { value: 'product', label: '产品图' },
]

export const ratioOptions = ['auto', '1:1', '2:3', '3:2', '3:4', '4:3', '9:16', '16:9']
export const categoryOptions = ['随机', '人像', '场景', '产品', '海报', '插画', '壁纸', '建筑', '美食']
export const moodOptions = ['随机', '治愈', '孤独', '高级', '梦幻', '压迫感', '温暖', '荒诞', '浪漫']
export const inspirationStyleOptions = ['随机', '写实摄影', '电影感', '日系胶片', '二次元', '3D 渲染', '极简设计', '国风']
export const quickRefines = ['更写实', '更电影感', '更简洁', '更高级', '更梦幻', '增强光影', '减少元素', '改成竖屏构图', '改成商业海报']

export const promptTabs: Array<{ id: Tab; label: string }> = [
  { id: 'text', label: '提示词优化' },
  { id: 'inspiration', label: '灵感模式' },
  { id: 'image', label: '图片还原' },
  { id: 'history', label: '历史' },
]

export const inspirationSteps: InspirationStep[] = [
  {
    id: 'idea',
    label: '一句话想法',
    question: '先随便说一个画面想法，不用完整。',
    placeholder: '例如：雨夜东京街头，一个穿白色风衣的人回头看镜头',
    quick: ['孤独城市夜景', '高级产品海报', '治愈系房间', '复古电影人像'],
  },
  {
    id: 'purpose',
    label: '用途',
    question: '这张图准备用在哪里？用途会影响构图、留白和细节密度。',
    placeholder: '例如：手机壁纸 / 电商主图 / 社媒海报 / 角色设定',
    quick: ['手机壁纸', '社媒封面', '电商主图', '品牌海报', '角色设定'],
  },
  {
    id: 'style',
    label: '风格',
    question: '希望整体像哪种视觉风格？可以给一个类型或参考感受。',
    placeholder: '例如：写实摄影、电影感、日系胶片、极简商业海报',
    quick: ['写实摄影', '电影感', '日系胶片', '极简设计', '3D 渲染'],
  },
  {
    id: 'palette',
    label: '色调',
    question: '色调想偏冷、偏暖、低饱和，还是更鲜明？',
    placeholder: '例如：低饱和冷色，霓虹蓝紫，局部暖光',
    quick: ['低饱和冷色', '暖色柔光', '黑白高反差', '蓝紫霓虹', '品牌留白'],
  },
  {
    id: 'composition',
    label: '构图',
    question: '主体怎么摆？近景、远景、居中、留白、俯拍都可以。',
    placeholder: '例如：主体居中，低机位仰拍，背景留出标题区',
    quick: ['主体居中', '近景特写', '广角远景', '对称构图', '右侧留白'],
  },
  {
    id: 'constraints',
    label: '参考限制',
    question: '最后补充必须保留或避免的内容。',
    placeholder: '例如：不要文字，不要水印，保留红色雨伞和湿润路面',
    quick: ['不要文字', '不要水印', '避免真人脸', '保留品牌色', '无特别限制'],
  },
]

export const emptyInspirationAnswers: Record<InspirationStepId, string> = {
  idea: '',
  purpose: '',
  style: '',
  palette: '',
  composition: '',
  constraints: '',
}

export const emptyInspirationSkipped: InspirationSkipState = {
  idea: false,
  purpose: false,
  style: false,
  palette: false,
  composition: false,
  constraints: false,
}

export const observationLabels: Record<string, string> = {
  subject: '主体',
  composition: '构图',
  camera: '镜头',
  style: '风格',
  lighting: '光线',
  background: '背景',
  colors: '色彩',
  materials: '材质',
  depth: '景深/层次',
  mood: '氛围',
  textOrGraphics: '文字/图形',
  ratio: '比例',
  sourceSize: '源图尺寸',
  orientation: '画幅方向',
  metadataPrompt: '原提示词',
  metadataNegativePrompt: '原负面词',
  parameters: '生成参数',
  prompt: 'Prompt',
  negativePrompt: 'Negative Prompt',
  workflow: 'Workflow',
  software: '软件',
  comment: '注释',
  userComment: '用户注释',
  imageDescription: '图片描述',
  xmp: 'XMP',
  exif: 'EXIF',
}

export function kindLabel(kind?: string) {
  if (kind === 'image') return '图片还原'
  if (kind === 'inspiration') return '灵感扩写'
  if (kind === 'manual') return '手动会话'
  return '提示词优化'
}
