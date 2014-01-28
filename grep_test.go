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

var testdata = []struct {
	flags   string
	pattern string
	paths   string

	match      bool
	pathStderr string
	pathStdout string
	pathStdin  string
}{
	{
		"",
		"hello",
		"",

		true,
		"",
		"./testdata/hello stdin",
		"./testdata/hello stdin.in",
	},
	{
		"",
		"and|open",
		"./testdata/golang",

		true,
		"",
		"./testdata/andopen golang",
		"",
	},
	{
		"",
		"and|open",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/andopen golang,grep",
		"",
	},
	{
		"-c",
		"and|open",
		"./testdata/golang",

		true,
		"",
		"./testdata/c andopen golang",
		"",
	},
	{
		"-c",
		"and|open",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/c andopen golang,grep",
		"",
	},
	{
		"-c -v",
		"and|open",
		"./testdata/golang",

		true,
		"",
		"./testdata/cv andopen golang",
		"",
	},
	{
		"-c -v",
		"and|open",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/cv andopen golang,grep",
		"",
	},
	{
		"-h",
		"and",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/h and golang,grep",
		"",
	},
	{
		"-n",
		"and",
		"./testdata/golang",

		true,
		"",
		"./testdata/n and golang",
		"",
	},
	{
		"-n",
		"and",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/n and golang,grep",
		"",
	},
	{
		"-l",
		"and|open",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/l andopen golang,grep",
		"",
	},
	{
		"-L",
		"and|open",
		"./testdata/golang ./testdata/grep",

		false,
		"",
		"./testdata/ll andopen golang,grep",
		"",
	},
	{
		"-L",
		"nomatchforsure",
		"./testdata/golang ./testdata/grep",

		false,
		"",
		"./testdata/ll nomatchforsure golang,grep",
		"",
	},
	{
		"",
		"(",
		"./testdata/golang ./testdata/grep",

		false,
		"fake/whatever",
		"",
		"",
	},
	{
		"-s",
		"and|open",
		"./testdata/nonexistent ./testdata/golang ./testdata/grep",

		true,
		"",
		"./testdata/andopen golang,grep",
		"",
	},
	{
		"-c -q",
		"and|open",
		"./testdata/golang ./testdata/grep",

		true,
		"",
		"",
		"",
	},
}

func TestGrep(t *testing.T) {

	// This test is brutal no doubt. Let say next time it will be better.

	for _, test := range testdata {
		Flags.CountOnly = false
		Flags.FilesWithMatch = false
		Flags.FilesWithoutMatch = false
		Flags.Invert = false
		Flags.LineNumbers = false
		Flags.NoErrorMessages = false
		Flags.NoFilename = false
		Flags.Quiet = false

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
			case "-s":
				Flags.NoErrorMessages = true
			case "-h":
				Flags.NoFilename = true
			case "-q":
				Flags.Quiet = true
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

		match := Grep(test.pattern, paths)
		if match != test.match {
			t.Fatalf("context %q expected %v got %v", test.pathStdout, test.match, match)
		}

		switch test.pathStderr {
		case "":
			if buferr.String() != "" {
				t.Fatalf("expected stderr \"\" got %q", buferr.String())
			}
		case "fake/whatever":
			if buferr.String() == "" {
				t.Fatal("expected some stderr, got none")
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
					t.Fatal("context", test.pathStdout, "unexpected eof")
				}

				if goldenScan.Text() != actualScan.Text() {
					t.Fatalf("context %q expected %q got %q", test.pathStdout, goldenScan.Text(), actualScan.Text())
				}
			}

			if err := goldenScan.Err(); err != nil {
				t.Fatal(err)
			}

			if err := actualScan.Err(); err != nil {
				t.Fatal("unexpected error", err)
			}
		}

		if Flags.NoErrorMessages {
			if stderr != ioutil.Discard {
				t.Fatal("expected stderr set to ioutil.Discard if NoErrorMessages flag")
			}
		}

		if Flags.Quiet {
			if stderr != ioutil.Discard {
				t.Fatal("expected stderr set to ioutil.Discard if -q")
			}
			if stdout != ioutil.Discard {
				t.Fatal("expected stdout set to ioutil.Discard if -q")
			}
		}
	}
}
