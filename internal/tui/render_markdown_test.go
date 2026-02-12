package tui

import "testing"

func TestRenderMarkdownANSI_DoesNotPanicForCodeBlocks(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "")

	md := "```go\npackage main\nfunc main() { println(\"hi\") }\n```"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderMarkdownANSI panicked: %v", r)
		}
	}()

	rendered, err := renderMarkdownANSI(md, 80)
	if err != nil {
		t.Fatalf("renderMarkdownANSI returned error: %v", err)
	}
	if rendered == "" {
		t.Fatalf("expected rendered output to be non-empty")
	}
}
