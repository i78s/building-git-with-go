package pager

import (
	"io"
	"os/exec"
)

func SetupPager(isTTY bool, stdout, stderr io.Writer) (io.Writer, func()) {
	if !isTTY {
		return stdout, func() {}
	}

	reader, writer := io.Pipe()

	cmd := exec.Command("less")
	cmd.Stdin = reader
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		return stdout, func() {}
	}

	go func() {
		cmd.Wait()
		reader.Close()
	}()

	return writer, func() {
		writer.Close()
		cmd.Wait()
	}
}
