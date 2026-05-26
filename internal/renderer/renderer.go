package renderer

import (
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/privman/sbx-to-pdf/internal/model"
)

// Standard screenplay page layout (US Letter, Courier 12pt).
const (
	pageW = 8.5 * 72  // 612 pt
	pageH = 11.0 * 72 // 792 pt

	marginLeft   = 1.5 * 72  // 108 pt
	marginRight  = 1.0 * 72  // 72 pt
	marginTop    = 1.0 * 72  // 72 pt
	marginBottom = 1.0 * 72  // 72 pt

	fontSize   = 12.0
	lineHeight = fontSize // Courier 12pt, single-spaced

	// Element-specific left offsets (from page left edge).
	actionLeft        = marginLeft
	sceneHeadingLeft  = marginLeft
	generalLeft       = marginLeft
	characterLeft     = 3.7 * 72
	dialogueLeft      = 2.5 * 72
	dialogueRight     = 6.0 * 72
	parentheticalLeft = 3.1 * 72
	parentheticalRight = 5.6 * 72
	transitionRight   = pageW - marginRight

	// Scene number positions.
	sceneNumLeftX  = marginLeft - 0.5*72 // left of heading
	sceneNumRightX = pageW - marginRight + 0.25*72

	// Page number position.
	pageNumY = 0.5 * 72
	pageNumX = pageW - marginRight

	// Spacing between elements.
	elementGap = lineHeight // one blank line between elements
)

type Options struct {
	OutputPath    string
	OmitSceneNums bool
	OmitPageNums  bool
}

type Renderer struct {
	pdf  *fpdf.Fpdf
	opts Options
	y    float64
	page int
}

func Render(elements []model.Element, opts Options) error {
	r := &Renderer{
		pdf:  fpdf.NewCustom(&fpdf.InitType{UnitStr: "pt", Size: fpdf.SizeType{Wd: pageW, Ht: pageH}}),
		opts: opts,
	}
	r.pdf.SetMargins(marginLeft, marginTop, marginRight)
	r.pdf.SetAutoPageBreak(false, marginBottom)
	r.pdf.SetFont("Courier", "", fontSize)

	r.newPage()

	for i := 0; i < len(elements); i++ {
		elem := &elements[i]

		// Calculate how much vertical space this element needs.
		elemH := r.measureElement(elem)

		// Check keep-with-next: if this is a heading type, compute the full
		// chain height (all consecutive headings + first line of next paragraph).
		if elem.Type.IsHeading() {
			chainH := r.measureChain(elements, i)
			if r.y+chainH > pageH-marginBottom {
				r.newPage()
			}
			r.drawElement(elem)
			continue
		}

		// Paragraph element (Action or Dialogue): check if it fits.
		if r.y+elemH <= pageH-marginBottom {
			r.drawElement(elem)
			continue
		}

		// Doesn't fit entirely. For dialogue, we can split with CONT'D.
		if elem.Type == model.Dialogue {
			r.splitDialogue(elements, i)
			continue
		}

		// For action/other: start new page if even the first line doesn't fit.
		if r.y+lineHeight > pageH-marginBottom {
			r.newPage()
		}
		r.drawElement(elem)
	}

	return r.pdf.OutputFileAndClose(opts.OutputPath)
}

// newPage adds a page and resets y. Draws page number if needed.
func (r *Renderer) newPage() {
	r.pdf.AddPage()
	r.page++
	if !r.opts.OmitPageNums && r.page > 1 {
		r.pdf.SetFont("Courier", "", fontSize)
		numStr := fmt.Sprintf("%d.", r.page)
		w := r.pdf.GetStringWidth(numStr)
		r.pdf.Text(pageNumX-w, pageNumY+fontSize, numStr)
	}
	r.y = marginTop
}

// measureElement returns the rendered height of a single element.
func (r *Renderer) measureElement(elem *model.Element) float64 {
	left, right := r.elementMargins(elem.Type)
	width := right - left
	text := normalizeText(elem.PlainText())

	if elem.Type == model.SceneHeading || elem.Type == model.Character {
		text = strings.ToUpper(text)
	}

	lines := r.wrapText(text, width)
	h := float64(len(lines)) * lineHeight
	return h
}

// measureChain measures a heading chain + the first line of the next paragraph.
func (r *Renderer) measureChain(elements []model.Element, start int) float64 {
	h := 0.0
	for j := start; j < len(elements); j++ {
		elem := &elements[j]
		if j > start && !elem.Type.IsHeading() {
			// Add just the first line of this paragraph.
			h += lineHeight
			break
		}
		h += r.measureElement(elem)
		if j > start {
			h += elementGap
		}
	}
	return h
}

