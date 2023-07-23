//go:build ignore

package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var programs []*exec.Cmd

var lastRun time.Time

var opts struct {
	exts         string
	patterns     string
	skipPatterns string
	interval     time.Duration
	clear        bool
	vt100        bool
	cmds         []string
}

func main() {
	flag.StringVar(&opts.exts, "exts", ".go .h .c .sql .json", "A space separated list of file extensions to watch")
	flag.StringVar(&opts.patterns, "patterns", "", "A space separated list of patterns to watch")
	flag.StringVar(&opts.skipPatterns, "skip-patterns", ".data/* .git/* .hg/* .svn/* node_modules/* watch.go", "A space separated list of patterns to skip in watch mode")
	flag.DurationVar(&opts.interval, "interval", 2*time.Second, "The interval that watch mode checks for file changes")
	flag.BoolVar(&opts.clear, "clear", false, "Clear the terminal before running commands")
	flag.BoolVar(&opts.vt100, "vt100", false, "Clear the terminal using a VT100 escape code")
	flag.Parse()

	opts.exts = strings.TrimSpace(opts.exts)
	opts.patterns = strings.TrimSpace(opts.patterns)
	opts.skipPatterns = strings.TrimSpace(opts.skipPatterns)

	cmds := flag.Args()

	run(cmds)

	skipPatterns := strings.Fields(opts.skipPatterns)
	watchPatterns := strings.Fields(opts.patterns)
	skip := func(path string) bool {
		path = filepath.ToSlash(path)

		for _, pattern := range watchPatterns {
			matched, err := filepath.Match(pattern, path)
			if err != nil {
				fmt.Printf("watch pattern error: %v\n", err)
			}
			if matched {
				return false
			}
		}

		for _, pattern := range skipPatterns {
			matched, err := filepath.Match(pattern, path)
			if err != nil {
				fmt.Printf("watch skip pattern error: %v\n", err)
			}
			if matched {
				return true
			}
		}

		return false
	}

	exts := make(map[string]struct{})
	for _, ext := range strings.Fields(opts.exts) {
		if ext == "" {
			continue
		}

		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}

		exts[ext] = struct{}{}
	}

	files := make(map[string]time.Time)
	for {
		var shouldRun bool

		_ = filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Completely skip directories that are in the skip patterns
			if entry.IsDir() && skip(path) {
				return filepath.SkipDir
			}

			// Skip any files that don't match watch extensions
			if _, ok := exts[filepath.Ext(path)]; !entry.IsDir() && !ok {
				return nil
			}

			// Individually skip directories/files that haven't been entirely
			// skipped by the previous check
			if entry.IsDir() || skip(path) {
				return nil
			}

			fi, err := entry.Info()
			if err != nil {
				return err
			}

			if modified, ok := files[path]; !shouldRun && ok {
				shouldRun = modified.Before(fi.ModTime()) && lastRun.Before(fi.ModTime())
			}

			files[path] = fi.ModTime()

			return nil
		})

		if shouldRun {
			run(cmds)
		}

		time.Sleep(opts.interval)
	}
}

func run(cmdStrs []string) {
	lastRun = time.Now()

	if opts.clear {
		clear()
	}

	// Kill any running programs
	for _, cmd := range programs {
		switch runtime.GOOS {
		case "windows":
			pid := strconv.Itoa(cmd.Process.Pid)
			exec.Command("taskkill", "/t", "/f", "/pid", pid).Run()

		default:
			panic(fmt.Sprintf("kill: not implemented for %v", runtime.GOOS))
		}
	}

	programs = nil

	// Rather than writing a parser for nested command line args we use this
	// regular expression
	// It should be fine for most use cases where it matches:
	// - Escaped double quotes:  "(\\"|[^"])+"
	// - Space separated values: [^\s\\]+
	// - Escaped spaces:         (\\+\s[^\s\\]+)*
	re := regexp.MustCompile(`"(\\"|[^"])+"|[^\s\\]+(\\+\s[^\s\\]+)*`)

	// Run command strings
	for i, cmdStr := range cmdStrs {
		fields := re.FindAllString(cmdStr, -1)
		for i := range fields {
			fields[i] = strings.ReplaceAll(fields[i], `\ `, " ")
			fields[i] = strings.ReplaceAll(fields[i], `\"`, `"`)
			fields[i] = strings.ReplaceAll(fields[i], `\\`, `\`)
		}

		program, args, message := command(fields[0], fields[1:]...)

		fmt.Println(message)

		cmd := exec.Command(program, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		programs = append(programs, cmd)

		if err := cmd.Start(); err != nil {
			fmt.Println(err)

			continue
		}

		if i < len(cmdStrs)-1 {
			// We need to let the last command exit the loop
			// so we can continue to watch files
			if err := cmd.Wait(); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func clear() {
	if opts.vt100 {
		fmt.Print("\033c")
	} else {
		switch runtime.GOOS {
		case "windows":
			cmd := exec.Command("cmd", "/c", "cls")
			cmd.Stdout = os.Stdout
			cmd.Run()

		default:
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
	}
}

func command(program string, args ...string) (string, []string, string) {
	messageValues := make([]any, len(args))
	for i, arg := range args {
		messageValues[i] = arg
	}

	verbs := make([]string, len(args))
	for i, arg := range args {
		if strings.IndexFunc(arg, unicode.IsSpace) >= 0 {
			verbs[i] = "%q"
		} else {
			verbs[i] = "%v"
		}
	}
	message := fmt.Sprintf("%v "+strings.Join(verbs, " "), append([]any{program}, messageValues...)...)

	return program, args, message
}
