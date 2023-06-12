//go:build ignore

package main

import (
	"bytes"
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

// Tags for builds and test builds that are always applied regardless
// of provided flag values
const (
	stickyTags     = "json1 fts5"
	stickyTestTags = ""
)

var programs []*exec.Cmd

var lastRun time.Time

type flagStrs []string

func (s *flagStrs) String() string {
	var sb strings.Builder

	for _, str := range *s {
		sb.WriteString(str)
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func (s *flagStrs) Set(value string) error {
	*s = append(*s, value)

	return nil
}

var opts struct {
	goos              string
	goarch            string
	tags              string
	testTags          string
	unoptimised       bool
	debug             bool
	race              bool
	build             bool
	clear             bool
	cover             bool
	vet               bool
	test              bool
	verbose           bool
	before            flagStrs
	after             flagStrs
	watch             bool
	watchExts         string
	watchPatterns     string
	watchSkipPatterns string
	watchInterval     time.Duration
}

func main() {
	tagsDescription := "Build tags"
	if stickyTags != "" {
		tagsDescription = fmt.Sprintf("Additional build tags (constant tags: %q)", stickyTags)
	}

	testTagsDescription := "Test build tags"
	if stickyTags != "" || stickyTestTags != "" {
		testTagsDescription = fmt.Sprintf("Additional test build tags (constant tags: %q)", strings.TrimSpace(stickyTags+" "+stickyTestTags))
	}

	flag.StringVar(&opts.goos, "goos", "", "Sets the GOOS environment variable for the build")
	flag.StringVar(&opts.goarch, "goarch", "", "Sets the GOARCH environment variable for the build")
	flag.StringVar(&opts.tags, "tags", "", tagsDescription)
	flag.StringVar(&opts.testTags, "test-tags", "", testTagsDescription)
	flag.BoolVar(&opts.unoptimised, "unoptimised", false, "Disable optimisations/inlining")
	flag.BoolVar(&opts.debug, "debug", false, "Enable symbol table/DWARF generation and disable optimisations/inlining")
	flag.BoolVar(&opts.race, "race", false, "Enable data race detection")
	flag.BoolVar(&opts.build, "build", false, "Run go build")
	flag.BoolVar(&opts.clear, "clear", false, "Clear the terminal before build")
	flag.BoolVar(&opts.cover, "cover", false, "Generate an HTML cover report (opened in the browser if watch is disabled)")
	flag.BoolVar(&opts.vet, "vet", false, "Run go vet before build")
	flag.BoolVar(&opts.test, "test", false, "Run go test before build")
	flag.BoolVar(&opts.verbose, "verbose", false, "Print the commands that are being run along with all command output")
	flag.Var(&opts.before, "before", "Commands to run before a build is started")
	flag.Var(&opts.after, "after", "Commands to run after a build has completed")
	flag.BoolVar(&opts.watch, "watch", false, "Watches for changes and re-runs the build if changes are detected")
	flag.StringVar(&opts.watchExts, "watch-exts", ".go .h .c .sql .json", "A space separated list of file extensions to watch")
	flag.StringVar(&opts.watchPatterns, "watch-patterns", "", "A space separated list of patterns to watch")
	flag.StringVar(&opts.watchSkipPatterns, "watch-skip-patterns", ".data/* .git/* .hg/* .svn/* node_modules/* build.go", "A space separated list of patterns to skip in watch mode")
	flag.DurationVar(&opts.watchInterval, "watch-interval", 2*time.Second, "The interval that watch mode checks for file changes")
	flag.Parse()

	if s := strings.TrimSpace(stickyTags); s != "" {
		opts.tags = s + " " + opts.tags
	}
	opts.tags = strings.TrimSpace(opts.tags)

	if s := strings.TrimSpace(stickyTestTags); s != "" {
		opts.testTags = s + " " + opts.testTags
	}
	opts.testTags = strings.TrimSpace(opts.tags + " " + opts.testTags)

	opts.watchExts = strings.TrimSpace(opts.watchExts)
	opts.watchPatterns = strings.TrimSpace(opts.watchPatterns)
	opts.watchSkipPatterns = strings.TrimSpace(opts.watchSkipPatterns)

	if !opts.build && !opts.vet && !opts.test && !opts.cover {
		opts.vet = true
		opts.test = true
		opts.build = true
	}

	mainPackages, err := packages()
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	pkgs := flag.Args()
	if len(pkgs) == 0 {
		pkgs = mainPackages
	} else {
		for i, pkg := range pkgs {
			// We only want to replace packages that are relative paths
			if !strings.HasPrefix(pkg, "./") {
				continue
			}

			// If any of the packages just contain ./... then we want to build
			// all packages anyway, so we break out here
			if pkg == "./..." {
				pkgs = mainPackages

				break
			}

			pkg = strings.TrimPrefix(pkg, "./")
			pkg = strings.TrimSuffix(pkg, "/...")
			pkg = strings.TrimSuffix(pkg, "/")

			for _, mainPkg := range mainPackages {
				if strings.Contains(mainPkg, pkg) {
					pkgs[i] = mainPkg
				}
			}
		}
	}

	if opts.goos == "" {
		opts.goos = runtime.GOOS
	}

	if opts.goarch == "" {
		opts.goarch = runtime.GOARCH
	}

	// Always immediately run the build pipeline at least once, even if in watch mode
	run(pkgs)

	if opts.watch {
		fmt.Println("-> watching for changes...")

		skipPatterns := strings.Fields(opts.watchSkipPatterns)
		watchPatterns := strings.Fields(opts.watchPatterns)
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
		for _, ext := range strings.Fields(opts.watchExts) {
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
				run(pkgs)

				fmt.Println("-> watching for changes...")
			}

			time.Sleep(opts.watchInterval)
		}
	}
}

func run(pkgs []string) {
	lastRun = time.Now()

	if opts.clear {
		clear()
	}

	fmt.Printf("-> running build:\n")

	kill()
	hook("before")

	if opts.vet {
		if err := vet(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	// If cover is enabled then we skip this because cover includes a call to
	// test with cover profile flags anyway
	if opts.test && !opts.cover {
		if err := test(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	if opts.cover {
		if err := cover(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	if opts.build {
		for _, pkg := range pkgs {
			if err := build(pkg); err != nil {
				if !opts.watch {
					os.Exit(1)
				}

				continue
			}
		}
	}

	hook("after")
}

func clear() {
	fmt.Print("\033c")
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

func packages() ([]string, error) {
	out, err := exec.Command("go", "list", "-f", "[{{ .Name }}]{{ .ImportPath }}", "./...").CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return nil, err
	}

	var packages []string
	for _, pkg := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(pkg, "[main]") {
			continue
		}

		packages = append(packages, strings.TrimPrefix(pkg, "[main]"))
	}

	return packages, nil
}

func generate() error {
	program, args, message := command("go", "generate", "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go generate... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func test() error {
	program, args, message := command("go", "test", "-race", "-vet", "off", "-tags", opts.testTags, "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go test... ", prefix)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(program, args...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("error")
	} else {
		fmt.Println("ok")
	}

	if err != nil || opts.verbose {
		if stdout.Len() > 0 {
			out := stdout.String()

			if err != nil && !opts.verbose {
				var filtered []string

				lines := strings.Split(out, "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok") {
						continue
					}

					filtered = append(filtered, line)
				}

				out = strings.Join(filtered, "\n")
			}

			fmt.Println(out)
		}

		if stderr.Len() > 0 {
			fmt.Println(stderr.String())
		}
	}

	return err
}

func cover() error {
	program, args, message := command("go", "test", "-race", "-tags", opts.testTags, "-coverpkg", "./...", "-coverprofile", ".cover.out", "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go test (cover)... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	if !opts.watch {
		program, args, message := command("go", "tool", "cover", "-html", ".cover.out")
		prefix := "->"
		if opts.verbose {
			prefix = "  "
			fmt.Printf("-> %v\n", message)
		}

		fmt.Printf("%v go tool cover... ", prefix)

		if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
			fmt.Println("error")

			if len(out) > 0 {
				fmt.Println(string(out))
			}

			return err
		} else {
			fmt.Println("ok")

			if len(out) > 0 && opts.verbose {
				fmt.Println(string(out))
			}
		}
	}

	return nil
}

func vet() error {
	program, args, message := command("go", "vet", "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go vet... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func build(pkg string) error {
	parts := strings.Split(pkg, "/")
	binaryName := parts[len(parts)-1]
	if opts.goos == "windows" {
		binaryName += ".exe"
	}

	tagsMessage := opts.tags
	if tagsMessage == "" {
		tagsMessage = "-"
	}

	args := []string{"build", "-o", ".", "-tags", opts.tags}
	gcflags := []string{}
	ldflags := []string{
		fmt.Sprintf("-X 'main.branch=%v'", commitBranch("")),
		fmt.Sprintf("-X 'main.version=%v'", closestTag("")),
		fmt.Sprintf("-X 'main.commit=%v'", commitHash("")),
		fmt.Sprintf("-X 'main.tags=%v'", tagsMessage),
	}

	if opts.verbose {
		args = append(args, "-v", "-x")
	}

	if opts.debug || opts.unoptimised {
		// -N disables all optimisations
		// -l disables inlining
		// See: go tool compile --help
		gcflags = append(gcflags, "all=-N -l")
	}

	if opts.debug {
		if opts.goos == "windows" {
			// Required on Windows to view disassembly in things like pprof
			args = append(args, "-buildmode", "exe")
		}

		ldflags = append(ldflags, "-X 'main.target=Debug'")
	} else {
		args = append(args, "-trimpath")

		ldflags = append(ldflags, "-X 'main.target=Release'")

		// -s disables the symbol table
		// -w disables DWARF generation
		// See: go tool link --help
		ldflags = append(ldflags, "-s")
		ldflags = append(ldflags, "-w")
	}

	if opts.race {
		args = append(args, "-race")

		ldflags = append(ldflags, "-X 'main.race=Enabled'")
	} else {
		ldflags = append(ldflags, "-X 'main.race=Disabled'")
	}

	if len(gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(gcflags, " "))
	}

	if len(ldflags) > 0 {
		args = append(args, "-ldflags", strings.Join(ldflags, " "))
	}

	args = append(args, pkg)

	var env []string
	if opts.goos != "" {
		env = append(env, "GOOS="+opts.goos)
	}
	if opts.goarch != "" {
		env = append(env, "GOARCH="+opts.goarch)
	}

	program, args, message := command("go", args...)
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go build %v... ", prefix, strings.TrimSuffix(pkg, "..."))

	cmd := exec.Command(program, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Print("ok ")

		var info []string

		if opts.debug {
			info = append(info, "debug")
		} else {
			info = append(info, "release")
		}

		if opts.race {
			info = append(info, "race")
		}

		fmt.Printf("(%v)\n", strings.Join(info, "/"))

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func kill() {
	for i, cmd := range programs {
		switch runtime.GOOS {
		case "windows":
			pid := strconv.Itoa(cmd.Process.Pid)
			out, err := exec.Command("tasklist", "/fi", "pid eq "+pid).CombinedOutput()
			if err != nil {
				fmt.Println(err)
			}

			if processRunning := !strings.Contains(strings.ToLower(string(out)), "no tasks"); !processRunning {
				continue
			}

			fmt.Printf("-> killing process #%v...\n", i)

			if err := cmd.Process.Kill(); err != nil {
				fmt.Println(err)
				fmt.Printf("-> forcibly killing process #%v...\n", i)

				if err := exec.Command("taskkill", "/pid", pid, "/f", "/t").Run(); err != nil {
					fmt.Println(err)

					continue
				}
			}

		default:
			fmt.Printf("-> interrupting process #%v...\n", i)

			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				fmt.Println(err)
				fmt.Printf("-> killing process #%v...\n", i)

				if err := cmd.Process.Kill(); err != nil {
					fmt.Println(err)
				}

				continue
			}

			if _, err := cmd.Process.Wait(); err != nil {
				fmt.Println(err)
				fmt.Printf("-> killing process #%v...\n", i)

				if err := cmd.Process.Kill(); err != nil {
					fmt.Println(err)
				}

				continue
			}
		}
	}

	programs = nil
}

func hook(name string) {
	var cmdStrs flagStrs
	switch name {
	case "before":
		cmdStrs = opts.before

	case "after":
		cmdStrs = opts.after

	default:
		panic(fmt.Sprintf("invalid hook name %q", name))
	}

	// Rather than writing a parser for nested command line args we use this
	// regular expression
	// It should be fine for most use cases where it matches:
	// - Escaped double quotes:  "(\\"|[^"])+"
	// - Space separated values: [^\s\\]+
	// - Escaped spaces:         (\\+\s[^\s\\]+)*
	re := regexp.MustCompile(`"(\\"|[^"])+"|[^\s\\]+(\\+\s[^\s\\]+)*`)

	for i, cmdStr := range cmdStrs {
		fields := re.FindAllString(cmdStr, -1)
		for i := range fields {
			fields[i] = strings.ReplaceAll(fields[i], `\ `, " ")
			fields[i] = strings.ReplaceAll(fields[i], `\"`, `"`)
			fields[i] = strings.ReplaceAll(fields[i], `\\`, `\`)
		}

		program, args, message := command(fields[0], fields[1:]...)

		fmt.Printf("-> build command #%v... %v\n", i, message)

		cmd := exec.Command(program, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		programs = append(programs, cmd)

		if err := cmd.Start(); err != nil {
			fmt.Println(err)

			continue
		}
	}
}

func closestTag(dir string) string {
	git, err := exec.LookPath("git")
	if err != nil {
		return "unavailable"
	}

	cmd := exec.Command(git, "describe", "--long", "--tags")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	parts := strings.Split(string(out), "-")
	tag := strings.Join(parts[:len(parts)-2], "-")
	commitsAhead, _ := strconv.Atoi(parts[len(parts)-2])
	commitHash := tagCommitHash(dir, tag)
	version := strings.TrimPrefix(strings.TrimSpace(tag), "v")

	var additional []string
	if commitsAhead > 0 {
		noun := "commit"
		if commitsAhead > 1 {
			noun = "commits"
		}

		additional = append(additional, fmt.Sprintf("%v %v ahead of %v", commitsAhead, noun, commitHash))
	}

	if hasUncommittedChanges(dir) {
		additional = append(additional, "built with uncommitted changes")
	}

	if len(additional) > 0 {
		version = fmt.Sprintf("%v (%v)", version, strings.Join(additional, "; "))
	}

	return version
}

func tagCommitHash(dir, tag string) string {
	git, err := exec.LookPath("git")
	if err != nil {
		return "unavailable"
	}

	cmd := exec.Command(git, "show-ref", "-d", "--tags", tag)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasSuffix(line, "^{}") {
			parts := strings.Split(line, " ")

			return parts[0][:7]
		}
	}

	return "unknown"
}

func hasUncommittedChanges(dir string) bool {
	git, err := exec.LookPath("git")
	if err != nil {
		return false
	}

	cmd := exec.Command(git, "status", "-su")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return len(out) > 0
}

func commitBranch(dir string) string {
	git, err := exec.LookPath("git")
	if err != nil {
		return "unavailable"
	}

	cmd := exec.Command(git, "branch", "--show-current")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(out))
}

func commitHash(dir string) string {
	git, err := exec.LookPath("git")
	if err != nil {
		return "unavailable"
	}

	cmd := exec.Command(git, "rev-list", "-1", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(out))
}
