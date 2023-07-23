func assertDiff(
	t *testing.T,
	tmpDir string,
	args []string,
	options DiffOption,
	stdout *bytes.Buffer,
	stderr *bytes.Buffer,
	expected string,
) {
	cmd, err := NewDiff(tmpDir, args, options, stdout, stderr)
		assertDiff(t, tmpDir, []string{}, DiffOption{}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{Cached: true}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{Cached: true}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{Cached: true}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{Cached: true}, stdout, stderr, expected)
		assertDiff(t, tmpDir, []string{}, DiffOption{Cached: true}, stdout, stderr, expected)