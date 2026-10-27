package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteMarkdownDestination(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"../12-appendix/8-aslr.md", "../12-appendix/8-aslr.html"},
		{"../12-appendix/8-aslr.md#part", "../12-appendix/8-aslr.html#part"},
		{"8-aslr.MD", "8-aslr.html"},
		{"notes.markdown", "notes.html"},
		{"../img/a.png", "../img/a.png"},
		{"#section", "#section"},
		{"https://example.com/a.md", "https://example.com/a.md"},
		{"mailto:a.md", "mailto:a.md"},
	}
	for _, tc := range cases {
		got := string(rewriteMarkdownDestination([]byte(tc.in)))
		if got != tc.want {
			t.Errorf("rewrite(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildRewritesMarkdownLinks(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "book.json"), `{"title":"links"}`)
	mustWrite(t, filepath.Join(root, "SUMMARY.md"), "# Summary\n\n* [拆解](6-develop-inst-debugger/5-disassemble.md)\n* [ASLR](12-appendix/8-aslr.md)\n")
	mustWrite(t, filepath.Join(root, "6-develop-inst-debugger", "5-disassemble.md"), "见 [12.8 认识ASLR](../12-appendix/8-aslr.md)。\n\n![图](../img/a.png)\n")
	mustWrite(t, filepath.Join(root, "12-appendix", "8-aslr.md"), "# ASLR\n")

	output := filepath.Join(root, "_book")
	b, err := NewBuilder(root, output)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Build(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(output, "6-develop-inst-debugger", "5-disassemble.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if !strings.Contains(html, `href="../12-appendix/8-aslr.html"`) {
		t.Fatalf("rewritten link missing:\n%s", html)
	}
	if strings.Contains(html, `8-aslr.md`) {
		t.Fatalf("markdown href was left unchanged:\n%s", html)
	}
	if !strings.Contains(html, `src="../img/a.png"`) {
		t.Fatalf("image src was rewritten:\n%s", html)
	}
}
