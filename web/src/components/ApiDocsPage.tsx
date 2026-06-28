import { useState } from 'react'
import './ApiDocsPage.css'

const REGISTER_URL = 'https://ai-image.ailinyu.de/'
const DOCS_REPO_URL = 'https://github.com/y08lin4/LyAi-Image-Generation-API-Documentation'
const API_BASE_URL = 'https://ai-image.ailinyu.de'
const API_KEY_PLACEHOLDER = '<API_KEY>'
const ENV_VAR_NAME = 'LYRA_API_KEY'
const DEFAULT_MODEL = 'gpt-image-2'
const WAIT_STATUSES = ['queued', 'running']
const SUCCESS_STATUSES = ['succeeded', 'partial_failed']
const FAILURE_STATUSES = ['failed', 'cancelled', 'interrupted']
const TERMINAL_STATUSES = [...SUCCESS_STATUSES, ...FAILURE_STATUSES]

type ApiExample = {
  id: string
  language: string
  fileName: string
  code: string
}

type CodeCardProps = {
  examples: ApiExample[]
  activeId: string
  title: string
  description: string
  copyLabel: string
  onActiveIdChange: (id: string) => void
  onCopy: (label: string, value: string) => void
}

type CodeBlockFrameProps = {
  title: string
  fileName?: string
  code: string
  copyLabel: string
  wrap?: boolean
  onCopy: (label: string, value: string) => void
}

const REQUEST_PAYLOAD = {
  model: DEFAULT_MODEL,
  prompt: 'A clean product photo of a translucent smart speaker on a stone pedestal',
  size: '1024x1024',
  quality: 'auto',
  output_format: 'png',
  n: 1,
}

const REQUEST_BODY_PRETTY = JSON.stringify(REQUEST_PAYLOAD, null, 2)
const REQUEST_BODY_COMPACT = JSON.stringify(REQUEST_PAYLOAD)
const JAVA_REQUEST_BODY = REQUEST_BODY_PRETTY.split('\n').map((line) => `      ${line}`).join('\n')

const QUICK_STEPS = [
  ['准备账号', '注册登录 LyAi，在设置页保存 image-2 或 banana 的云端上游 Key。'],
  ['生成 Bearer Key', `复制 lyra_sk_...，服务端脚本从环境变量 ${ENV_VAR_NAME} 读取；不要写进前端代码。`],
  ['创建、轮询、下载', `POST 创建任务，轮询到 ${TERMINAL_STATUSES.join('/')}，成功或部分成功时下载第一个 ok=true 结果。`],
] as const

const ENDPOINTS = [
  ['POST', '/v1/images/generations', 'OpenAI 兼容创建端点，推荐 SDK 默认使用。'],
  ['POST', '/v1/image-tasks', 'Lyra 原生创建端点，适合需要完整任务字段的接入。'],
  ['GET', '/v1/image-tasks/{taskId}', '轮询状态、进度、results 和错误信息。'],
  ['GET', '/v1/image-tasks/{taskId}/images/{index}', '下载 results 中 ok=true 的图片二进制。'],
  ['POST', '/v1/image-tasks/{taskId}/cancel', '取消排队任务；运行中任务为尽力取消。'],
] as const

const ERROR_CODES = [
  '400 BAD_JSON: 请求体不是有效 JSON。',
  '400 UPSTREAM_KEY_REQUIRED: 用户没有保存对应 provider 的云端上游 Key。',
  '400 TASK_CREATE_FAILED: 参数错误、非 text-to-image 或上游创建失败。',
  '401 UNAUTHORIZED: Bearer 缺失、格式错误或无效。',
  '402 USER_CREDITS_NOT_ENOUGH: 账号生成次数不足。',
  '404 TASK_NOT_FOUND / TASK_IMAGE_NOT_FOUND: 任务或图片不存在，或不属于当前 Bearer Key。',
  '429 AUTH_RATE_LIMITED: 无效 Bearer 尝试过多，请稍后重试。',
  '500 INTERNAL_ERROR: 服务端内部错误。',
]

