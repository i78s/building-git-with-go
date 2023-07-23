	"strings"
	"unicode"

	"github.com/fatih/color"
	args     []string
	options  DiffOption
func NewDiff(dir string, args []string, options DiffOption, stdout, stderr io.Writer) (*Diff, error) {
		options:  options,
	if d.options.Cached {
		color.New(color.Bold).Fprintf(d.stdout, "new file mode %s\n", b.mode)
		color.New(color.Bold).Fprintf(d.stdout, "deleted file mode %s\n", a.mode)
		color.New(color.Bold).Fprintf(d.stdout, "old mode %s\n", a.mode)
		color.New(color.Bold).Fprintf(d.stdout, "new mode %s\n", b.mode)
	color.New(color.FgCyan).Fprintf(d.stdout, "%s\n", hunk.Header())

		text := strings.TrimRightFunc(edit.String(), unicode.IsSpace)

		switch edit.Type {
		case diff.EQL:
			fmt.Fprintf(d.stdout, "%s\n", text)
			break
		case diff.INS:
			color.New(color.FgGreen).Fprintf(d.stdout, "%s\n", text)
			break
		case diff.DEL:
			color.New(color.FgRed).Fprintf(d.stdout, "%s\n", text)
			break
		}