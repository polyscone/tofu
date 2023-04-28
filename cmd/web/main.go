package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
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
	secret  string

	log struct {
		style logger.Style
	}

	server struct {
		addr         Addr
		insecure     bool
		insecureHTTP bool
		proxies      string
	}
}

func main() {
	requiredFlags := []string{"secret", "addr"}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %v:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %v [command] [-dev] [-addr <addr>] [-log-style <text|json>]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Commands:")
		fmt.Fprintf(flag.CommandLine.Output(), "  version\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tDisplay binary version information\n")
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.StringVar(&opts.data, "data", "./.tofu", "The directory to use for storing application data")
	flag.BoolVar(&opts.dev, "dev", false, "Whether to run in development mode")
	flag.BoolVar(&opts.version, "version", false, "Display binary version information")
	flag.Var(&opts.log.style, "log-style", "The output style for log messages (text|json)")
	flag.Var(&opts.server.addr, "addr", "The address to run the build server on, for example :8080; random if empty")
	flag.BoolVar(&opts.server.insecure, "insecure", false, "Run in insecure mode without HTTPS")
	flag.BoolVar(&opts.server.insecureHTTP, "insecure-http", false, "Run in secure mode but without HTTPS")
	flag.StringVar(&opts.server.proxies, "trusted-proxies", "", "A space separated list of trusted proxy addresses")
	flag.StringVar(&opts.secret, "secret", "", "The secret to use for things like encrypting/decrypting data")
	flag.Parse()

	if opts.server.insecure {
		opts.server.insecureHTTP = true
	}

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

	if opts.log.style == "" {
		opts.log.style = logger.JSON
	}

	infoLogger := logger.New(os.Stdout, opts.log.style)
	errorLogger := logger.New(os.Stderr, opts.log.style)

	log.SetFlags(0)
	log.SetOutput(infoLogger)

	logger.OutputStyle = opts.log.style

	logger.Info.SetOutput(infoLogger)
	logger.Error.SetOutput(errorLogger)

	opts.server.addr.Insecure = opts.server.insecureHTTP

	// Required flag checks
	for _, name := range requiredFlags {
		// This is a copy of the fail function in the flag package with
		// some small changes
		fail := func(name, value, err any) {
			// If err is nil here then it's most likely because we forced a call
			// to fail on a required field that was left empty, but there was no
			// custom error message, so we just set it to a default one in that case
			if err == nil {
				err = errors.Tracef("required flag")
			}

			fmt.Printf("invalid value %q for flag -%v: %v\n", value, name, err)
			flag.Usage()

			os.Exit(2)
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

	listener, err := opts.server.addr.Listener()
	if err != nil {
		logger.PrintError(err)

		os.Exit(1)
	}

	ctx := context.Background()

	dbFile := filepath.Join(opts.data, "tofu.db")
	db, err := sqlite.Open(ctx, sqlite.KindFile, dbFile)
	if err != nil {
		logger.PrintError(err)

		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.PrintError(errors.Tracef(err))
		}
	}()

	bus, broker, err := app.Compose(ctx, db, []byte(opts.secret))
	if err != nil {
		logger.PrintError(err)

		os.Exit(1)
	}

	sessions, err := web.NewSQLiteSessionRepo(ctx, db, 2*time.Hour)
	if err != nil {
		logger.PrintError(err)

		os.Exit(1)
	}

	tokens, err := web.NewSQLiteTokenRepo(ctx, db)
	if err != nil {
		logger.PrintError(err)

		os.Exit(1)
	}

	mailer, err := web.NewMailClient("localhost", 25)
	if err != nil {
		logger.PrintError(err)

		os.Exit(1)
	}

	proxies := strings.Fields(opts.server.proxies)

	handler := web.NewHandler(bus, broker, sessions, tokens, mailer, web.Options{
		Dev:      opts.dev,
		Insecure: opts.server.insecure,
		Proxies:  proxies,
	})

	srv := http.Server{
		Addr:         opts.server.addr.Value,
		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      handler,
	}

	go func() {
		logger.Info.Printf("listening on %v (pid %v)\n", opts.server.addr, os.Getpid())

		if opts.server.addr.Insecure {
			err := srv.Serve(listener)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.PrintError(errors.Tracef(err))
			}
		} else {
			cert := filepath.Join(opts.data, "cert.pem")
			key := filepath.Join(opts.data, "key.pem")

			err := srv.ServeTLS(listener, cert, key)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.PrintError(errors.Tracef(err))
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	caught := <-stop

	logger.Info.Printf("caught %v signal; shutting down\n", caught)

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.PrintError(errors.Tracef(err))
	}
}