const API_EXAMPLES: ApiExample[] = [
  {
    id: 'curl',
    language: 'cURL',
    fileName: 'terminal',
    code: [
      'export LYRA_API_KEY="lyra_sk_xxx"',
      `BASE_URL="${API_BASE_URL}"`,
      '',
      'CREATE_JSON=$(curl --fail-with-body -sS -X POST "$BASE_URL/v1/images/generations" \\',
      '  -H "Authorization: Bearer $LYRA_API_KEY" \\',
      '  -H "Content-Type: application/json" \\',
      `  -d '${REQUEST_BODY_COMPACT}')`,
      'TASK_ID=$(python -c "import json,sys; d=json.load(sys.stdin); print(d.get(\'taskId\') or d[\'task\'][\'id\'])" <<< "$CREATE_JSON")',
      'echo "task_id=$TASK_ID"',
      '',
      'while true; do',
      '  TASK_JSON=$(curl --fail-with-body -sS "$BASE_URL/v1/image-tasks/$TASK_ID" -H "Authorization: Bearer $LYRA_API_KEY")',
      '  STATUS=$(python -c "import json,sys; print(json.load(sys.stdin)[\'task\'][\'status\'])" <<< "$TASK_JSON")',
      '  echo "status=$STATUS"',
      '  [[ " succeeded partial_failed failed cancelled interrupted " == *" $STATUS "* ]] && break',
      '  sleep 3',
      'done',
      '',
      '[[ " failed cancelled interrupted " == *" $STATUS "* ]] && { echo "$TASK_JSON"; exit 1; }',
      'OK_INDEX=$(python -c "import json,sys; t=json.load(sys.stdin)[\'task\']; print(next((r.get(\'index\', i) for i,r in enumerate(t.get(\'results\', [])) if r.get(\'ok\')), \'\'))" <<< "$TASK_JSON")',
      'test -n "$OK_INDEX" || { echo "no ok=true result"; exit 1; }',
      'DOWNLOAD_URL="$BASE_URL/v1/image-tasks/$TASK_ID/images/$OK_INDEX"',
      'echo "result_url=$DOWNLOAD_URL"',
      'curl --fail-with-body -L "$DOWNLOAD_URL" -H "Authorization: Bearer $LYRA_API_KEY" --output "result-$OK_INDEX.png"',
    ].join('\n'),
  },
  {
    id: 'typescript',
    language: 'TypeScript',
    fileName: 'generate-image.ts',
    code: [
      'import { writeFile } from "node:fs/promises";',
      '',
      `const BASE_URL = "${API_BASE_URL}";`,
      'const API_KEY = process.env.LYRA_API_KEY;',
      'const TERMINAL = new Set(["succeeded", "partial_failed", "failed", "cancelled", "interrupted"]);',
      'const FAILURE = new Set(["failed", "cancelled", "interrupted"]);',
      'if (!API_KEY) throw new Error("Set LYRA_API_KEY before running this script.");',
      '',
      'async function api(path: string, init: RequestInit = {}) {',
      '  const response = await fetch(BASE_URL + path, {',
      '    ...init,',
      '    headers: {',
      '      Authorization: "Bearer " + API_KEY,',
      '      ...(init.body ? { "Content-Type": "application/json" } : {}),',
      '      ...init.headers,',
      '    },',
      '  });',
      '  if (!response.ok) throw new Error(`${response.status} ${response.statusText}: ${await response.text()}`);',
      '  return response;',
      '}',
      '',
      'const created = await api("/v1/images/generations", {',
      '  method: "POST",',
      `  body: JSON.stringify(${REQUEST_BODY_PRETTY}),`,
      '}).then((res) => res.json());',
      'const taskId = created.taskId ?? created.task.id;',
      '',
      'let task = created.task;',
      'while (true) {',
      '  task = await api("/v1/image-tasks/" + taskId).then((res) => res.json()).then((data) => data.task);',
      '  console.log(task.id, task.status, task.progress ?? 0);',
      '  if (TERMINAL.has(task.status)) break;',
      '  await new Promise((resolve) => setTimeout(resolve, 3000));',
      '}',
      '',
      'if (FAILURE.has(task.status)) throw new Error("task failed: " + task.status);',
      'const result = task.results?.find((item: { ok?: boolean }) => item.ok);',
      'if (!result) throw new Error("task finished without ok=true result");',
      'const resultUrl = `${BASE_URL}/v1/image-tasks/${taskId}/images/${result.index}`;',
      'console.log("result_url", resultUrl);',
      'const image = await api("/v1/image-tasks/" + taskId + "/images/" + result.index).then((res) => res.arrayBuffer());',
      'await writeFile("result-" + result.index + ".png", Buffer.from(image));',
    ].join('\n'),
  },
  {
    id: 'python',
    language: 'Python',
    fileName: 'generate_image.py',
    code: [
      'import os',
      'import time',
      'import requests',
      '',
      `BASE_URL = "${API_BASE_URL}"`,
      'TERMINAL = {"succeeded", "partial_failed", "failed", "cancelled", "interrupted"}',
      'FAILURE = {"failed", "cancelled", "interrupted"}',
      'api_key = os.environ.get("LYRA_API_KEY")',
      'if not api_key:',
      '    raise SystemExit("Set LYRA_API_KEY before running this script.")',
      'headers = {"Authorization": f"Bearer {api_key}"}',
      '',
      'created = requests.post(f"{BASE_URL}/v1/images/generations", headers={**headers, "Content-Type": "application/json"}, json=' + JSON.stringify(REQUEST_PAYLOAD) + ', timeout=120)',
      'created.raise_for_status()',
      'payload = created.json()',
      'task_id = payload.get("taskId") or payload["task"]["id"]',
      '',
      'while True:',
      '    snapshot = requests.get(f"{BASE_URL}/v1/image-tasks/{task_id}", headers=headers, timeout=30)',
      '    snapshot.raise_for_status()',
      '    task = snapshot.json()["task"]',
      '    print(task["id"], task["status"], task.get("progress", 0))',
      '    if task["status"] in TERMINAL:',
      '        break',
      '    time.sleep(3)',
      '',
      'if task["status"] in FAILURE:',
      '    raise SystemExit(f"task failed: {task[\'status\']}")',
      'result = next((item for item in task.get("results", []) if item.get("ok")), None)',
      'if not result:',
      '    raise SystemExit("task finished without ok=true result")',
      'result_url = f"{BASE_URL}/v1/image-tasks/{task_id}/images/{result[\'index\']}"',
      'print("result_url", result_url)',
      'image = requests.get(result_url, headers=headers, timeout=120)',
      'image.raise_for_status()',
      'open(f"result-{result[\'index\']}.png", "wb").write(image.content)',
    ].join('\n'),
  },
  {
    id: 'go',
    language: 'Go',
    fileName: 'main.go',
    code: [
      'package main',
      '',
      'import (',
      '  "bytes"',
      '  "encoding/json"',
      '  "fmt"',
      '  "io"',
      '  "net/http"',
      '  "os"',
      '  "time"',
      ')',
      '',
      `const baseURL = "${API_BASE_URL}"`,
      '',
      'func request(method string, path string, body []byte) []byte {',
      '  apiKey := os.Getenv("LYRA_API_KEY")',
      '  if apiKey == "" { panic("Set LYRA_API_KEY before running this script.") }',
      '  req, _ := http.NewRequest(method, baseURL+path, bytes.NewReader(body))',
      '  req.Header.Set("Authorization", "Bearer "+apiKey)',
      '  if body != nil { req.Header.Set("Content-Type", "application/json") }',
      '  resp, err := http.DefaultClient.Do(req); if err != nil { panic(err) }',
      '  defer resp.Body.Close()',
      '  data, _ := io.ReadAll(resp.Body)',
      '  if resp.StatusCode >= 400 { panic(string(data)) }',
      '  return data',
      '}',
      '',
      'func main() {',
      `  createdBytes := request("POST", "/v1/images/generations", []byte(\`${REQUEST_BODY_COMPACT}\`))`,
      '  var created map[string]any; json.Unmarshal(createdBytes, &created)',
      '  taskID, _ := created["taskId"].(string)',
      '  if taskID == "" { taskID, _ = created["task"].(map[string]any)["id"].(string) }',
      '  terminal := map[string]bool{"succeeded": true, "partial_failed": true, "failed": true, "cancelled": true, "interrupted": true}',
      '  failure := map[string]bool{"failed": true, "cancelled": true, "interrupted": true}',
      '',
      '  var task map[string]any; status := ""',
      '  for {',
      '    var snapshot map[string]any',
      '    json.Unmarshal(request("GET", "/v1/image-tasks/"+taskID, nil), &snapshot)',
      '    task = snapshot["task"].(map[string]any); status = task["status"].(string)',
      '    fmt.Println(taskID, status, task["progress"])',
      '    if terminal[status] { break }',
      '    time.Sleep(3 * time.Second)',
      '  }',
      '  if failure[status] { panic("task failed: " + status) }',
      '  index := firstOKIndex(task); if index < 0 { panic("task finished without ok=true result") }',
      '  resultPath := fmt.Sprintf("/v1/image-tasks/%s/images/%d", taskID, index)',
      '  fmt.Println("result_url=" + baseURL + resultPath)',
      '  image := request("GET", resultPath, nil)',
      '  os.WriteFile(fmt.Sprintf("result-%d.png", index), image, 0644)',
      '}',
      '',
      'func firstOKIndex(task map[string]any) int {',
      '  results, _ := task["results"].([]any)',
      '  for fallback, item := range results {',
      '    result, _ := item.(map[string]any)',
      '    if result["ok"] == true { if index, ok := result["index"].(float64); ok { return int(index) }; return fallback }',
      '  }',
      '  return -1',
      '}',
    ].join('\n'),
  },
  {
    id: 'java',
    language: 'Java',
    fileName: 'GenerateImage.java',
    code: [
      'import java.net.URI;',
      'import java.net.http.*;',
      'import java.nio.file.*;',
      'import java.time.Duration;',
      'import java.util.Set;',
      '',
      'public class GenerateImage {',
      `  static final String BASE_URL = "${API_BASE_URL}";`,
      '  static final HttpClient HTTP = HttpClient.newHttpClient();',
      '  static final Set<String> TERMINAL = Set.of("succeeded", "partial_failed", "failed", "cancelled", "interrupted");',
      '  static final Set<String> FAILURE = Set.of("failed", "cancelled", "interrupted");',
      '',
      '  public static void main(String[] args) throws Exception {',
      '    String body = """',
      JAVA_REQUEST_BODY,
      '      """;',
      '    String created = send("POST", "/v1/images/generations", body);',
      '    String taskId = field(created, "taskId");',
      '    if (taskId.isBlank()) taskId = field(created, "id");',
      '    String snapshot; String status;',
      '    while (true) {',
      '      snapshot = send("GET", "/v1/image-tasks/" + taskId, null);',
      '      status = field(snapshot, "status");',
      '      System.out.println(taskId + " " + status);',
      '      if (TERMINAL.contains(status)) {',
      '        if (FAILURE.contains(status)) throw new RuntimeException("task failed: " + status);',
      '        int index = firstOkIndex(snapshot);',
      '        if (index < 0) throw new RuntimeException("task finished without ok=true result");',
      '        String resultPath = "/v1/image-tasks/" + taskId + "/images/" + index;',
      '        System.out.println("result_url=" + BASE_URL + resultPath);',
      '        byte[] image = sendBytes(resultPath);',
      '        Files.write(Path.of("result-" + index + ".png"), image);',
      '        break;',
      '      }',
      '      Thread.sleep(3000);',
      '    }',
      '  }',
      '',
      '  static String send(String method, String path, String body) throws Exception {',
      '    String apiKey = System.getenv("LYRA_API_KEY");',
      '    if (apiKey == null || apiKey.isBlank()) throw new IllegalStateException("Set LYRA_API_KEY before running this script.");',
      '    HttpRequest.Builder builder = HttpRequest.newBuilder(URI.create(BASE_URL + path)).header("Authorization", "Bearer " + apiKey).timeout(Duration.ofSeconds(120));',
      '    if (body == null) builder.GET(); else builder.header("Content-Type", "application/json").method(method, HttpRequest.BodyPublishers.ofString(body));',
      '    HttpResponse<String> response = HTTP.send(builder.build(), HttpResponse.BodyHandlers.ofString());',
      '    if (response.statusCode() >= 400) throw new RuntimeException(response.body());',
      '    return response.body();',
      '  }',
      '',
      '  static byte[] sendBytes(String path) throws Exception {',
      '    String apiKey = System.getenv("LYRA_API_KEY");',
      '    if (apiKey == null || apiKey.isBlank()) throw new IllegalStateException("Set LYRA_API_KEY before running this script.");',
      '    HttpRequest req = HttpRequest.newBuilder(URI.create(BASE_URL + path)).header("Authorization", "Bearer " + apiKey).GET().build();',
      '    HttpResponse<byte[]> res = HTTP.send(req, HttpResponse.BodyHandlers.ofByteArray());',
      '    if (res.statusCode() >= 400) throw new RuntimeException("download failed: HTTP " + res.statusCode());',
      '    return res.body();',
      '  }',
      '',
      '  static String field(String json, String name) {',
      '    int at = json.indexOf("\\\"" + name + "\\\":");',
      '    int start = at < 0 ? -1 : json.indexOf("\\\"", at + name.length() + 3);',
      '    int end = start < 0 ? -1 : json.indexOf("\\\"", start + 1);',
      '    return end > start ? json.substring(start + 1, end) : "";',
      '  }',
      '',
      '  static int firstOkIndex(String json) {',
      '    int okAt = json.indexOf("\\\"ok\\\":true");',
      '    if (okAt < 0) okAt = json.indexOf("\\\"ok\\\": true");',
      '    if (okAt < 0) return -1;',
      '    int indexAt = json.lastIndexOf("\\\"index\\\"", okAt);',
      '    int colon = indexAt < 0 ? -1 : json.indexOf(":", indexAt);',
      '    int pos = colon + 1; while (pos < json.length() && Character.isWhitespace(json.charAt(pos))) pos++;',
      '    int end = pos; while (end < json.length() && Character.isDigit(json.charAt(end))) end++;',
      '    return end > pos ? Integer.parseInt(json.substring(pos, end)) : 0;',
      '  }',
      '}',
    ].join('\n'),
  },
]

