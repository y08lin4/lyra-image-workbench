import { useState } from 'react'

const REGISTER_URL = 'https://ai-image.ailinyu.de/'
const DOCS_REPO_URL = 'https://github.com/y08lin4/LyAi-Image-Generation-API-Documentation'
const API_BASE_URL = 'https://ai-image.ailinyu.de'

type ApiExample = {
  language: string
  fileName: string
  code: string
}

const REQUEST_BODY = '{"model":"gpt-image-2","prompt":"A clean product photo of a translucent smart speaker on a stone pedestal","size":"1024x1024"}'

const API_EXAMPLES: ApiExample[] = [
  {
    language: 'curl',
    fileName: 'terminal',
    code: [
      'curl -X POST "' + API_BASE_URL + '/v1/images/generations"',
      '-H "Authorization: Bearer $LYRA_API_KEY"',
      '-H "Content-Type: application/json"',
      "-d '" + REQUEST_BODY + "'",
    ].join(' '),
  },
  {
    language: 'Python',
    fileName: 'generate_image.py',
    code: [
      'import os',
      'import requests',
      '',
      'api_key = os.environ["LYRA_API_KEY"]',
      'response = requests.post(',
      '    "' + API_BASE_URL + '/v1/images/generations",',
      '    headers={"Authorization": f"Bearer {api_key}"},',
      '    json={',
      '        "model": "gpt-image-2",',
      '        "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",',
      '        "size": "1024x1024",',
      '    },',
      '    timeout=120,',
      ')',
      'response.raise_for_status()',
      'print(response.json())',
    ].join('\n'),
  },
  {
    language: 'Go',
    fileName: 'main.go',
    code: [
      'package main',
      '',
      'import (',
      '  "bytes"',
      '  "fmt"',
      '  "io"',
      '  "net/http"',
      '  "os"',
      ')',
      '',
      'func main() {',
      '  body := []byte("' + escapeForQuotedString(REQUEST_BODY) + '")',
      '  req, err := http.NewRequest("POST", "' + API_BASE_URL + '/v1/images/generations", bytes.NewReader(body))',
      '  if err != nil { panic(err) }',
      '  req.Header.Set("Authorization", "Bearer "+os.Getenv("LYRA_API_KEY"))',
      '  req.Header.Set("Content-Type", "application/json")',
      '  resp, err := http.DefaultClient.Do(req)',
      '  if err != nil { panic(err) }',
      '  defer resp.Body.Close()',
      '  data, _ := io.ReadAll(resp.Body)',
      '  fmt.Println(string(data))',
      '}',
    ].join('\n'),
  },
  {
    language: 'Java',
    fileName: 'GenerateImage.java',
    code: [
      'import java.net.URI;',
      'import java.net.http.HttpClient;',
      'import java.net.http.HttpRequest;',
      'import java.net.http.HttpResponse;',
      '',
      'public class GenerateImage {',
      '  public static void main(String[] args) throws Exception {',
      '    String body = "' + escapeForQuotedString(REQUEST_BODY) + '";',
      '    HttpRequest request = HttpRequest.newBuilder()',
      '      .uri(URI.create("' + API_BASE_URL + '/v1/images/generations"))',
      '      .header("Authorization", "Bearer " + System.getenv("LYRA_API_KEY"))',
      '      .header("Content-Type", "application/json")',
      '      .POST(HttpRequest.BodyPublishers.ofString(body))',
      '      .build();',
      '    HttpResponse<String> response = HttpClient.newHttpClient().send(request, HttpResponse.BodyHandlers.ofString());',
      '    System.out.println(response.body());',
      '  }',
      '}',
    ].join('\n'),
  },
  {
    language: 'JavaScript / Node',
    fileName: 'generate-image.mjs',
    code: [
      'const response = await fetch("' + API_BASE_URL + '/v1/images/generations", {',
      '  method: "POST",',
      '  headers: {',
      '    Authorization: "Bearer " + process.env.LYRA_API_KEY,',
      '    "Content-Type": "application/json",',
      '  },',
      '  body: JSON.stringify({',
      '    model: "gpt-image-2",',
      '    prompt: "A clean product photo of a translucent smart speaker on a stone pedestal",',
      '    size: "1024x1024",',
      '  }),',
      '});',
      '',
      'if (!response.ok) throw new Error(await response.text());',
      'console.log(await response.json());',
    ].join('\n'),
  },
]

const TASK_POLLING_EXAMPLE = [
  'curl "' + API_BASE_URL + '/v1/image-tasks/$TASK_ID"',
  '-H "Authorization: Bearer $LYRA_API_KEY"',
].join(' ')

