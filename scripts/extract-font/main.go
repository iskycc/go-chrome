// extract-font reads a binary file from a git tree-ish and writes it
// to disk. Used to recover the original variable CascadiaNextSC font
// after it was removed in a previous commit, so the bake-font tool
// can regenerate it at a heavier weight.
//
// Usage:
//
//	extract-font HEAD~1:assets/fonts/CascadiaNextSC.wght.ttf output.ttf
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: extract-font REV:PATH OUTPUT")
		os.Exit(2)
	}
	spec := os.Args[1]
	outPath := os.Args[2]

	cmd := exec.Command("git", "--no-pager", "cat-file", "-p", spec)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stdout pipe:", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}
	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	n, err := io.Copy(out, stdout)
	if err != nil && !strings.Contains(err.Error(), "exit status") {
		fmt.Fprintln(os.Stderr, "copy:", err)
		os.Exit(1)
	}
	if err := cmd.Wait(); err != nil {
		// git prints "exit status" sometimes when the pipe is closed
		// after the file ends; that's expected. Only fail on real errors.
		if !strings.Contains(err.Error(), "exit status") {
			fmt.Fprintln(os.Stderr, "wait:", err)
			os.Exit(1)
		}
	}
	_ = out.Close()
	st, _ := os.Stat(outPath)
	fmt.Printf("wrote %s (%d bytes, copied %d)\n", outPath, st.Size(), n)
}