const ORDERED_API_EXAMPLES = ['curl', 'typescript', 'python', 'go', 'java']
  .map((id) => API_EXAMPLES.find((example) => example.id === id))
  .filter((example): example is ApiExample => Boolean(example))

const AI_INTEGRATION_PROMPT = [
  '请根据以下信息接入 LyAi Image Generation API。',
  '',
  `Base URL: ${API_BASE_URL}`,
  `环境变量: ${ENV_VAR_NAME}=lyra_sk_xxx`,
  '认证: 所有 /v1/* 请求都带 Authorization: Bearer <LYRA_API_KEY>。不要把 Bearer Key 写进前端代码。',
  '',
  '创建任务:',
  '- 推荐端点: POST /v1/images/generations',
  '- 备用原生端点: POST /v1/image-tasks',
  '- Header: Content-Type: application/json',
  `- 示例请求体: ${REQUEST_BODY_COMPACT}`,
  '- /v1/images/generations 中 prompt 必填；model 可省略，image-2 默认 gpt-image-2；size 支持 auto、1024x1024、1024x1536、1536x1024、768x1024、1024x768、1008x1792、1792x1008 以及 2K/4K 对应尺寸。',
  '- 原生端点可传 ratio(auto/1:1/2:3/3:2/3:4/4:3/9:16/16:9)、resolution(auto/standard/2k/4k)、quality(auto/low/medium/high)、outputFormat(png/jpeg/webp/auto)、count(1-24)、concurrency(最小 1)。',
  '- 创建响应里读取 taskId；若没有 taskId，则读取 task.id。保存原始响应，便于错误排查。',
  '',
  '轮询任务:',
  '- GET /v1/image-tasks/{taskId}',
  '- 每 2-5 秒轮询一次，建议最长等待 15 分钟。',
  `- ${WAIT_STATUSES.join('/')} 继续轮询。`,
  `- ${SUCCESS_STATUSES.join('/')} 为可下载终态；partial_failed 也要暴露失败 result。`,
  `- ${FAILURE_STATUSES.join('/')} 停止轮询并抛出任务失败错误。`,
  '',
  '下载结果:',
  '- GET /v1/image-tasks/{taskId}/images/{index}',
  '- index 来自 task.results[].index；只下载 ok=true 的结果；响应 body 是图片二进制，按 outputFormat 或 png 写入文件。',
  `- 结果 URL 形如 ${API_BASE_URL}/v1/image-tasks/{taskId}/images/{index}；下载请求同样要带 Authorization Bearer。`,
  '- 如果没有 ok=true 的 result，不要猜测 URL，抛出明确错误并输出 task.results。partial_failed 可以下载成功项，也要把失败项暴露给调用方。',
  '',
  '错误码:',
  ...ERROR_CODES.map((item) => `- ${item}`),
  '',
  '实现要求：读取 LYRA_API_KEY，创建任务，打印 taskId，按上述终态规则轮询，succeeded/partial_failed 时下载第一个 ok=true 结果，failed/cancelled/interrupted 时抛错，并处理 HTTP 错误、超时和 429 AUTH_RATE_LIMITED。',
].join('\n')

