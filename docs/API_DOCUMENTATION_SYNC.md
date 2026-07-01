# LyAi Image Generation API Documentation Sync Draft

> Target external repo: `y08lin4/LyAi-Image-Generation-API-Documentation`
>
> This file is a Markdown draft for manual synchronization. This task did not push to GitHub and did not access the external network.

## Copy-ready External Markdown

The section below can be copied as the external repository README or API quickstart.

---

# LyAi Image Generation API

LyAi Image Generation API turns the Lyra Image Workbench web workflow into a programmatic image generation API. Users register on the site, save their upstream image provider key into their protected cloud account space, create a Lyra Bearer Key, then call `/v1/*` endpoints from scripts, SDKs, automation tools, or another AI agent.

> Banana Nano / Banana provider has moved to the independent `banana` branch. Current `dev` API documentation covers Image-2 and `image-2（满血版）`; Banana stays on the independent `banana` branch and is not advertised as an available provider.

The API is task based:

1. Create an image generation task.
2. Poll the task until it reaches a terminal status.
3. Download every successful result image.
4. Optionally cancel a queued or running task.

Lyra stores generated images in the task result system and returns authenticated download URLs. The Bearer Key only authorizes access to Lyra; it is not the upstream NewAPI/OpenAI-compatible gateway key.

## Prerequisites

Before calling the API:

