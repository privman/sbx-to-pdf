package model

type ElementType int

const (
	SceneHeading    ElementType = iota // divtype0
	General                            // divtype1
	Action                             // divtype2
	Character                          // divtype3
	Parenthetical                      // divtype4
	Dialogue                           // divtype5
	Transition                         // divtype6
)

func (t ElementType) IsHeading() bool {
	switch t {
	case SceneHeading, Character, General, Parenthetical:
		return true
	}
	return false
}

type Span struct {
	Text   string
	Bold   bool
	Italic bool
}

type Element struct {
	Type     ElementType
	Spans    []Span
	SceneNum string // only for SceneHeading (from data-scene)
}

func (e *Element) PlainText() string {
	var s string
	for _, sp := range e.Spans {
		s += sp.Text
	}
	return s
}