export function ApiDocsPage() {
  const [copied, setCopied] = useState('')
  const [copyError, setCopyError] = useState('')
  const [activeLanguage, setActiveLanguage] = useState(ORDERED_API_EXAMPLES[0].id)
  const [apiKeyDraft, setApiKeyDraft] = useState('')
  const [apiKeyVisible, setApiKeyVisible] = useState(false)

  const activeExample = ORDERED_API_EXAMPLES.find((example) => example.id === activeLanguage) || ORDERED_API_EXAMPLES[0]
  const trimmedApiKey = apiKeyDraft.trim()
  const keyToken = trimmedApiKey || API_KEY_PLACEHOLDER
  const authHeader = `Authorization: Bearer ${keyToken}`
  const envLine = `${ENV_VAR_NAME}=${keyToken}`
  const apiKeyPreview = trimmedApiKey ? (apiKeyVisible ? trimmedApiKey : maskApiKey(trimmedApiKey)) : API_KEY_PLACEHOLDER

  async function copyText(label: string, value: string) {
    setCopyError('')
    try {
      await copyToClipboard(value)
      setCopied(label)
      window.setTimeout(() => setCopied((current) => current === label ? '' : current), 1800)
    } catch (err) {
      setCopied('')
      setCopyError(err instanceof Error ? err.message : '复制失败，请手动选择内容复制')
    }
  }

  return (
    <section className="workflow-page api-docs-page api-console-page" aria-labelledby="api-docs-title">
      <header className="workflow-page-header api-console-header">
        <div>
          <p className="eyebrow">API Docs</p>
          <h2 id="api-docs-title">SDK 接入控制台</h2>
          <p>最短路径：Bearer Key、创建任务、轮询状态、下载图片。</p>
        </div>
        <div className="api-console-actions">
          <a href={REGISTER_URL} target="_blank" rel="noreferrer">注册和配置</a>
          <a href={DOCS_REPO_URL} target="_blank" rel="noreferrer">GitHub 参考</a>
          <button type="button" onClick={() => void copyText('接入清单', AI_INTEGRATION_PROMPT)}>复制接入清单</button>
        </div>
      </header>

      <div className="api-console-grid">
        <div className="api-console-left">
          <section className="api-console-panel api-console-steps" aria-labelledby="api-steps-title">
            <div className="api-console-panel-head">
              <h3 id="api-steps-title">3 步接入</h3>
              <span>{API_BASE_URL}</span>
            </div>
            <ol>
              {QUICK_STEPS.map(([title, body]) => (
                <li key={title}>
                  <b>{title}</b>
                  <p>{body}</p>
                </li>
              ))}
            </ol>
          </section>

          <section className="api-console-panel api-console-key-panel" aria-labelledby="api-key-title">
            <div className="api-console-panel-head">
              <h3 id="api-key-title">Bearer Key</h3>
              <span>只在服务端使用</span>
            </div>
            <label className="api-console-key-input" htmlFor="api-key-draft">
              <span>粘贴 Key 后复制 Header 或环境变量</span>
              <div>
                <input
                  id="api-key-draft"
                  type={apiKeyVisible ? 'text' : 'password'}
                  value={apiKeyDraft}
                  onChange={(event) => setApiKeyDraft(event.target.value)}
                  placeholder="lyra_sk_..."
                  spellCheck={false}
                  autoComplete="off"
                />
                <button type="button" disabled={!trimmedApiKey} aria-pressed={apiKeyVisible} onClick={() => setApiKeyVisible((visible) => !visible)}>
                  {apiKeyVisible ? '隐藏' : '显示'}
                </button>
              </div>
            </label>
            <div className="api-console-key-preview" aria-live="polite"><code>{apiKeyPreview}</code></div>
            <div className="api-console-button-row">
              <button type="button" onClick={() => void copyText('Authorization Header', authHeader)}>复制 Header</button>
              <button type="button" onClick={() => void copyText('环境变量', envLine)}>复制环境变量</button>
            </div>
          </section>

          <section className="api-console-panel api-console-reference" aria-labelledby="api-reference-title">
            <div className="api-console-panel-head">
              <h3 id="api-reference-title">端点和错误码</h3>
              <button type="button" onClick={() => void copyText('请求体示例', REQUEST_BODY_PRETTY)}>复制请求体</button>
            </div>
            <div className="api-console-endpoints">
              {ENDPOINTS.map(([method, path, note]) => (
                <div className="api-console-endpoint" key={`${method}-${path}`}>
                  <span>{method}</span>
                  <code>{path}</code>
                  <p>{note}</p>
                </div>
              ))}
            </div>
            <div className="api-console-status-row" aria-label="任务状态">
              <span>等待：{WAIT_STATUSES.join(' / ')}</span>
              <span>可下载：{SUCCESS_STATUSES.join(' / ')}</span>
              <span>失败：{FAILURE_STATUSES.join(' / ')}</span>
            </div>
            <ul className="api-console-error-list">
              {ERROR_CODES.map((errorCode) => <li key={errorCode}>{errorCode}</li>)}
            </ul>
          </section>
        </div>

        <div className="api-console-right">
          <section className="api-console-panel api-console-examples" aria-labelledby="api-examples-title">
            <div className="api-console-panel-head">
              <div>
                <h3 id="api-examples-title">语言示例</h3>
                <p>每个示例都包含创建、轮询和下载。</p>
              </div>
            </div>
            <CodeCard
              examples={ORDERED_API_EXAMPLES}
              activeId={activeExample.id}
              title={activeExample.language}
              description={activeExample.fileName}
              copyLabel={`${activeExample.language} 示例`}
              onActiveIdChange={setActiveLanguage}
              onCopy={(label, value) => void copyText(label, value)}
            />
          </section>

          <section className="api-console-panel api-console-ai-prompt" aria-labelledby="ai-prompt-title">
            <div className="api-console-panel-head">
              <div>
                <h3 id="ai-prompt-title">接入清单</h3>
                <p>包含认证、创建、轮询、下载和错误处理规则。</p>
              </div>
            </div>
            <CodeBlockFrame
              title="integration-prompt"
              fileName="可复制到项目说明或实现任务"
              code={AI_INTEGRATION_PROMPT}
              copyLabel="接入清单"
              wrap
              onCopy={(label, value) => void copyText(label, value)}
            />
          </section>
        </div>
      </div>

      {copied ? <div className="ok api-console-toast" role="status">已复制：{copied}</div> : null}
      {copyError ? <div className="error api-console-toast" role="alert">{copyError}</div> : null}
    </section>
  )
}

