//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode"
)

var tags = struct {
	build string
	test  string
}{
	build: "",
	test:  "",
}

var opts struct {
	cmd         string
	goos        string
	goarch      string
	tags        string
	testTags    string
	unoptimised bool
	debug       bool
	race        bool
	verbose     bool
}

func main() {
	flag.StringVar(&opts.cmd, "cmd", "build", "Sets the command to run <build|generate|vet|test|cover>")
	flag.StringVar(&opts.goos, "goos", "", "Sets the GOOS environment variable for the build")
	flag.StringVar(&opts.goarch, "goarch", "", "Sets the GOARCH environment variable for the build")
	flag.StringVar(&opts.tags, "tags", "", "Additional build tags")
	flag.StringVar(&opts.testTags, "test-tags", "", "Additional test build tags")
	flag.BoolVar(&opts.unoptimised, "unoptimised", false, "Disable optimisations/inlining")
	flag.BoolVar(&opts.debug, "debug", false, "Enable symbol table/DWARF generation and disable optimisations/inlining")
	flag.BoolVar(&opts.race, "race", false, "Enable data race detection in the final binary")
	flag.BoolVar(&opts.verbose, "verbose", false, "Print the commands that are being run along with all command output")
	flag.Parse()

	if s := strings.TrimSpace(opts.tags); s != "" {
		tags.build += " " + s
	}
	tags.build = strings.TrimSpace(tags.build)

	if s := strings.TrimSpace(opts.testTags); s != "" {
		tags.test += " " + s
	}
	tags.test = strings.TrimSpace(tags.build + " " + tags.test)

	mainPackages, err := packages()
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	pkgs := flag.Args()
	if len(pkgs) == 0 {
		pkgs = mainPackages
	}

	if opts.goos == "" {
		opts.goos = runtime.GOOS
	}

	if opts.goarch == "" {
		opts.goarch = runtime.GOARCH
	}

	switch opts.cmd {
	case "build":
		for _, pkg := range pkgs {
			if err := build(pkg); err != nil {
				os.Exit(1)
			}
		}

	case "generate":
		if err := generate(); err != nil {
			os.Exit(1)
		}

	case "vet":
		if err := vet(); err != nil {
			os.Exit(1)
		}

	case "test":
		if err := test(""); err != nil {
			os.Exit(1)
		}

	case "cover":
		if err := cover(); err != nil {
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command %q, please see help for details\n", opts.cmd)
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
	if opts.verbose {
		fmt.Printf("-> %v ... ", message)
	} else {
		fmt.Print("-> go generate ... ")
	}

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

func test(coverfile string) error {
	args := []string{"test", "-vet", "off", "-race"}

	if coverfile != "" {
		args = append(args, "-coverprofile", coverfile)
	}

	if len(tags.test) > 0 {
		args = append(args, "-tags", tags.test)
	}

	args = append(args, "./...")

	program, args, message := command("go", args...)
	if opts.verbose {
		fmt.Printf("-> %v ... ", message)
	} else {
		fmt.Print("-> go test ... ")
	}

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			if !opts.verbose {
				var filtered []string
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok") {
						continue
					}
					if coverfile != "" && strings.Contains(line, "coverage:") {
						continue
					}

					filtered = append(filtered, line)
				}

				out = []byte(strings.Join(filtered, "\n"))
			}

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

func cover() error {
	const coverfile = "_cover.out"
	if err := test(coverfile); err != nil {
		return err
	}

	program, args, message := command("go", "tool", "cover", "-html", coverfile)
	if opts.verbose {
		fmt.Printf("-> %v ... ", message)
	} else {
		fmt.Print("-> go tool cover ... ")
	}

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

func vet() error {
	args := []string{"vet"}

	if len(tags.test) > 0 {
		args = append(args, "-tags", tags.test)
	}

	args = append(args, "./...")

	program, args, message := command("go", args...)
	if opts.verbose {
		fmt.Printf("-> %v ... ", message)
	} else {
		fmt.Print("-> go vet ... ")
	}

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
	var gcflags []string
	var ldflags []string
	args := []string{"build", "-o", "."}

	if len(tags.build) > 0 {
		args = append(args, "-tags", tags.build)
	}

	if opts.debug || opts.unoptimised {
		// -N disables all optimisations
		// -l disables inlining
		// See: go tool compile --help
		gcflags = append(gcflags, "all=-N -l")
	}

	if opts.debug {
		if opts.goos == "windows" {
			// This is required on Windows to view disassembly in things like pprof
			args = append(args, "-buildmode", "exe")
		}

		ldflags = append(ldflags, "-X 'main.target=Debug'")
	} else {
		args = append(args, "-trimpath")

		// -s disables the symbol table
		// -w disables DWARF generation
		// See: go tool link --help
		ldflags = append(ldflags, "-s")
		ldflags = append(ldflags, "-w")

		ldflags = append(ldflags, "-X 'main.target=Release'")
	}

	if opts.race {
		args = append(args, "-race")
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
	if opts.verbose {
		fmt.Printf("-> %v ... ", message)
	} else {
		fmt.Printf("-> go build %v ... ", pkg)
	}

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
