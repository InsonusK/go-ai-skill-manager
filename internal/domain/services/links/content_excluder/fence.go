package content_excluder

import (
	"strings"

	link_parser "github.com/InsonusK/go-ai-skill-manager/internal/domain/services/links/parser"
)

type fence struct {
	start, end int
	// info is the opening fence line's info string, trimmed, e.g.
	// "go title=x" for "```go title=x"; empty for an untagged block.
	info string
}

// fenced returns every fenced code block of text (``` or ~~~, indented at
// most 3 spaces), each ending after its closing fence line or at the end of
// text when unclosed.
func fenced(text string) []fence {
	var out []fence
	offset := 0
	start := -1
	marker := ""
	info := ""
	for _, line := range strings.SplitAfter(text, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		if indent <= 3 {
			token := strings.TrimSpace(trimmed)
			if start < 0 && (strings.HasPrefix(token, "```") || strings.HasPrefix(token, "~~~")) {
				char := token[0]
				n := 0
				for n < len(token) && token[n] == char {
					n++
				}
				marker = token[:n]
				start = offset
				info = strings.TrimSpace(token[n:])
			} else if start >= 0 && strings.HasPrefix(token, marker) && strings.Trim(token, string(marker[0])) == "" {
				out = append(out, fence{start, offset + len(line), info})
				start = -1
			}
		}
		offset += len(line)
	}
	if start >= 0 {
		out = append(out, fence{start, len(text), info})
	}
	return out
}

// CodeFences returns the span of every fenced code block of content (see
// fenced), whatever its info string -- fence lines included.
func CodeFences(content string) []link_parser.Span {
	out := []link_parser.Span{}
	for _, f := range fenced(content) {
		out = append(out, link_parser.Span{Start: f.start, End: f.end})
	}
	return out
}

// CodeBlock is a fenced code block of a file's content tagged with an info
// string: Info is its first word, e.g. "go" for "```go".
type CodeBlock struct {
	link_parser.Span
	Info string
}

// TaggedCodeBlocks returns every fenced code block of content that has an
// info string (see fenced) -- fence lines included. Untagged blocks are
// left out.
//
// Пример: "```go\nx\n```\n```\ny\n```\n" -> один блок "go" -- первые
// три строки.
func TaggedCodeBlocks(content string) []CodeBlock {
	out := []CodeBlock{}
	for _, f := range fenced(content) {
		if f.info != "" {
			out = append(out, CodeBlock{Span: link_parser.Span{Start: f.start, End: f.end}, Info: strings.Fields(f.info)[0]})
		}
	}
	return out
}
