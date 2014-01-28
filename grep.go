package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
)

var ErrNoMatch = errors.New("grep: no match")

var Flags struct {
	CountOnly         bool
	FilesWithMatch    bool
	FilesWithoutMatch bool
	Invert            bool
	LineNumbers       bool
	NoFilename        bool
}

var (
	printName bool
	stderr    io.Writer = os.Stderr
	stdin     io.Reader = os.Stdin
	stdout    io.Writer = os.Stdout
)

func init() {
	flag.BoolVar(&Flags.CountOnly, "c", false, `
	Suppress normal output; instead print a count of matching lines for
	each input file. With the -v, count non-matching lines.`)

	flag.BoolVar(&Flags.FilesWithMatch, "l", false, `
	Suppress normal output; instead print the name of each input file from
	which output would normally have been printed. The scanning will stop
	on the first match.`)

	flag.BoolVar(&Flags.FilesWithoutMatch, "L", false, `
	Suppress normal output; instead print the name of each input file from
	which no output would normally have been printed. The scanning will
	stop on the first match.`)

	flag.BoolVar(&Flags.Invert, "v", false, `
	Invert the sense of matching, to select non-matching lines.`)

	flag.BoolVar(&Flags.LineNumbers, "n", false, `
	Prefix each line of output with the line number within its input
	file.`)

	flag.BoolVar(&Flags.NoFilename, "h", false, `
	Suppress the prefixing of filenames on output when multiple files are
	searched.`)

}

func main() {
	// call cmdMain in a separate function so that it can use defer and
	// have them run before the exit.
	os.Exit(cmdMain())
}

func cmdMain() (exitCode int) {
	var cpuprofile = flag.String("cpuprofile", "", `
	Write CPU profile to this file.`)

	flag.Usage = func() {
		fmt.Fprintln(stderr, "usage: grep [flags] pattern [path ...]")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "creating cpu profile:", err)
			os.Exit(2)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if flag.NArg() == 0 {
		flag.Usage()
	}

	if err := Grep(flag.Args()[0], flag.Args()[1:]); err != nil {
		if err != ErrNoMatch {
			fmt.Fprintln(stderr, err)
		}
		return 2
	}

	return 0
}

func Grep(pattern string, globs []string) error {
	printName = !Flags.NoFilename && len(globs) > 1

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	if len(globs) == 0 {
		return grepFile("", stdin, re)
	}

	matchFiles := 0

	for _, glob := range globs {
		paths, err := filepath.Glob(glob)
		if err != nil {
			return err
		}
		for _, name := range paths {
			f, err := os.Open(name)
			if err != nil {
				return err
			}
			defer f.Close()

			if err := grepFile(name, f, re); err != nil {
				if err == ErrNoMatch {
					continue
				}
				return err
			}

			matchFiles++
		}
	}

	if matchFiles == 0 {
		return ErrNoMatch
	}

	return nil
}

func grepFile(name string, in io.Reader, pattern *regexp.Regexp) error {
	scanner := bufio.NewScanner(in)
	lineNumber := 0
	count := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if pattern.MatchString(line) == Flags.Invert {
			continue
		}

		if Flags.FilesWithoutMatch {
			return ErrNoMatch
		}

		if Flags.FilesWithMatch {
			if printName {
				fmt.Fprintln(stdout, name)
			}
			return nil
		}

		count++

		if Flags.CountOnly {
			continue
		}

		if printName {
			fmt.Fprint(stdout, name)
			fmt.Fprint(stdout, ":")
		}

		if Flags.LineNumbers {
			fmt.Fprint(stdout, lineNumber)
			fmt.Fprint(stdout, ":")
		}

		fmt.Fprintln(stdout, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if Flags.FilesWithoutMatch {
		if printName {
			fmt.Fprintln(stdout, name)
		}
	} else if Flags.CountOnly {
		if count > 0 {
			if printName {
				fmt.Fprint(stdout, name)
				fmt.Fprint(stdout, ":")
			}
			fmt.Fprintln(stdout, count)
		}
	}

	return nil
}
