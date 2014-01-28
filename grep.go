// Command grep provides the similar functionality as the popular grep utility.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
)

var Flags struct {
	CountOnly         bool
	FilesWithMatch    bool
	FilesWithoutMatch bool
	Invert            bool
	LineNumbers       bool
	NoErrorMessages   bool
	NoFilename        bool
	Quiet             bool
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

	flag.BoolVar(&Flags.NoErrorMessages, "s", false, `
	Suppress error messages about nonexistent or unreadable files.`)

	flag.BoolVar(&Flags.NoFilename, "h", false, `
	Suppress the prefixing of filenames on output when multiple files are
	searched.`)

	flag.BoolVar(&Flags.Quiet, "q", false, `
	Quiet; do not write anything to standard output. Exit immediately with
	zero status if any match is found, even if an error was detected.`)

}

func main() {
	// cmdMain exists for the ability to use defer and have them run before
	// the exit.
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

	if Grep(flag.Args()[0], flag.Args()[1:]) {
		return 0
	}

	return 2
}

// Grep searches the input files, or standard input if no files, for lines
// containing a match to the given pattern. By default, grep prints the
// matching lines. Returns true if any match; false otherwise.
//
func Grep(pattern string, globs []string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return false
	}

	// Important! Output can be suppressed after compiling pattern and
	// showing its error if any.
	if Flags.Quiet {
		stderr = ioutil.Discard
		stdout = ioutil.Discard
	} else if Flags.NoErrorMessages {
		stderr = ioutil.Discard
	}

	if len(globs) == 0 {
		return grepFile("", stdin, re)
	}

	matchFiles := 0

	for _, glob := range globs {
		paths, err := filepath.Glob(glob)
		if err != nil {
			fmt.Fprintf(stderr, "grep: %s: %s\n", glob, err)
			continue
		}

		// It's hard to predict if there are multiple files. Note That
		// for multiple files is file name printed, if not prevented by
		// Flags.NoFilename.
		printName = !Flags.NoFilename && (len(globs) > 1 || len(paths) > 1)

		if len(paths) == 0 {
			// This glob pattern has no matching file. Adding glob
			// to paths and continuing causes file not found, which
			// is wanted.
			paths = append(paths, glob)
		}

		for _, name := range paths {
			f, err := os.Open(name)
			if err != nil {
				fmt.Fprintf(stderr, "grep: %s: %s\n", name, err)
				continue
			}
			defer f.Close()

			if grepFile(name, f, re) {
				matchFiles++
			}
		}
	}

	return matchFiles > 0
}

func grepFile(name string, in io.Reader, pattern *regexp.Regexp) bool {
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
			return false
		}

		if Flags.Quiet {
			return true
		}

		if Flags.FilesWithMatch {
			if printName {
				fmt.Fprintln(stdout, name)
			}
			return true
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
		fmt.Fprintln(stderr, err)
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

	return count > 0
}