// splitDialogue handles a dialogue element that must be split across pages.
// It repeats the character name with (CONT'D) at the top of the new page.
func (r *Renderer) splitDialogue(elements []model.Element, idx int) {
	elem := &elements[idx]
	left, right := r.elementMargins(elem.Type)
	width := right - left
	text := normalizeText(elem.PlainText())
	lines := r.wrapText(text, width)

	avail := pageH - marginBottom - r.y
	fitLines := int(avail / lineHeight)
	if fitLines < 1 {
		fitLines = 0
	}
	if fitLines > len(lines) {
		fitLines = len(lines)
	}

	// Render lines that fit on this page.
	if fitLines > 0 {
		r.pdf.SetFont("Courier", "", fontSize)
		for _, line := range lines[:fitLines] {
			r.pdf.Text(left, r.y+fontSize, line)
			r.y += lineHeight
		}
	}

	// Find the last character element before this dialogue.
	charName := r.findLastCharacter(elements, idx)

	r.newPage()

	// Repeat character name with (CONT'D).
	if charName != "" {
		charUpper := strings.ToUpper(normalizeText(charName))
		if !strings.HasSuffix(charUpper, "(CONT'D)") && !strings.HasSuffix(charUpper, "(CONT)") {
			charUpper += " (CONT'D)"
		}
		r.pdf.SetFont("Courier", "", fontSize)
		r.pdf.Text(characterLeft, r.y+fontSize, charUpper)
		r.y += lineHeight

		// Also repeat parenthetical if it was between character and dialogue.
		paren := r.findParentheticalBefore(elements, idx)
		if paren != "" {
			paren = normalizeText(paren)
			if !strings.HasPrefix(paren, "(") {
				paren = "(" + paren + ")"
			}
			r.pdf.Text(parentheticalLeft, r.y+fontSize, paren)
			r.y += lineHeight
		}
	}

	// Render remaining dialogue lines.
	if fitLines < len(lines) {
		r.pdf.SetFont("Courier", "", fontSize)
		for _, line := range lines[fitLines:] {
			if r.y+lineHeight > pageH-marginBottom {
				r.newPage()
				if charName != "" {
					charUpper := strings.ToUpper(normalizeText(charName))
					if !strings.HasSuffix(charUpper, "(CONT'D)") && !strings.HasSuffix(charUpper, "(CONT)") {
						charUpper += " (CONT'D)"
					}
					r.pdf.SetFont("Courier", "", fontSize)
					r.pdf.Text(characterLeft, r.y+fontSize, charUpper)
					r.y += lineHeight
				}
			}
			r.pdf.Text(left, r.y+fontSize, line)
			r.y += lineHeight
		}
	}

	r.y += elementGap
}

func (r *Renderer) findLastCharacter(elements []model.Element, beforeIdx int) string {
	for j := beforeIdx - 1; j >= 0; j-- {
		if elements[j].Type == model.Character {
			return elements[j].PlainText()
		}
		if elements[j].Type == model.SceneHeading {
			break
		}
	}
	return ""
}

func (r *Renderer) findParentheticalBefore(elements []model.Element, dialogueIdx int) string {
	if dialogueIdx > 0 && elements[dialogueIdx-1].Type == model.Parenthetical {
		return elements[dialogueIdx-1].PlainText()
	}
	return ""
}

// drawElement renders a single element at the current y position.
func (r *Renderer) drawElement(elem *model.Element) {
	left, right := r.elementMargins(elem.Type)
	width := right - left

	// Scene numbers for scene headings.
	if elem.Type == model.SceneHeading && !r.opts.OmitSceneNums && elem.SceneNum != "" {
		r.pdf.SetFont("Courier", "", fontSize)
		r.pdf.Text(sceneNumLeftX, r.y+fontSize, elem.SceneNum)
		// Right scene number: right-align.
		numW := r.pdf.GetStringWidth(elem.SceneNum)
		r.pdf.Text(sceneNumRightX-numW, r.y+fontSize, elem.SceneNum)
	}

	if elem.Type == model.Transition {
		r.drawTransition(elem)
		r.y += elementGap
		return
	}

	// Render with bold/italic span support.
	r.drawSpans(elem, left, width)

	// No gap after Character or Parenthetical — they bind tightly with Dialogue.
	if elem.Type != model.Character && elem.Type != model.Parenthetical {
		r.y += elementGap
	}
}

func (r *Renderer) drawTransition(elem *model.Element) {
	text := strings.ToUpper(normalizeText(elem.PlainText()))
	r.pdf.SetFont("Courier", "", fontSize)
	w := r.pdf.GetStringWidth(text)
	r.pdf.Text(transitionRight-w, r.y+fontSize, text)
	r.y += lineHeight
}

