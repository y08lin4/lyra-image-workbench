package statusmeta

type Entry struct {
	Code        string `json:"code"`
	English     string `json:"english"`
	Chinese     string `json:"chinese"`
	Description string `json:"description"`
}

type Metadata struct {
	Statuses []Entry `json:"statuses"`
	Stages   []Entry `json:"stages"`
	Events   []Entry `json:"events"`
}

func All() Metadata {
	return Metadata{
		Statuses: []Entry{
			{Code: "J100", English: "queued", Chinese: "排队中", Description: "任务已创建，等待后端 worker 执行"},
			{Code: "J200", English: "running", Chinese: "运行中", Description: "后端正在执行任务，前端断开不会取消"},
			{Code: "J300", English: "succeeded", Chinese: "已成功", Description: "所有图片都已生成并保存到本机"},
			{Code: "J206", English: "partial_failed", Chinese: "部分成功", Description: "至少一张成功，至少一张失败"},
			{Code: "J500", English: "failed", Chinese: "已失败", Description: "没有图片成功落盘"},
			{Code: "J499", English: "cancelled", Chinese: "已取消", Description: "用户取消任务；运行中的上游请求只能尽力取消"},
			{Code: "J520", English: "interrupted", Chinese: "已中断", Description: "程序停止或重启导致任务结果无法确认"},
		},
		Stages: []Entry{
			{Code: "S100", English: "queued", Chinese: "排队中", Description: "等待进入执行队列"},
			{Code: "S110", English: "preparing", Chinese: "准备中", Description: "校验参数、读取配置和准备参考图"},
			{Code: "S120", English: "submitting", Chinese: "提交中", Description: "准备向内网 NewAPI 发起请求"},
			{Code: "S130", English: "waiting_upstream", Chinese: "等待上游", Description: "NewAPI 正在生成图片，默认 600 秒超时"},
			{Code: "S140", English: "downloading", Chinese: "下载图片", Description: "上游返回 URL 时由后端下载图片"},
			{Code: "S150", English: "saving", Chinese: "保存本机", Description: "图片正在写入 outputs 目录"},
			{Code: "S300", English: "succeeded", Chinese: "已成功", Description: "当前阶段已完成"},
			{Code: "S206", English: "partial_failed", Chinese: "部分成功", Description: "任务部分图片失败"},
			{Code: "S500", English: "failed", Chinese: "已失败", Description: "任务执行失败"},
			{Code: "S499", English: "cancelled", Chinese: "已取消", Description: "任务已取消"},
			{Code: "S520", English: "interrupted", Chinese: "已中断", Description: "任务执行被程序停止打断"},
		},
		Events: []Entry{
			{Code: "E100", English: "snapshot", Chinese: "任务快照", Description: "SSE 连接建立时返回完整当前状态"},
			{Code: "E110", English: "progress", Chinese: "进度更新", Description: "任务状态、阶段、百分比和中文文案更新"},
			{Code: "E120", English: "result", Chinese: "单图结果", Description: "某一张图片成功或失败"},
			{Code: "E130", English: "heartbeat", Chinese: "心跳保活", Description: "证明观察连接仍然存在，不代表上游完成"},
			{Code: "E300", English: "done", Chinese: "任务结束", Description: "任务进入 succeeded、partial_failed、failed、cancelled 或 interrupted"},
			{Code: "E500", English: "error", Chinese: "错误事件", Description: "任务或事件流出现错误，需同时显示中英文和状态码"},
		},
	}
}