function CodeCard({ examples, activeId, title, description, copyLabel, onActiveIdChange, onCopy }: CodeCardProps) {
  const activeExample = examples.find((example) => example.id === activeId) || examples[0]
  const tabPanelId = `api-example-${activeExample.id}-panel`

  return (
    <div className="api-console-code-card">
      <div className="api-console-code-toolbar">
        <div className="api-console-code-tabs" role="tablist" aria-label="选择语言示例">
          {examples.map((example) => {
            const selected = activeExample.id === example.id
            return (
              <button
                key={example.id}
                type="button"
                role="tab"
                aria-selected={selected}
                aria-controls={selected ? tabPanelId : undefined}
                className={selected ? 'active' : ''}
                onClick={() => onActiveIdChange(example.id)}
              >
                {example.language}
              </button>
            )
          })}
        </div>
        <button className="api-console-copy-button" type="button" onClick={() => onCopy(copyLabel, activeExample.code)}>
          <span className="api-console-copy-icon" aria-hidden="true" />
          复制
        </button>
      </div>
      <div className="api-console-code-meta">
        <strong>{title}</strong>
        <span>{description}</span>
      </div>
      <CodeListing id={tabPanelId} code={activeExample.code} />
    </div>
  )
}

function CodeBlockFrame({ title, fileName, code, copyLabel, wrap = false, onCopy }: CodeBlockFrameProps) {
  return (
    <div className="api-console-code-card api-console-prompt-card">
      <div className="api-console-code-toolbar">
        <div className="api-console-code-title">
          <strong>{title}</strong>
          {fileName ? <span>{fileName}</span> : null}
        </div>
        <button className="api-console-copy-button" type="button" onClick={() => onCopy(copyLabel, code)}>
          <span className="api-console-copy-icon" aria-hidden="true" />
          复制
        </button>
      </div>
      <CodeListing code={code} wrap={wrap} />
    </div>
  )
}