1. Open [https://ai-image.ailinyu.de/](https://ai-image.ailinyu.de/).
2. Register or sign in.
3. In the web settings page, save the Image-2 cloud upstream key (codex-key/cloud upstream key).
4. Generate a Lyra Bearer API Key in the API/developer key area.
5. Copy the Bearer Key immediately. It is shown only once and starts with `lyra_sk_`.

If no cloud upstream key is saved, the site will reject Bearer Key creation with `UPSTREAM_KEY_REQUIRED`. If the upstream key is later removed, task creation will also return `UPSTREAM_KEY_REQUIRED`.

## Base URL

```text
https://ai-image.ailinyu.de
```

All examples below use:

```bash
export LYRA_API_KEY="lyra_sk_xxx"
```

## Authentication

All `/v1/*` requests must include:

```http
Authorization: Bearer <LYRA_API_KEY>
```

Rules:

- `LYRA_API_KEY` is the Lyra Bearer Key, not the upstream provider key.
- Do not put Bearer Keys in browser frontend code.
- Prefer environment variables or server-side secret storage.
- `/v1/*` does not read `apiKey`, `bananaApiKey`, `X-Image-Workbench-API-Key`, or `X-Image-Workbench-Banana-API-Key` as upstream keys.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/v1/images/generations` | OpenAI-style text-to-image task creation. Recommended default for SDKs. |
| `POST` | `/v1/image-tasks` | Lyra native text-to-image task creation. |
| `GET` | `/v1/image-tasks/{taskId}` | Poll task snapshot, status, progress, results, and errors. |
| `GET` | `/v1/image-tasks/{taskId}/images/{index}` | Download one successful result image. |
| `POST` | `/v1/image-tasks/{taskId}/cancel` | Cancel a queued task or best-effort cancel a running task. |

## Create A Task

Recommended SDK entry:

```http
POST /v1/images/generations
Authorization: Bearer lyra_sk_xxx
Content-Type: application/json
```

Request:

```json
{
  "model": "image-2",
  "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
  "quality": "auto",
  "output_format": "png",
  "n": 1
}
```

Full image-2（满血版） size example:

```json
{
  "model": "image-2-4k",
  "prompt": "A 16:9 cinematic product poster for a translucent smart speaker",
  "size": "1536x864",
  "quality": "high",
  "output_format": "png",
  "n": 1
}
```

Success response:

```json
{
  "ok": true,
  "taskId": "img_20260627120000_abcd1234abcd1234",
  "consumedCredits": 1,
  "task": {
    "id": "img_20260627120000_abcd1234abcd1234",
    "provider": "image-2",
    "model": "image-2",
    "mode": "text-to-image",
    "status": "queued",
    "statusText": "排队中",
    "statusCode": "J100",
    "stage": "queued",
    "stageText": "排队中",
    "stageCode": "S100",
    "progress": 0,
    "results": [],
    "createdAt": "2026-06-27T12:00:00+08:00",
    "updatedAt": "2026-06-27T12:00:00+08:00"
  }
}
```

Native Lyra entry:

```http
POST /v1/image-tasks
Authorization: Bearer lyra_sk_xxx
Content-Type: application/json
```

Request:

```json
{
  "provider": "image-2",
  "model": "image-2",
  "mode": "text-to-image",
  "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
  "ratio": "1:1",
  "resolution": "standard",
  "quality": "auto",
  "outputFormat": "png",
  "count": 1,
  "concurrency": 1
}
```

The external API currently supports `text-to-image`. `image-to-image` reference upload for Bearer API callers is not part of the current external contract.

## Poll A Task

```http
GET /v1/image-tasks/{taskId}
Authorization: Bearer lyra_sk_xxx
```

Example response after success:

```json
{
  "ok": true,
  "task": {
    "id": "img_20260627120000_abcd1234abcd1234",
    "provider": "image-2",
    "model": "image-2",
    "mode": "text-to-image",
    "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
    "ratio": "1:1",
    "resolution": "standard",
    "quality": "auto",
    "outputFormat": "png",
    "count": 1,
    "consumedCredits": 1,
    "concurrency": 1,
    "status": "succeeded",
    "statusText": "已成功",
    "statusCode": "J300",
    "stage": "succeeded",
    "stageText": "已成功",
    "stageCode": "S300",
    "progress": 100,
    "results": [
      {
        "index": 0,
        "ok": true,
        "status": "succeeded",
        "statusText": "已成功",
        "statusCode": "J300",
        "imageUrl": "/v1/image-tasks/img_20260627120000_abcd1234abcd1234/images/0",
        "mime": "image/png",
        "bytes": 123456,
        "revisedPrompt": "",
        "actualSize": "1024x1024",
        "actualQuality": "auto",
        "outputFormat": "png",
        "elapsedMs": 48231
      }
    ],
    "createdAt": "2026-06-27T12:00:00+08:00",
    "updatedAt": "2026-06-27T12:00:50+08:00",
    "startedAt": "2026-06-27T12:00:02+08:00",
    "finishedAt": "2026-06-27T12:00:50+08:00"
  }
}
```

Polling rules:

- Poll every 2 to 5 seconds.
- Suggested SDK timeout: 15 minutes.
- `queued`, `running`: keep polling.
- `succeeded`: stop polling and download all `task.results` where `ok=true`.
- `partial_failed`: stop polling, download all `ok=true` results, and expose failed result entries to the caller.
- `failed`, `cancelled`, `interrupted`: stop polling and raise an error.

## Download A Result

```http
GET /v1/image-tasks/{taskId}/images/{index}
Authorization: Bearer lyra_sk_xxx
```

`index` comes from `task.results[].index`, starting at `0`. Only download results where `ok=true`.

The response body is image binary. Use the response `Content-Type` or the result `outputFormat` to choose a file extension.

Typical response headers:

```http
Content-Type: image/png
Cache-Control: private, max-age=86400
```

## Cancel A Task

```http
POST /v1/image-tasks/{taskId}/cancel
Authorization: Bearer lyra_sk_xxx
```

Response:

```json
{
  "ok": true,
  "task": {
    "id": "img_20260627120000_abcd1234abcd1234",
    "status": "cancelled",
    "stage": "cancelled",
    "progress": 100
  }
}
```

Queued tasks can be cancelled directly. Running tasks are cancelled on a best-effort basis; upstream billing cannot be guaranteed to stop once the upstream request has started.

## Error Format

JSON errors use this shape:

```json
{
  "ok": false,
  "code": "TASK_NOT_FOUND",
  "status": 404,
  "english": "TASK_NOT_FOUND",
  "chinese": "任务不存在",
  "message": "任务不存在"
}
```

## Error Codes

| HTTP | Code | Meaning |
| --- | --- | --- |
| `400` | `BAD_JSON` | Request body is not valid JSON. |
| `400` | `UPSTREAM_KEY_REQUIRED` | The account has no cloud upstream key for the selected provider. |
| `400` | `TASK_CREATE_FAILED` | Task creation failed, usually because of invalid parameters, unsupported mode, or upstream failure. |
| `401` | `UNAUTHORIZED` | Bearer Key is missing, malformed, or invalid. |
| `402` | `USER_CREDITS_NOT_ENOUGH` | The account does not have enough generation credits. |
| `404` | `TASK_NOT_FOUND` | Task does not exist or does not belong to this Bearer Key's account space. |
| `404` | `TASK_IMAGE_NOT_FOUND` | Image does not exist, result is not successful, or task does not belong to this Bearer Key's account space. |
| `429` | `AUTH_RATE_LIMITED` | Too many failed authentication attempts; retry later. |
| `500` | `INTERNAL_ERROR` | Server-side error. |

Task result entries may also contain `errorCode`, `errorText`, and `errorEnglish` when an individual image fails. Common upstream-derived result codes include `E_UPSTREAM_TIMEOUT`, `E_UPSTREAM_AUTH`, `E_UPSTREAM_RATE_LIMIT`, `E_UPSTREAM_QUOTA`, `E_PROVIDER_UNSUPPORTED_PARAM`, `E_OUTPUT_FORMAT_UNSUPPORTED`, and `E_SAVE_IMAGE_FAILED`.

## Parameter Table

| Field | Endpoint | Type | Required | Accepted values / behavior |
| --- | --- | --- | --- | --- |
| `prompt` | both create endpoints | string | yes | Text prompt. Empty prompts are rejected. |
| `model` | both | string | no | Use `image-2` for the base entry. Use `image-2-4k` for the UI label `image-2（满血版）`. Legacy `gpt-image-2` is still accepted and normalized to `image-2`. |
| `provider` | both | string | no | `image-2`, `image2`, `image-2-4k`, legacy `gpt-image-2`. Empty means `image-2`. |
| `size` | `/v1/images/generations` | string | no | For `image-2`, omit `size`; the server will not submit it upstream. For `image-2-4k` / `image-2（满血版）`, accepts `auto`, mapped preset sizes, or custom `WIDTHxHEIGHT`. |
| `ratio` | both | string | no | `auto`, `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`. Unknown values normalize to `1:1`. |
| `resolution` | both | string | no | `auto`, `standard`, `2k`, `4k`. Unknown values normalize to `standard`. |
| `quality` | both | string | no | `auto`, `low`, `medium`, `high`. Unknown values normalize to `auto`. |
| `output_format` | `/v1/images/generations` | string | no | `png`, `jpeg`, `jpg`, `webp`, `auto`. `jpg` becomes `jpeg`; unknown values become `png`. |
| `outputFormat` | `/v1/image-tasks`; also accepted by generation adapter | string | no | Same as `output_format`. |
| `n` | `/v1/images/generations` | integer | no | Number of images. Normalized to `1` through `24`. |
| `count` | `/v1/image-tasks`; also accepted by generation adapter | integer | no | Same as `n`. |
| `concurrency` | both | integer | no | Minimum `1`. Controls per-task parallel image generation. |
| `mode` | `/v1/image-tasks` | string | yes for native endpoint | Must be `text-to-image` for the current external API. |
| `apiKey` | none | string | no | Ignored for `/v1/*`; do not send upstream keys to external API requests. |
| `bananaApiKey` | none | string | no | Legacy/unsupported field ignored for `/v1/*`; do not send upstream keys to external API requests. |

### Size Mapping

| `size` | `ratio` | `resolution` |
| --- | --- | --- |
| `auto` | `auto` | `auto` |
| `1024x1024` | `1:1` | `standard` |
| `1024x1536` | `2:3` | `standard` |
| `1536x1024` | `3:2` | `standard` |
| `768x1024` | `3:4` | `standard` |
| `1024x768` | `4:3` | `standard` |
| `1008x1792` | `9:16` | `standard` |
| `1792x1008` | `16:9` | `standard` |
| `2048x2048` | `1:1` | `2k` |
| `1344x2016` | `2:3` | `2k` |
| `2016x1344` | `3:2` | `2k` |
| `1536x2048` | `3:4` | `2k` |
| `2048x1536` | `4:3` | `2k` |
| `1152x2048` | `9:16` | `2k` |
| `2048x1152` | `16:9` | `2k` |
| `2880x2880` | `1:1` | `4k` |
| `2336x3504` | `2:3` | `4k` |
| `3504x2336` | `3:2` | `4k` |
| `2448x3264` | `3:4` | `4k` |
| `3264x2448` | `4:3` | `4k` |
| `2160x3840` | `9:16` | `4k` |
| `3840x2160` | `16:9` | `4k` |

### Banana Migration Note

Banana Nano / Banana provider has moved to the independent `banana` branch. Current `dev` API documentation does not list Banana as an accepted provider or model family.

## Code Examples

All examples:

- Read the Bearer Key from `LYRA_API_KEY`.
- Create a task through `/v1/images/generations`.
- Poll `/v1/image-tasks/{taskId}`.
- Treat `failed`, `cancelled`, and `interrupted` as hard failures.
- Download the first `ok=true` result for `succeeded` or `partial_failed`.

### curl

```bash
export LYRA_API_KEY="lyra_sk_xxx"
BASE_URL="https://ai-image.ailinyu.de"
TERMINAL="succeeded partial_failed failed cancelled interrupted"
FAILURE="failed cancelled interrupted"

CREATE_JSON=$(curl --fail-with-body -sS -X POST "$BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $LYRA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "image-2","prompt":"A clean product photo of a translucent smart speaker on a stone pedestal","quality":"auto","output_format":"png","n":1}')

TASK_ID=$(python -c 'import json,sys; d=json.load(sys.stdin); print(d.get("taskId") or d["task"]["id"])' <<< "$CREATE_JSON")
echo "task_id=$TASK_ID"

while true; do
  TASK_JSON=$(curl --fail-with-body -sS "$BASE_URL/v1/image-tasks/$TASK_ID" \
    -H "Authorization: Bearer $LYRA_API_KEY")
  STATUS=$(python -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])' <<< "$TASK_JSON")
  PROGRESS=$(python -c 'import json,sys; print(json.load(sys.stdin)["task"].get("progress", 0))' <<< "$TASK_JSON")
  echo "task_id=$TASK_ID status=$STATUS progress=$PROGRESS"
  if [[ " $TERMINAL " == *" $STATUS "* ]]; then break; fi
  sleep 3
done

if [[ " $FAILURE " == *" $STATUS "* ]]; then
  echo "task failed: $STATUS" >&2
  echo "$TASK_JSON"
  exit 1
fi

OK_INDEX=$(python -c 'import json,sys; t=json.load(sys.stdin)["task"]; print(next((str(r.get("index", i)) for i,r in enumerate(t.get("results", [])) if r.get("ok")), ""))' <<< "$TASK_JSON")
if [ -z "$OK_INDEX" ]; then
  echo "task finished without an ok=true result" >&2
  echo "$TASK_JSON"
  exit 1
fi

curl --fail-with-body -L "$BASE_URL/v1/image-tasks/$TASK_ID/images/$OK_INDEX" \
  -H "Authorization: Bearer $LYRA_API_KEY" \
  --output "result-$OK_INDEX.png"
```

### TypeScript

```ts
type LyraTaskResult = { index: number; ok: boolean; outputFormat?: string };
type LyraTask = { id: string; status: string; progress?: number; statusText?: string; results?: LyraTaskResult[] };

type CreateResponse = { ok: boolean; taskId?: string; task: LyraTask; consumedCredits?: number };
type TaskResponse = { ok: boolean; task: LyraTask };

const BASE_URL = "https://ai-image.ailinyu.de";
const TERMINAL = new Set(["succeeded", "partial_failed", "failed", "cancelled", "interrupted"]);
const FAILURE = new Set(["failed", "cancelled", "interrupted"]);

async function request(path: string, init: RequestInit = {}) {
  const response = await fetch(BASE_URL + path, {
    ...init,
    headers: {
      Authorization: `Bearer ${process.env.LYRA_API_KEY}`,
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
  });
  if (!response.ok) throw new Error(await response.text());
  return response;
}

const created = await request("/v1/images/generations", {
  method: "POST",
  body: JSON.stringify({
    model: "image-2",
    prompt: "A clean product photo of a translucent smart speaker on a stone pedestal",
    quality: "auto",
    output_format: "png",
    n: 1,
  }),
}).then((res) => res.json() as Promise<CreateResponse>);

const taskId = created.taskId ?? created.task.id;
console.log("task_id=", taskId);

let task = created.task;
while (true) {
  task = await request(`/v1/image-tasks/${taskId}`)
    .then((res) => res.json() as Promise<TaskResponse>)
    .then((data) => data.task);
  console.log("task", task.id, task.status, task.progress ?? 0, task.statusText ?? "");
  if (TERMINAL.has(task.status)) break;
  await new Promise((resolve) => setTimeout(resolve, 3000));
}

if (FAILURE.has(task.status)) {
  throw new Error(`task ended without downloadable image: ${task.status} ${task.statusText ?? ""}`);
}

const okResult = task.results?.find((result) => result.ok);
if (!okResult) throw new Error("task finished but no result has ok=true");

const image = await request(`/v1/image-tasks/${taskId}/images/${okResult.index}`).then((res) => res.arrayBuffer());
const ext = okResult.outputFormat === "jpeg" ? "jpg" : (okResult.outputFormat || "png");
await import("node:fs/promises").then((fs) => fs.writeFile(`result-${okResult.index}.${ext}`, Buffer.from(image)));
```

### Python

```python
import os
import time
import requests

BASE_URL = "https://ai-image.ailinyu.de"
TERMINAL = {"succeeded", "partial_failed", "failed", "cancelled", "interrupted"}
FAILURE = {"failed", "cancelled", "interrupted"}
headers = {"Authorization": f"Bearer {os.environ['LYRA_API_KEY']}"}

created = requests.post(
    f"{BASE_URL}/v1/images/generations",
    headers={**headers, "Content-Type": "application/json"},
    json={
        "model": "image-2",
        "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
        "quality": "auto",
        "output_format": "png",
        "n": 1,
    },
    timeout=120,
)
created.raise_for_status()
created_payload = created.json()
task_id = created_payload.get("taskId") or created_payload["task"]["id"]
print("task_id=", task_id)

task = created_payload["task"]
while True:
    snapshot = requests.get(f"{BASE_URL}/v1/image-tasks/{task_id}", headers=headers, timeout=30)
    snapshot.raise_for_status()
    task = snapshot.json()["task"]
    print("task", task["id"], task["status"], task.get("progress", 0), task.get("statusText", ""))
    if task["status"] in TERMINAL:
        break
    time.sleep(3)

if task["status"] in FAILURE:
    raise SystemExit("task ended without downloadable image: {} {}".format(task["status"], task.get("statusText", "")))

ok_result = next((result for result in task.get("results", []) if result.get("ok")), None)
if not ok_result:
    raise SystemExit("task finished but no result has ok=true")

image = requests.get(f"{BASE_URL}/v1/image-tasks/{task_id}/images/{ok_result['index']}", headers=headers, timeout=120)
image.raise_for_status()
suffix = (ok_result.get("outputFormat") or "png").replace("jpeg", "jpg")
open(f"result-{ok_result['index']}.{suffix}", "wb").write(image.content)
```

### Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "https://ai-image.ailinyu.de"

func request(method, path string, body []byte) []byte {
	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LYRA_API_KEY"))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		panic(string(data))
	}
	return data
}

func main() {
	createdBytes := request("POST", "/v1/images/generations", []byte(`{"model": "image-2","prompt":"A clean product photo of a translucent smart speaker on a stone pedestal","quality":"auto","output_format":"png","n":1}`))
	var created map[string]any
	_ = json.Unmarshal(createdBytes, &created)
	taskID, _ := created["taskId"].(string)
	if taskID == "" {
		task, _ := created["task"].(map[string]any)
		taskID, _ = task["id"].(string)
	}
	if taskID == "" {
		panic("missing taskId")
	}
	fmt.Println("task_id=", taskID)

	terminal := map[string]bool{"succeeded": true, "partial_failed": true, "failed": true, "cancelled": true, "interrupted": true}
	failure := map[string]bool{"failed": true, "cancelled": true, "interrupted": true}
	status := ""
	var task map[string]any
	for {
		snapshotBytes := request("GET", "/v1/image-tasks/"+taskID, nil)
		var snapshot map[string]any
		_ = json.Unmarshal(snapshotBytes, &snapshot)
		task = snapshot["task"].(map[string]any)
		status = task["status"].(string)
		fmt.Println("task", task["id"], status, task["progress"])
		if terminal[status] {
			break
		}
		time.Sleep(3 * time.Second)
	}

	if failure[status] {
		panic("task ended without downloadable image: " + status)
	}
	okIndex := firstOKIndex(task)
	if okIndex < 0 {
		panic("task finished but no result has ok=true")
	}
	image := request("GET", fmt.Sprintf("/v1/image-tasks/%s/images/%d", taskID, okIndex), nil)
	_ = os.WriteFile(fmt.Sprintf("result-%d.png", okIndex), image, 0644)
}

func firstOKIndex(task map[string]any) int {
	results, ok := task["results"].([]any)
	if !ok {
		return -1
	}
	for fallbackIndex, item := range results {
		result, ok := item.(map[string]any)
		if !ok || result["ok"] != true {
			continue
		}
		if index, ok := result["index"].(float64); ok {
			return int(index)
		}
		return fallbackIndex
	}
	return -1
}
```

### Java

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.Set;

public class GenerateImage {
  static final String BASE_URL = "https://ai-image.ailinyu.de";
  static final HttpClient HTTP = HttpClient.newHttpClient();
  static final Set<String> TERMINAL = Set.of("succeeded", "partial_failed", "failed", "cancelled", "interrupted");
  static final Set<String> FAILURE = Set.of("failed", "cancelled", "interrupted");

  public static void main(String[] args) throws Exception {
    String body = "{\"model\":\"image-2\",\"prompt\":\"A clean product photo of a translucent smart speaker on a stone pedestal\",\"quality\":\"auto\",\"output_format\":\"png\",\"n\":1}";
    String created = send("POST", "/v1/images/generations", body);
    String taskId = stringField(created, "taskId");
    if (taskId.isBlank()) taskId = stringField(created, "id");
    if (taskId.isBlank()) throw new RuntimeException("missing taskId");
    System.out.println("task_id=" + taskId);

    String snapshot = created;
    String status = "";
    while (true) {
      snapshot = send("GET", "/v1/image-tasks/" + taskId, null);
      status = stringField(snapshot, "status");
      System.out.println("task " + taskId + " " + status);
      if (TERMINAL.contains(status)) break;
      Thread.sleep(3000);
    }

    if (FAILURE.contains(status)) throw new RuntimeException("task ended without downloadable image: " + status);
    int okIndex = firstOkIndex(snapshot);
    if (okIndex < 0) throw new RuntimeException("task finished but no result has ok=true");

    HttpRequest imageReq = HttpRequest.newBuilder(URI.create(BASE_URL + "/v1/image-tasks/" + taskId + "/images/" + okIndex))
      .header("Authorization", "Bearer " + System.getenv("LYRA_API_KEY"))
      .timeout(Duration.ofSeconds(120))
      .GET()
      .build();
    HttpResponse<byte[]> imageResp = HTTP.send(imageReq, HttpResponse.BodyHandlers.ofByteArray());
    if (imageResp.statusCode() >= 400) throw new RuntimeException("image download failed: HTTP " + imageResp.statusCode());
    Files.write(Path.of("result-" + okIndex + ".png"), imageResp.body());
  }

  static String send(String method, String path, String body) throws Exception {
    HttpRequest.Builder builder = HttpRequest.newBuilder(URI.create(BASE_URL + path))
      .header("Authorization", "Bearer " + System.getenv("LYRA_API_KEY"))
      .timeout(Duration.ofSeconds(120));
    if (body == null) builder.GET();
    else builder.header("Content-Type", "application/json").method(method, HttpRequest.BodyPublishers.ofString(body));
    HttpResponse<String> response = HTTP.send(builder.build(), HttpResponse.BodyHandlers.ofString());
    if (response.statusCode() >= 400) throw new RuntimeException(response.body());
    return response.body();
  }

  static String stringField(String json, String field) {
    String needle = "\"" + field + "\":";
    int at = json.indexOf(needle);
    if (at < 0) return "";
    int start = json.indexOf("\"", at + needle.length());
    int end = start >= 0 ? json.indexOf("\"", start + 1) : -1;
    return start >= 0 && end > start ? json.substring(start + 1, end) : "";
  }

  static int firstOkIndex(String json) {
    int cursor = 0;
    while (cursor < json.length()) {
      int okAt = json.indexOf("\"ok\":true", cursor);
      if (okAt < 0) okAt = json.indexOf("\"ok\": true", cursor);
      if (okAt < 0) return -1;
      int objectStart = json.lastIndexOf("{", okAt);
      int objectEnd = json.indexOf("}", okAt);
      if (objectStart >= 0 && objectEnd > objectStart) {
        String object = json.substring(objectStart, objectEnd);
        int indexAt = object.indexOf("\"index\"");
        if (indexAt >= 0) {
          int colon = object.indexOf(":", indexAt);
          int pos = colon + 1;
          while (pos < object.length() && Character.isWhitespace(object.charAt(pos))) pos++;
          int end = pos;
          while (end < object.length() && Character.isDigit(object.charAt(end))) end++;
          if (end > pos) return Integer.parseInt(object.substring(pos, end));
        }
      }
      cursor = okAt + 4;
    }
    return -1;
  }
}
```

## 给 AI 的完整提示词 / Complete Prompt For AI

Copy the prompt below into another AI assistant. The user only needs to provide `LYRA_API_KEY`.

```text
You are integrating LyAi Image Generation API. Do not ask the user to inspect the repository, browse external docs, or find additional API details. Everything needed is in this prompt. The user only needs to provide a Lyra Bearer Key through the LYRA_API_KEY environment variable.

Base URL:
https://ai-image.ailinyu.de

Authentication:
Use Authorization: Bearer <LYRA_API_KEY> for every /v1/* request. The key starts with lyra_sk_. It is a Lyra Bearer Key, not the upstream image provider key. Do not send apiKey, bananaApiKey, X-Image-Workbench-API-Key, or X-Image-Workbench-Banana-API-Key to /v1/*.

Prerequisites the user must have completed on the website:
1. Registered or signed in at https://ai-image.ailinyu.de/.
2. Saved the Image-2 cloud upstream key in web settings.
3. Generated and copied a Lyra Bearer API Key.

Create task:
POST https://ai-image.ailinyu.de/v1/images/generations
Headers:
Authorization: Bearer <LYRA_API_KEY>
Content-Type: application/json
Body:
{
  "model": "image-2",
  "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
  "quality": "auto",
  "output_format": "png",
  "n": 1
}

Alternative native create endpoint:
POST https://ai-image.ailinyu.de/v1/image-tasks
Body:
{
  "provider": "image-2",
  "model": "image-2",
  "mode": "text-to-image",
  "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
  "ratio": "1:1",
  "resolution": "standard",
  "quality": "auto",
  "outputFormat": "png",
  "count": 1,
  "concurrency": 1
}

Creation response includes ok, taskId, consumedCredits, and task. Read taskId from response.taskId or response.task.id.

Poll task:
GET https://ai-image.ailinyu.de/v1/image-tasks/{taskId}
Use the same Authorization header.

Polling behavior:
- Poll every 2 to 5 seconds.
- Default timeout should be about 15 minutes unless the user asks otherwise.
- queued and running mean keep polling.
- succeeded means download every result where ok=true.
- partial_failed means download every result where ok=true and also expose failed result entries to the caller.
- failed, cancelled, and interrupted mean stop and throw an error.

Download result:
GET https://ai-image.ailinyu.de/v1/image-tasks/{taskId}/images/{index}
Use the same Authorization header. The index comes from task.results[].index. Only download results where ok=true. The body is image binary.

Cancel task:
POST https://ai-image.ailinyu.de/v1/image-tasks/{taskId}/cancel
Queued tasks can be cancelled. Running task cancellation is best effort.

Supported fields:
- prompt: required string.
- model: optional. Use image-2 for the base entry; use image-2-4k for image-2（满血版）. Legacy gpt-image-2 still maps to image-2.
- provider: optional. image-2/image2/image-2-4k/legacy gpt-image-2.
- size: omit for image-2. For image-2-4k, use auto, mapped preset sizes, or custom WIDTHxHEIGHT. Custom width/height must be divisible by 16, ratio 1:3 to 3:1, and no more than 3840 edge / 3840x2160 total pixels.
- ratio: auto, 1:1, 2:3, 3:2, 3:4, 4:3, 9:16, 16:9.
- resolution: auto, standard, 2k, 4k.
- quality: auto, low, medium, high.
- output_format/outputFormat: png, jpeg/jpg, webp, auto.
- n/count: normalized to 1 through 24.
- concurrency: minimum 1.
- External API currently supports text-to-image only.

Important errors:
- 400 BAD_JSON: invalid JSON.
- 400 UPSTREAM_KEY_REQUIRED: user has not saved the provider's cloud upstream key.
- 400 TASK_CREATE_FAILED: invalid parameters, unsupported mode, or upstream failure.
- 401 UNAUTHORIZED: Bearer missing or invalid.
- 402 USER_CREDITS_NOT_ENOUGH: not enough generation credits.
- 404 TASK_NOT_FOUND: task missing or not in this Bearer Key's account space.
- 404 TASK_IMAGE_NOT_FOUND: image missing or result not successful.
- 429 AUTH_RATE_LIMITED: too many invalid Bearer attempts; back off and retry later.
- 500 INTERNAL_ERROR: server error.

Now generate a complete script or SDK in the user's requested language. It must read LYRA_API_KEY, create a task, print taskId, poll with the terminal-status rules above, download the first ok=true result for succeeded or partial_failed, and raise clear errors for failed/cancelled/interrupted, HTTP errors, timeout, and AUTH_RATE_LIMITED.
```

---

## 站内/外文档口径清单 / Site/External Documentation Consistency Checklist

Keep the in-site API docs and the external documentation repository aligned on the items below.

| Area | Canonical value / wording | Keep aligned in |
| --- | --- | --- |
| External docs repo | `y08lin4/LyAi-Image-Generation-API-Documentation` | External README and in-site GitHub link |
| Registration URL | `https://ai-image.ailinyu.de/` | In-site header copy, prerequisites, AI prompt |
| Base URL | `https://ai-image.ailinyu.de` | Code examples, endpoint tables, AI prompt |
| Auth header | `Authorization: Bearer <API_KEY>` | Every language example |
| Bearer prefix | `lyra_sk_...` | Key UI, prerequisites, security notes |
| Upstream key note | Bearer Key is not the upstream key; users must save the cloud upstream key before generating or using Bearer API calls | Prerequisites, auth section, error docs |
| Recommended create endpoint | `POST /v1/images/generations` | SDK examples and AI prompt |
| Native create endpoint | `POST /v1/image-tasks` with `mode: text-to-image` | Endpoint table and native request example |
| Poll endpoint | `GET /v1/image-tasks/{taskId}` | SDK examples and polling section |
| Download endpoint | `GET /v1/image-tasks/{taskId}/images/{index}` | SDK examples and download section |
| Cancel endpoint | `POST /v1/image-tasks/{taskId}/cancel` | Endpoint table and cancel section |
| Example order | curl, TypeScript, Python, Go, Java | In-site tabs and external README |
| Example prompt | `A clean product photo of a translucent smart speaker on a stone pedestal` | All examples |
| Example payload | `model=image-2`, `quality=auto`, `output_format=png`, `n=1` | All examples |
| Terminal statuses | `succeeded`, `partial_failed`, `failed`, `cancelled`, `interrupted` | Examples, AI prompt, polling docs |
| Failure statuses | `failed`, `cancelled`, `interrupted` | Examples and AI prompt |
| Partial success behavior | Download all or first `ok=true` result, expose failed result entries to callers | Polling docs and examples |
| Error response fields | `ok`, `code`, `status`, `english`, `chinese`, `message` | Error section |
| Key error codes | `BAD_JSON`, `UPSTREAM_KEY_REQUIRED`, `TASK_CREATE_FAILED`, `UNAUTHORIZED`, `USER_CREDITS_NOT_ENOUGH`, `TASK_NOT_FOUND`, `TASK_IMAGE_NOT_FOUND`, `AUTH_RATE_LIMITED`, `INTERNAL_ERROR` | Error section and AI prompt |
| Unsupported secret inputs | Do not document request-body `apiKey`/`bananaApiKey` or `X-Image-Workbench-*` headers as usable upstream-key paths | Auth section and AI prompt |

When the implementation changes, update the in-site API docs constants and this sync draft in the same work item, then manually sync the copy-ready section to the external repository.

