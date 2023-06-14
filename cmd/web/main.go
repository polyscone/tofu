package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"golang.org/x/exp/slog"
)

// Build information is set at compile time using the `-X` ldflags.
var (
	version = "-"
	branch  = "-"
	commit  = "-"
	tags    = "-"
	target  = "-"
	race    = "-"
)

var opts struct {
	version bool
	dev     bool
	data    string

	log struct {
		style LoggerStyle
	}

	server struct {
		addr         Addr
		insecure     bool
		insecureHTTP bool
		proxies      Proxies
	}
}

func main() {
	requiredFlags := []string{"addr"}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %v:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %v [command] [-dev] [-addr <addr>] [-log-style <text|json|pretty>]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Commands:")
		fmt.Fprintf(flag.CommandLine.Output(), "  version\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tDisplay binary version information\n")
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.StringVar(&opts.data, "data", "./.data", "The directory to use for storing application data")
	flag.BoolVar(&opts.dev, "dev", false, "Whether to run in development mode")
	flag.BoolVar(&opts.version, "version", false, "Display binary version information")
	flag.Var(&opts.log.style, "log-style", "The output style for log messages (text|json|pretty)")
	flag.Var(&opts.server.addr, "addr", "The address to run the build server on, for example :8080; random if empty")
	flag.BoolVar(&opts.server.insecure, "insecure", false, "Run in insecure mode without HTTPS")
	flag.BoolVar(&opts.server.insecureHTTP, "insecure-http", false, "Run in secure mode but without HTTPS")
	flag.Var(&opts.server.proxies, "trusted-proxies", "A space separated list of trusted proxy addresses")
	flag.Parse()

	if flag.NArg() != 0 && flag.Arg(0) != "version" {
		fmt.Fprintf(flag.CommandLine.Output(), "Unknown command %q\n", flag.Arg(0))
		flag.Usage()

		os.Exit(2)
	}

	if opts.version || flag.Arg(0) == "version" {
		var info string

		info += fmt.Sprintln("Version:      ", version)
		info += fmt.Sprintln("Branch:       ", branch)
		info += fmt.Sprintln("Commit:       ", commit)
		info += fmt.Sprintln("Tags:         ", tags)
		info += fmt.Sprintln("Go version:   ", strings.TrimPrefix(runtime.Version(), "go"))
		info += fmt.Sprintln("OS/Arch:      ", runtime.GOOS+"/"+runtime.GOARCH)
		info += fmt.Sprintln("Target:       ", target)
		info += fmt.Sprintln("Race detector:", race)

		fmt.Print(info)

		return
	}

	if opts.server.insecure {
		opts.server.insecureHTTP = true
	}

	if opts.log.style == "" {
		opts.log.style = styleJSON
	}

	var level slog.LevelVar
	var handler slog.Handler
	switch opts.log.style {
	case styleJSON:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &level})

	case styleText:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: &level})

	case stylePretty:
		handler = NewPrettyHandler(os.Stdout, &level)

	default:
		fmt.Printf("Unknown log style %q", opts.log.style)

		os.Exit(2)
	}

	slog.SetDefault(slog.New(handler))

	opts.server.addr.insecure = opts.server.insecureHTTP

	// Required flag checks
	var requiredMessages []string
	for _, name := range requiredFlags {
		fail := func(name, value string, err error) {
			// If err is nil here then it's most likely because we forced a call
			// to fail on a required field that was left empty, but there was no
			// custom error message, so we just set it to a default one in that case
			if err == nil {
				err = errors.New("required flag")
			}

			message := fmt.Sprintf("invalid value %q for flag -%v: %v", value, name, err)
			requiredMessages = append(requiredMessages, message)
		}

		// If the provided value is an empty string then we force a call to the
		// flag value's Set() method to get any custom error messages
		// Since flags here are required this is ok because we expect Set() to
		// be called at least once per flag anyway
		// These checks will always fail with flags set using flag.Func
		if f := flag.Lookup(name); f.Value.String() == "" {
			fail(name, f.Value.String(), f.Value.Set(f.Value.String()))
		}
	}
	if len(requiredMessages) != 0 {
		fmt.Println(strings.Join(requiredMessages, "\n"))
		flag.Usage()

		os.Exit(2)
	}

	if err := os.MkdirAll(opts.data, 0755); err != nil {
		slog.Error("make data directory", "error", err)

		os.Exit(1)
	}

	if err := initHasher(); err != nil {
		slog.Error("initialise hasher", "error", err)

		os.Exit(1)
	}

	tenants := filepath.Join(opts.data, "tenants.json")
	if err := initTenants(tenants); err != nil {
		slog.Error("initialise tenants", "error", err)
	}

	httputil.TrustedProxies = opts.server.proxies

	listener, err := opts.server.addr.Listener()
	if err != nil {
		slog.Error("get listener", "error", err)

		os.Exit(1)
	}

	srv := http.Server{
		ErrorLog:     slog.NewLogLogger(handler, slog.LevelError),
		Addr:         opts.server.addr.value,
		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      web.NewMultiTenantHandler(newTenant),
	}

	go func() {
		slog.Info("listening", "addr", opts.server.addr.String(), "pid", os.Getpid())

		if opts.server.addr.insecure {
			err := srv.Serve(listener)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("serve over HTTP", "error", err)
			}
		} else {
			cert := filepath.Join(opts.data, "cert.pem")
			key := filepath.Join(opts.data, "key.pem")

			err := srv.ServeTLS(listener, cert, key)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("serve over HTTPS", "error", err)
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	caught := <-stop
	signal.Stop(stop)

	slog.Info("shutting down", "signal", caught.String())

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		slog.Error("shut down", "error", err)
	}

	databases.mu.Lock()
	defer databases.mu.Unlock()

	for alias, db := range databases.data {
		if err := db.Close(); err != nil {
			slog.Error("close database connection", "alias", alias, "error", err)
		}
	}
}