// drawSpans renders element text with inline bold/italic, word-wrapped.
func (r *Renderer) drawSpans(elem *model.Element, left, width float64) {
	forceUpper := elem.Type == model.SceneHeading || elem.Type == model.Character
	wrapParens := elem.Type == model.Parenthetical

	// Build a flat list of styled words for wrapping.
	type styledWord struct {
		text   string
		bold   bool
		italic bool
	}

	var words []styledWord
	for _, sp := range elem.Spans {
		t := normalizeText(sp.Text)
		if forceUpper {
			t = strings.ToUpper(t)
		}
		parts := strings.Fields(t)
		for _, p := range parts {
			words = append(words, styledWord{text: p, bold: sp.Bold, italic: sp.Italic})
		}
	}

	// Wrap parenthetical text in parens if not already.
	if wrapParens && len(words) > 0 {
		if !strings.HasPrefix(words[0].text, "(") {
			words[0].text = "(" + words[0].text
		}
		last := &words[len(words)-1]
		if !strings.HasSuffix(last.text, ")") {
			last.text = last.text + ")"
		}
	}

	if len(words) == 0 {
		return
	}

	// Wrap words into lines.
	type lineSegment struct {
		text   string
		bold   bool
		italic bool
	}
	type wrappedLine struct {
		segments []lineSegment
	}

	var lines []wrappedLine
	var curLine []lineSegment
	curWidth := 0.0

	flushLine := func() {
		if len(curLine) > 0 {
			lines = append(lines, wrappedLine{segments: curLine})
			curLine = nil
			curWidth = 0
		}
	}

	spaceWidth := r.getStringWidth(" ", false, false)

	for _, w := range words {
		ww := r.getStringWidth(w.text, w.bold, w.italic)
		needed := ww
		if curWidth > 0 {
			needed += spaceWidth
		}

		if curWidth > 0 && curWidth+needed > width {
			flushLine()
		}

		if curWidth > 0 {
			// Add space before word: append to last segment if same style, else new segment.
			if len(curLine) > 0 && curLine[len(curLine)-1].bold == w.bold && curLine[len(curLine)-1].italic == w.italic {
				curLine[len(curLine)-1].text += " " + w.text
			} else {
				// Add space to previous segment.
				if len(curLine) > 0 {
					curLine[len(curLine)-1].text += " "
					curWidth += spaceWidth
				}
				curLine = append(curLine, lineSegment{text: w.text, bold: w.bold, italic: w.italic})
			}
			curWidth += needed
		} else {
			curLine = append(curLine, lineSegment{text: w.text, bold: w.bold, italic: w.italic})
			curWidth = ww
		}
	}
	flushLine()

	// Render each line.
	for _, line := range lines {
		if r.y+lineHeight > pageH-marginBottom {
			r.newPage()
		}
		x := left
		for _, seg := range line.segments {
			style := ""
			if seg.bold {
				style += "B"
			}
			if seg.italic {
				style += "I"
			}
			r.pdf.SetFont("Courier", style, fontSize)
			r.pdf.Text(x, r.y+fontSize, seg.text)
			x += r.pdf.GetStringWidth(seg.text)
		}
		r.y += lineHeight
	}
}

func (r *Renderer) getStringWidth(s string, bold, italic bool) float64 {
	style := ""
	if bold {
		style += "B"
	}
	if italic {
		style += "I"
	}
	r.pdf.SetFont("Courier", style, fontSize)
	return r.pdf.GetStringWidth(s)
}

// normalizeText replaces Unicode characters unsupported by Courier with ASCII equivalents.
func normalizeText(s string) string {
	r := strings.NewReplacer(
		"‘", "'", // left single quote
		"’", "'", // right single quote
		"“", "\"", // left double quote
		"”", "\"", // right double quote
		"–", "-", // en dash
		"—", "--", // em dash
		"…", "...", // ellipsis
		" ", " ", // non-breaking space
	)
	return r.Replace(s)
}

// elementMargins returns the left x and right x for an element type.
func (r *Renderer) elementMargins(t model.ElementType) (float64, float64) {
	switch t {
	case model.SceneHeading:
		return sceneHeadingLeft, pageW - marginRight
	case model.Action:
		return actionLeft, pageW - marginRight
	case model.General:
		return generalLeft, pageW - marginRight
	case model.Character:
		return characterLeft, pageW - marginRight
	case model.Dialogue:
		return dialogueLeft, dialogueRight
	case model.Parenthetical:
		return parentheticalLeft, parentheticalRight
	case model.Transition:
		return marginLeft, transitionRight
	default:
		return actionLeft, pageW - marginRight
	}
}

// wrapText splits text into lines that fit within the given width.
func (r *Renderer) wrapText(text string, width float64) []string {
	r.pdf.SetFont("Courier", "", fontSize)
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		test := cur + " " + w
		if r.pdf.GetStringWidth(test) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur = test
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
