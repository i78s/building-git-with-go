	"building-git/lib/command/print_diff"
	rootPath  string
	args      []string
	options   DiffOption
	repo      *repository.Repository
	status    *repository.Status
	stdout    io.Writer
	stderr    io.Writer
	prindDiff *print_diff.PrintDiff
	Patch  bool
	prindDiff, _ := print_diff.NewPrintDiff(dir, stdout, stderr)
		rootPath:  rootPath,
		args:      args,
		options:   options,
		repo:      repo,
		stdout:    stdout,
		stderr:    stderr,
		prindDiff: prindDiff,
	} else if len(d.args) == 2 {
		d.diffCommits()
func (d *Diff) diffCommits() {
	if !d.options.Patch {
		return
	}

	a, _ := repository.NewRevision(d.repo, d.args[0]).Resolve(repository.COMMIT)
	b, _ := repository.NewRevision(d.repo, d.args[1]).Resolve(repository.COMMIT)
	d.prindDiff.PrintCommitDiff(a, b, d.repo.Database)
}

	if !d.options.Patch {
		return
	}
			d.prindDiff.PrintDiff(d.prindDiff.FromNothing(path), d.fromIndex(path))
			d.prindDiff.PrintDiff(d.fromHead(path), d.fromIndex(path))
			d.prindDiff.PrintDiff(d.fromHead(path), d.prindDiff.FromNothing(path))
	if !d.options.Patch {
		return
	}
			d.prindDiff.PrintDiff(d.fromIndex(path), d.fromFile(path))
			d.prindDiff.PrintDiff(d.fromIndex(path), d.prindDiff.FromNothing(path))
func (d *Diff) fromHead(path string) *print_diff.Target {
	return print_diff.NewTarget(path, aOid, aMode, blob.String())
func (d *Diff) fromIndex(path string) *print_diff.Target {
	return d.prindDiff.FromEntry(path, entry)
func (d *Diff) fromFile(path string) *print_diff.Target {
	stat := d.status.Stats[path]
	return print_diff.NewTarget(path, bOid, bMode, blob.String())