package remotes

import "strings"

type RefSpec struct {
	source string
	target string
	forced bool
}

func NewRefSpec(source, target string, forced bool) *RefSpec {
	return &RefSpec{
		source: source,
		target: target,
		forced: forced,
	}
}

func (r *RefSpec) String() string {
	spec := ""
	if r.forced {
		spec = "+"
	}
	return spec + strings.Join([]string{r.source, r.target}, ":")
}
