package parser

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"

	"github.com/privman/sbx-to-pdf/internal/model"
)

var divTypeMap = map[string]model.ElementType{
	"divtype0": model.SceneHeading,
	"divtype1": model.General,
	"divtype2": model.Action,
	"divtype3": model.Character,
	"divtype4": model.Parenthetical,
	"divtype5": model.Dialogue,
	"divtype6": model.Transition,
}

func Parse(r io.Reader) ([]model.Element, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var elements []model.Element
	collectDivs(doc, &elements)
	return elements, nil
}

func collectDivs(n *html.Node, out *[]model.Element) {
	if n.Type == html.ElementNode && n.Data == "div" {
		cls := attr(n, "class")
		if et, ok := divTypeMap[cls]; ok {
			elem := model.Element{Type: et}
			if et == model.SceneHeading {
				elem.SceneNum = attr(n, "data-scene")
			}
			extractSpans(n, &elem.Spans, false, false)
			// Collapse whitespace in spans
			for i := range elem.Spans {
				elem.Spans[i].Text = collapseWhitespace(elem.Spans[i].Text)
			}
			// Merge adjacent spans with same style
			elem.Spans = mergeSpans(elem.Spans)
			// Drop elements with empty text
			if strings.TrimSpace(elem.PlainText()) != "" {
				*out = append(*out, elem)
			}
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectDivs(c, out)
	}
}

func extractSpans(n *html.Node, spans *[]model.Span, bold, italic bool) {
	if n.Type == html.TextNode {
		if n.Data != "" {
			*spans = append(*spans, model.Span{
				Text:   n.Data,
				Bold:   bold,
				Italic: italic,
			})
		}
		return
	}
	if n.Type == html.ElementNode {
		switch n.Data {
		case "strong", "b":
			bold = true
		case "em", "i":
			italic = true
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractSpans(c, spans, bold, italic)
	}
}

func mergeSpans(spans []model.Span) []model.Span {
	if len(spans) == 0 {
		return nil
	}
	merged := []model.Span{spans[0]}
	for _, s := range spans[1:] {
		last := &merged[len(merged)-1]
		if last.Bold == s.Bold && last.Italic == s.Italic {
			last.Text += s.Text
		} else {
			merged = append(merged, s)
		}
	}
	return merged
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				b.WriteRune(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return b.String()
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
