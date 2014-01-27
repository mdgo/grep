package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// someError represents some, no matter of kind, error.
type someError struct{}

func (someError) Error() string { return "any error" }

var testdata = []struct {
	flags   string
	pattern string
	paths   string

	err        error
	pathStderr string
	pathStdout string
	pathStdin  string
}{
	{
		"",
		"hello",
		"",

		nil,
		"",
		"./testdata/hello stdin",
		"./testdata/hello stdin.in",
	},
	{
		"",
		"and|open",
		"./testdata/golang",

		nil,
		"",
		"./testdata/andopen golang",
		"",
	},
	{
		"",
		"and|open",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/andopen golang,grep",
		"",
	},
	{
		"-c",
		"and|open",
		"./testdata/golang",

		nil,
		"",
		"./testdata/c andopen golang",
		"",
	},
	{
		"-c",
		"and|open",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/c andopen golang,grep",
		"",
	},
	{
		"-c -v",
		"and|open",
		"./testdata/golang",

		nil,
		"",
		"./testdata/cv andopen golang",
		"",
	},
	{
		"-c -v",
		"and|open",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/cv andopen golang,grep",
		"",
	},
	{
		"-h",
		"and",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/h and golang,grep",
		"",
	},
	{
		"-n",
		"and",
		"./testdata/golang",

		nil,
		"",
		"./testdata/n and golang",
		"",
	},
	{
		"-n",
		"and",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/n and golang,grep",
		"",
	},
	{
		"-l",
		"and|open",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/l andopen golang,grep",
		"",
	},
	{
		"-L",
		"and|open",
		"./testdata/golang ./testdata/grep",

		ErrNoMatch,
		"",
		"./testdata/ll andopen golang,grep",
		"",
	},
	{
		"-L",
		"nomatchforsure",
		"./testdata/golang ./testdata/grep",

		nil,
		"",
		"./testdata/ll nomatchforsure golang,grep",
		"",
	},
	{
		"",
		"(",
		"./testdata/golang ./testdata/grep",

		someError{},
		"",
		"",
		"",
	},
}

func TestGrep(t *testing.T) {
	for _, test := range testdata {
		Flags.CountOnly = false
		Flags.FilesWithMatch = false
		Flags.FilesWithoutMatch = false
		Flags.Invert = false
		Flags.LineNumbers = false
		Flags.NoFilename = false

		for _, f := range strings.Split(test.flags, " ") {
			switch f {
			case "-c":
				Flags.CountOnly = true
			case "-l":
				Flags.FilesWithMatch = true
			case "-L":
				Flags.FilesWithoutMatch = true
			case "-v":
				Flags.Invert = true
			case "-n":
				Flags.LineNumbers = true
			case "-h":
				Flags.NoFilename = true
			}
		}

		buferr := &bytes.Buffer{}
		bufout := &bytes.Buffer{}

		stderr = buferr
		stdout = bufout

		if test.pathStdin != "" {
			f, err := os.Open(test.pathStdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			defer f.Close()
			b, err := ioutil.ReadAll(f)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			stdin = strings.NewReader(string(b))
		}

		var paths []string
		if test.paths != "" {
			paths = strings.Split(test.paths, " ")
		}

		err := Grep(test.pattern, paths)
		if _, ok := test.err.(someError); ok {
			if err == nil {
				t.Fatal("expected any error, got no error")
			}
		} else if err != test.err {
			t.Fatalf("expected %v got %v", test.err, err)
		}

		if test.pathStderr == "" {
			if buferr.String() != "" {
				t.Fatalf("expected stderr \"\" got %q", buferr.String())
			}
		}

		if test.pathStdout == "" {
			if bufout.String() != "" {
				t.Fatalf("expected stdout \"\" got %q", bufout.String())
			}
		} else {
			goldenFile, err := os.Open(test.pathStdout)
			if err != nil {
				t.Fatal(err)
			}
			defer goldenFile.Close()
			goldenScan := bufio.NewScanner(goldenFile)
			actualScan := bufio.NewScanner(bufout)

			for goldenScan.Scan() {
				if !actualScan.Scan() {
					t.Fatal("case", test.pathStdout, "unexpected eof")
				}

				if goldenScan.Text() != actualScan.Text() {
					t.Fatalf("expected %q got %q", goldenScan.Text(), actualScan.Text())
				}
			}

			if err := goldenScan.Err(); err != nil {
				t.Fatal(err)
			}

			if err := actualScan.Err(); err != nil {
				t.Fatal("unexpected error", err)
			}
		}
	}
}