function CodeListing({ id, code, wrap = false }: { id?: string; code: string; wrap?: boolean }) {
  const lines = code.split('\n')

  return (
    <pre id={id} className={`api-console-code-block${wrap ? ' api-console-code-block-wrap' : ''}`} role={id ? 'tabpanel' : undefined}>
      <code>
        {lines.map((line, index) => (
          <span className="api-console-code-line" key={`${index}-${line}`}>
            <span className="api-console-code-number" aria-hidden="true">{index + 1}</span>
            <span className="api-console-code-text">{line || ' '}</span>
          </span>
        ))}
      </code>
    </pre>
  )
}

function maskApiKey(value: string) {
  if (value.length <= 12) return 'lyra_sk_...'
  return `${value.slice(0, 8)}${'•'.repeat(10)}${value.slice(-4)}`
}

async function copyToClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // Fall through to the textarea fallback for restricted browser contexts.
    }
  }

  const textArea = document.createElement('textarea')
  textArea.value = value
  textArea.setAttribute('readonly', '')
  textArea.style.position = 'fixed'
  textArea.style.left = '-9999px'
  textArea.style.top = '0'
  document.body.appendChild(textArea)
  try {
    textArea.focus()
    textArea.select()
    const ok = document.execCommand('copy')
    if (!ok) throw new Error('复制失败，请手动选择内容复制')
  } finally {
    textArea.value = ''
    textArea.remove()
  }
}
