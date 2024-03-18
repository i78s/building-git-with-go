package remotes

import (
	"regexp"
	"strings"
)

var REFSPEC_FORMAT = regexp.MustCompile(`^(\+?)([^:]+):([^:]+)$`)

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

func ParseRefspec(spec string) *RefSpec {
	matches := REFSPEC_FORMAT.FindStringSubmatch(spec)
	forced := matches[1] == "+"
	return NewRefSpec(
		matches[2],
		matches[3],
		forced,
	)
}

func ExpandRefspecs(specs []string, refs []string) map[string][]interface{} {
	refspecs := make([]*RefSpec, len(specs))
	for i, spec := range specs {
		refspecs[i] = ParseRefspec(spec)
	}

	mappings := make(map[string][]interface{})
	for _, spec := range refspecs {
		for k, v := range spec.MatchRefs(refs) {
			mappings[k] = v
		}
	}
	return mappings
}

func (r *RefSpec) MatchRefs(refs []string) map[string][]interface{} {
	mappings := make(map[string][]interface{})
	patternStr := "^" + strings.ReplaceAll(regexp.QuoteMeta(r.source), `\*`, "(.*)") + "$"
	pattern := regexp.MustCompile(patternStr)

	for _, ref := range refs {
		matches := pattern.FindStringSubmatch(ref)
		if matches != nil {
			dist := r.target
			if len(matches) > 1 {
				dist = regexp.MustCompile(`\*`).ReplaceAllString(dist, matches[1])
			}
			mappings[dist] = []interface{}{ref, r.forced}
		}
	}
	return mappings
}

func (r *RefSpec) String() string {
	spec := ""
	if r.forced {
		spec = "+"
	}
	return spec + strings.Join([]string{r.source, r.target}, ":")
}
