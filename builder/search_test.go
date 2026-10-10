package builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSectionsFromHTML(t *testing.T) {
	html := `<p>开篇说明</p><h2 id="listen">监听文件</h2><p>修改 Markdown 后自动刷新。</p>`
	sections := sectionsFromHTML("热重载", html)
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2: %#v", len(sections), sections)
	}
	if sections[0].Heading != "热重载" || sections[0].Text != "开篇说明" || sections[0].Anchor != "" {
		t.Fatalf("preamble = %#v", sections[0])
	}
	if sections[1].Heading != "监听文件" || sections[1].Anchor != "listen" || sections[1].Text != "修改 Markdown 后自动刷新。" {
		t.Fatalf("section = %#v", sections[1])
	}
}

func TestPlainTextSeparatesTags(t *testing.T) {
	got := plainText(`<p>foo</p><p>bar</p> code &amp; text`)
	if got != "foo bar code & text" {
		t.Fatalf("plain text = %q", got)
	}
}

func TestBuildSearchIndex(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "book.json"), `{"title":"检索示例"}`)
	mustWrite(t, filepath.Join(root, "README.md"), "# 介绍\n\n这是一本用来测试全文检索的书。\n")
	mustWrite(t, filepath.Join(root, "SUMMARY.md"), "# Summary\n\n* [介绍](README.md)\n  * [热重载](guide/reload.md)\n")
	mustWrite(t, filepath.Join(root, "guide", "reload.md"), "# 热重载\n\n修改 Markdown 后，预览会自动刷新。\n\n## 监听文件\n\ngitbook serve 会监听文件变化。\n")

	output := filepath.Join(root, "_book")
	b, err := NewBuilder(root, output)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Build(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(output, "static", "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index SearchIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Pages) != 2 {
		t.Fatalf("pages = %d, want 2 (index.html should not duplicate README): %#v", len(index.Pages), index.Pages)
	}

	intro := index.Pages[0]
	if intro.Title != "介绍" || intro.URL != "/README.html" {
		t.Fatalf("intro = %#v", intro)
	}
	if !sectionContains(intro.Sections, "全文检索") {
		t.Fatalf("intro sections missing body text: %#v", intro.Sections)
	}

	reload := index.Pages[1]
	if reload.Title != "热重载" || reload.URL != "/guide/reload.html" {
		t.Fatalf("reload = %#v", reload)
	}
	var listen *SearchSection
	for i := range reload.Sections {
		if reload.Sections[i].Heading == "监听文件" {
			listen = &reload.Sections[i]
		}
	}
	if listen == nil || listen.Anchor == "" || !strings.Contains(listen.Text, "监听文件变化") {
		t.Fatalf("listen section = %#v", reload.Sections)
	}
}

func TestBuildSearchIndexWithoutSummaryEntry(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "book.json"), `{"title":"Only Readme"}`)
	mustWrite(t, filepath.Join(root, "README.md"), "# 只有首页\n\n正文里有一个独特词条。\n")
	mustWrite(t, filepath.Join(root, "SUMMARY.md"), "# Summary\n\n* [空目录]()\n")

	output := filepath.Join(root, "_book")
	b, err := NewBuilder(root, output)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Build(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(output, "static", "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index SearchIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Pages) != 1 || index.Pages[0].URL != "/" || index.Pages[0].Title != "只有首页" {
		t.Fatalf("pages = %#v", index.Pages)
	}
	if !sectionContains(index.Pages[0].Sections, "独特词条") {
		t.Fatalf("sections = %#v", index.Pages[0].Sections)
	}
}

func sectionContains(sections []SearchSection, needle string) bool {
	for _, section := range sections {
		if strings.Contains(section.Heading, needle) || strings.Contains(section.Text, needle) {
			return true
		}
	}
	return false
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
