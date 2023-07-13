package diff

type EditType string

const (
	EQL EditType = "eql"
	INS EditType = "ins"
	DEL EditType = "del"
)

type Edit struct {
	etype EditType
	text  string
}

func NewEdit(etype EditType, text string) *Edit {
	return &Edit{
		etype: etype,
		text:  text,
	}
}

var symbols = map[EditType]string{
	EQL: " ",
	INS: "+",
	DEL: "-",
}

func (e Edit) String() string {
	return symbols[EditType(e.etype)] + e.text
}