const TASK_RESULT_EXAMPLE = [
  'curl "' + API_BASE_URL + '/v1/image-tasks/$TASK_ID/images/0"',
  '-H "Authorization: Bearer $LYRA_API_KEY"',
  '--output result.png',
].join(' ')
const AI_INTEGRATION_PROMPT = [
  '你正在接入 LyAi Image Generation API。',
  '注册站点：https://ai-image.ailinyu.de/',
  '先注册账号、配置上游服务、生成 Bearer API Key。',
  '所有请求都使用 Authorization: Bearer <API_KEY>。',
  '主要示例接口：POST https://ai-image.ailinyu.de/v1/images/generations。',
  '响应会返回 task.id；用 GET https://ai-image.ailinyu.de/v1/image-tasks/{task.id} 轮询状态。',
  '完成后的图片可通过 task.results[n].imageUrl 或 /v1/image-tasks/{task.id}/images/{index} 获取。',
  '文档仓库：https://github.com/y08lin4/LyAi-Image-Generation-API-Documentation。',
  '请生成包含错误处理、超时控制、环境变量读取 API Key 的接入代码。',
].join('\n')

export function ApiDocsPage() {
  const [copied, setCopied] = useState('')
  const [copyError, setCopyError] = useState('')

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
    <section className="workflow-page api-docs-page" aria-labelledby="api-docs-title">
      <header className="workflow-page-header api-docs-header">
        <div>
          <p className="eyebrow">API Docs</p>
          <h2 id="api-docs-title">API 文档</h2>
          <p>先去 <a href={REGISTER_URL} target="_blank" rel="noreferrer">注册站点</a> 注册、配置上游、生成 Bearer Key。</p>
        </div>
        <div className="api-docs-actions">
          <a href={DOCS_REPO_URL} target="_blank" rel="noreferrer">GitHub 文档仓库</a>
          <button type="button" onClick={() => void copyText('AI 提示词', AI_INTEGRATION_PROMPT)}>复制给 AI 的提示词</button>
        </div>
      </header>

      <section className="api-docs-panel" aria-labelledby="api-auth-title">
        <h3 id="api-auth-title">认证</h3>
        <ul>
          <li>Base URL：<code>{API_BASE_URL}</code></li>
          <li>Header：<code>Authorization: Bearer &lt;API_KEY&gt;</code></li>
          <li>示例模型：<code>gpt-image-2</code></li>
        </ul>
      </section>

      <section className="api-docs-examples" aria-label="代码示例">
        {API_EXAMPLES.map((example) => (
          <article key={example.language} className="api-docs-example">
            <div className="panel-title">
              <strong>{example.language}</strong>
              <span>{example.fileName}</span>
            </div>
            <pre><code>{example.code}</code></pre>
            <button type="button" onClick={() => void copyText(example.language, example.code)}>复制示例</button>
          </article>
        ))}
      </section>


      <section className="api-docs-panel" aria-labelledby="api-poll-title">
        <div className="panel-title">
          <strong id="api-poll-title">轮询任务和获取图片</strong>
          <button type="button" onClick={() => void copyText('轮询示例', TASK_POLLING_EXAMPLE + '\n' + TASK_RESULT_EXAMPLE)}>复制</button>
        </div>
        <ul>
          <li>创建接口会返回 <code>taskId</code> 和 <code>task.id</code>。</li>
          <li>轮询 <code>GET /v1/image-tasks/&#123;taskId&#125;</code>，直到 <code>task.status</code> 为 <code>succeeded</code>、<code>partial_failed</code> 或 <code>failed</code>。</li>
          <li>成功后读取 <code>task.results[n].imageUrl</code>，或下载 <code>/v1/image-tasks/&#123;taskId&#125;/images/0</code>。</li>
        </ul>
        <pre><code>{TASK_POLLING_EXAMPLE + '\n' + TASK_RESULT_EXAMPLE}</code></pre>
      </section>
      <section className="api-docs-panel" aria-labelledby="ai-prompt-title">
        <div className="panel-title">
          <strong id="ai-prompt-title">复制给 AI 的提示词</strong>
          <button type="button" onClick={() => void copyText('AI 提示词', AI_INTEGRATION_PROMPT)}>复制</button>
        </div>
        <pre><code>{AI_INTEGRATION_PROMPT}</code></pre>
      </section>

      {copied ? <div className="ok">已复制：{copied}</div> : null}
      {copyError ? <div className="error">{copyError}</div> : null}
    </section>
  )
}

function escapeForQuotedString(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
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
  textArea.focus()
  textArea.select()
  const ok = document.execCommand('copy')
  textArea.remove()
  if (!ok) throw new Error('复制失败，请手动选择内容复制')
}