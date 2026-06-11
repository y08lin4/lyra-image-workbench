package promptlibrary

import "testing"

func TestParseMarkdownExtractsPromptCards(t *testing.T) {
	input := `# Awesome GPT Image 2 简体中文

<img src="assets/banner/readme-header-16x9.png" />

## 📷 摄影与照片级写实

### 便利店夜景
| Nano Banana 2 | GPT-Image |
|:-------------:|:---------:|
| <img width="400" alt="Nano Banana 2" src="assets/opennana/example.jpg" /> | ![GPT-Image](https://example.com/image.jpg) |

**提示词:**
` + "```text" + `
在夏夜晚上 10 点的便利店门口，生成一张超写实的城市街头多人合影。
` + "```" + `
**来源:** [卡尔的AI沃茨](https://mp.weixin.qq.com/s/example)

### 没有提示词的条目
<img src="https://example.com/skip.jpg" />

## 🎮 游戏与娱乐

### 像素城市
![image](https://example.com/city.png)

**Prompt:**
` + "```text" + `
Create a pixel-art city at dusk.
` + "```" + `
**Source:** [Creator](https://x.com/example)
`

	items := parseMarkdown(input, parseOptions{owner: "ZeroLu", repo: "awesome-gpt-image", branch: "main", readmePath: "README.zh-CN.md", readmeURL: "https://github.com/ZeroLu/awesome-gpt-image/blob/main/README.zh-CN.md"})
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2: %#v", len(items), items)
	}
	first := items[0]
	if first.Title != "便利店夜景" || first.Category != "📷 摄影与照片级写实" {
		t.Fatalf("unexpected first metadata: %#v", first)
	}
	if first.Prompt != "在夏夜晚上 10 点的便利店门口，生成一张超写实的城市街头多人合影。" {
		t.Fatalf("unexpected prompt: %q", first.Prompt)
	}
	if len(first.Images) != 2 {
		t.Fatalf("first images len=%d, want 2: %#v", len(first.Images), first.Images)
	}
	if first.Images[0].URL != "https://raw.githubusercontent.com/ZeroLu/awesome-gpt-image/main/assets/opennana/example.jpg" {
		t.Fatalf("relative image was not resolved: %s", first.Images[0].URL)
	}
	if len(first.Sources) != 1 || first.Sources[0].Label != "卡尔的AI沃茨" {
		t.Fatalf("source not parsed: %#v", first.Sources)
	}
	if items[1].Prompt != "Create a pixel-art city at dusk." {
		t.Fatalf("english prompt not parsed: %q", items[1].Prompt)
	}
}

func TestFilterLibrary(t *testing.T) {
	lib := Library{Categories: []string{"A", "B"}, Items: []Item{
		{Title: "猫", Category: "A", Prompt: "橘猫在窗边"},
		{Title: "狗", Category: "B", Prompt: "小狗在草地"},
	}}
	filtered := filterLibrary(lib, Query{Q: "橘猫", Limit: 10})
	if filtered.Total != 2 || filtered.Matching != 1 || len(filtered.Items) != 1 || filtered.Items[0].Title != "猫" {
		t.Fatalf("unexpected filtered library: %#v", filtered)
	}
}
