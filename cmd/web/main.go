package main

import (
	"context"
	"errors"
	"expvar"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/internal/size"
	"github.com/polyscone/tofu/web"
	"github.com/polyscone/tofu/web/handler"
)

// Set through ldflags at build time
var target string

var opts struct {
	version  bool
	dev      bool
	data     string
	basePath string

	log struct {
		style LogStyle
	}

	server struct {
		addr        Addr
		insecure    bool
		ipWhitelist IPWhitelist
		proxies     Proxies
	}

	debug struct {
		addr Addr
	}

	password struct {
		duration    time.Duration
		memory      int
		parallelism int
	}
}

func main() {
	defer func() {
		// If we panic we want to log it using whatever handler was
		// setup as the default in the slog package, rather than
		// just having the stack trace dumped without any structure
		if err := recover(); err != nil {
			const size = 64 << 10

			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]

			message := fmt.Errorf("%v\n%s", err, buf)

			slog.Error("panic", "error", message)

			os.Exit(1)
		}
	}()

	res := run()

	os.Exit(res)
}

func run() int {
	var config handler.Config
	requiredFlags := []string{"addr"}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %v:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %v [command] [-dev] [-addr <addr>] [-log-style <text|json|dev>]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Commands:")
		fmt.Fprintf(flag.CommandLine.Output(), "  version\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tDisplay binary version information\n")
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.StringVar(&opts.data, "data", "./.data", "The directory to use for storing application data")
	flag.StringVar(&opts.basePath, "base-path", "", "A prefix path to add to all routes")
	flag.BoolVar(&opts.dev, "dev", false, "Whether to run in development mode")
	flag.BoolVar(&opts.version, "version", false, "Display binary version information")

	flag.Var(&opts.log.style, "log-style", "The output style for log messages (text|json|dev)")

	flag.Var(&opts.server.addr, "addr", "The address to run the server on, for example :8080; random if empty")
	flag.BoolVar(&opts.server.insecure, "insecure", false, "Run in insecure mode without HTTPS")
	flag.Var(&opts.server.ipWhitelist, "ip-whitelist", "A space separated list of whitelisted ip addresses")
	flag.Var(&opts.server.proxies, "trusted-proxies", "A space separated list of trusted proxy addresses")

	flag.Var(&opts.debug.addr, "debug-addr", "The address to run the private debug server on, for example :8081; random if empty")

	flag.DurationVar(&opts.password.duration, "password-hash-duration", 1*time.Second, "The target duration of a password hash")
	flag.IntVar(&opts.password.memory, "password-hash-memory", 64*size.Kibibyte, "The amount of memory (KiB) to use when hashing a password")
	flag.IntVar(&opts.password.parallelism, "password-hash-parallelism", max(1, runtime.NumCPU()/2), "The number of threads to use when hashing a password")

	flag.Float64Var(&config.Site.RateLimit.Capacity, "site-ratelimit-cap", 50, "The token bucket capacity for the site rate limiter")
	flag.Float64Var(&config.Site.RateLimit.Replenish, "site-ratelimit-rep", 1, "The number of tokens to replenish every second for the site rate limiter")

	flag.Float64Var(&config.PWA.RateLimit.Capacity, "pwa-ratelimit-cap", 50, "The token bucket capacity for the PWA rate limiter")
	flag.Float64Var(&config.PWA.RateLimit.Replenish, "pwa-ratelimit-rep", 1, "The number of tokens to replenish every second for the PWA rate limiter")

	flag.Float64Var(&config.APIv1.RateLimit.Capacity, "api-v1-ratelimit-cap", 50, "The token bucket capacity for the v1 API rate limiter")
	flag.Float64Var(&config.APIv1.RateLimit.Replenish, "api-v1-ratelimit-rep", 1, "The number of tokens to replenish every second for the v1 API rate limiter")

	flag.Parse()

	if flag.NArg() > 0 && flag.Arg(0) != "version" {
		fmt.Fprintf(flag.CommandLine.Output(), "Unknown command %q\n", flag.Arg(0))
		flag.Usage()

		return 2
	}

	// Version data
	{
		modified := false
		revision := "-"
		tags := "-"
		_go := strings.TrimPrefix(runtime.Version(), "go")
		race := "disabled"

		info, _ := debug.ReadBuildInfo()
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value

			case "vcs.modified":
				modified = setting.Value == "true"

			case "-tags":
				tags = strings.ReplaceAll(setting.Value, ",", " ")

			case "-race":
				if setting.Value == "true" {
					race = "enabled"
				}
			}
		}

		if modified {
			revision += " (uncommitted changes)"
		}

		if target == "" {
			target = "-"
		}

		if opts.version || flag.Arg(0) == "version" {
			var version string

			version += fmt.Sprintln("Target:       ", target)
			version += fmt.Sprintln("Revision:     ", revision)
			version += fmt.Sprintln("Tags:         ", tags)
			version += fmt.Sprintln("Go version:   ", _go)
			version += fmt.Sprintln("OS/Arch:      ", runtime.GOOS+"/"+runtime.GOARCH)
			version += fmt.Sprintln("Race detector:", race)

			fmt.Print(version)

			return 0
		}

		newString := func(value string) expvar.Var {
			var s expvar.String

			s.Set(value)

			return &s
		}

		version := expvar.NewMap("version")

		version.Set("target", newString(target))
		version.Set("revision", newString(revision))
		version.Set("tags", newString(tags))
		version.Set("go", newString(_go))
		version.Set("os", newString(runtime.GOOS))
		version.Set("arch", newString(runtime.GOARCH))
		version.Set("race", newString(race))
	}

	now := time.Now()
	expvar.Publish("uptime", expvar.Func(func() any {
		return time.Since(now)
	}))

	expvar.Publish("now", expvar.Func(func() any {
		return time.Now()
	}))

	expvar.Publish("cgoCalls", expvar.Func(func() any {
		return runtime.NumCgoCall()
	}))

	expvar.Publish("cpus", expvar.Func(func() any {
		return runtime.NumCPU()
	}))

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	if opts.log.style == "" {
		opts.log.style = "json"
	}

	logHandler, err := opts.log.style.NewHandler(nil)
	if err != nil {
		fmt.Println(err)

		return 2
	}

	logger, err := opts.log.style.NewLogger(nil)
	if err != nil {
		fmt.Println(err)

		return 2
	}

	slog.SetDefault(logger)

	opts.server.addr.insecure = opts.server.insecure
	opts.debug.addr.insecure = true

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
	if len(requiredMessages) > 0 {
		fmt.Println(strings.Join(requiredMessages, "\n"))
		flag.Usage()

		return 2
	}

	if opts.basePath != "" {
		app.BasePath = opts.basePath
	}

	if err := os.MkdirAll(opts.data, 0755); err != nil {
		slog.Error("make data directory", "error", err)

		return 1
	}

	if len(opts.server.ipWhitelist) > 0 {
		// We always want to implicitly whitelist localhost as an IP
		if ip := "::1"; !slices.Contains(opts.server.ipWhitelist, ip) {
			opts.server.ipWhitelist = append(opts.server.ipWhitelist, ip)
		}
		if ip := "127.0.0.1"; !slices.Contains(opts.server.ipWhitelist, ip) {
			opts.server.ipWhitelist = append(opts.server.ipWhitelist, ip)
		}
	}

	// We always want to implicitly trust localhost as a proxy
	if ip := "::1"; !slices.Contains(opts.server.proxies, ip) {
		opts.server.proxies = append(opts.server.proxies, ip)
	}
	if ip := "127.0.0.1"; !slices.Contains(opts.server.proxies, ip) {
		opts.server.proxies = append(opts.server.proxies, ip)
	}

	if err := initHasher(); err != nil {
		slog.Error("initialize hasher", "error", err)

		return 1
	}

	tenants := filepath.Join(opts.data, "tenants.json")
	if err := initTenants(tenants); err != nil {
		slog.Error("initialize tenants", "error", err)
	}

	listener, err := opts.server.addr.Listener()
	if err != nil {
		slog.Error("get listener", "error", err)

		return 1
	}

	baseCtx, baseCtxCancel := context.WithCancel(context.Background())
	spill := 500 * time.Millisecond
	readHeaderTimeout := 5 * time.Second
	srv := http.Server{
		ErrorLog:          slog.NewLogLogger(logHandler, slog.LevelError),
		Addr:              opts.server.addr.value,
		IdleTimeout:       1 * time.Minute,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       web.HandlerTimeout + readHeaderTimeout + spill,
		WriteTimeout:      web.HandlerTimeout + spill,
		Handler:           web.NewMultiTenantHandler(logger, newTenant, config),
		BaseContext: func(_ net.Listener) context.Context {
			return baseCtx
		},
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

	if opts.debug.addr.value != "" {
		go func() {
			slog.Info("listening (debug)", "addr", opts.debug.addr.String(), "pid", os.Getpid())

			mux := http.NewServeMux()

			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			mux.Handle("/debug/vars", expvar.Handler())

			if err := http.ListenAndServe(opts.debug.addr.value, mux); err != nil {
				slog.Error("serve over HTTP (debug)", "error", err)
			}
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	caught := <-stop
	signal.Stop(stop)

	slog.Info("shutting down", "signal", caught.String())

	shutdownTimeout := 5 * time.Second
	go func() {
		time.Sleep(shutdownTimeout / 2)

		baseCtxCancel()
	}()

	ctxShutdown, ctxShutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer ctxShutdownCancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		slog.Error("shut down", "error", err)
	}

	background.Wait()

	closeCache()

	return 0
}
