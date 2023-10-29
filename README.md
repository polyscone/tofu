# Readme

## Building

Running `make` will invoke the default `build` rule which will build all binaries using `./...`.

The `PKG` variable is used in any command where a package path is expected.
You can override the package to build/test/vet/etc. with `PKG=./cmd/foo`.

The `OUT` variable controls the name of the resulting binary. By default it's set to `.` so that all binaries built with `./...` get names automatically based on their directory names.

Build tags are controlled with the `TAGS` variable.

Setting the `RACE` variable to anything will build with `-race`.

Setting the `DEBUG` variable to anything will build with optimisations and inlining disabled. Otherwise builds are done in "release" mode where paths are trimmed, and the symbol table and DWARF generation is disabled.

The `OPTIMISATIONS` variable controls the package pattern to use when printing compiler optimisation decisions. For example, `make OPTIMISATIONS=./internal/foo` will build with `-gcflags "./internal/foo=-m"`.

The `CHECK_BCE` variable controls the package pattern to use when printing compiler bounds check elimination decisions. For example, `make CHECK_BCE=./internal/foo` will build with `-gcflags "./internal/foo=-d=ssa/check_bce"`.

The benchmark count can be set via the `BENCH_COUNT` variable.

If any binaries expect to be a Windows GUI without a terminal/console attached then setting the `WINDOWSGUI` variable to anything will build with `-ldflags "-H windowsgui"`.

For all available `make` rules check the `Makefile` in the root of the project.

## The web application adapter

The `web` binary built from the `cmd/web` directory is an HTTP web adapter for the main application. It implements an MPA style web interface, SPA style PWA, and a simple JSON API that both the MPA and SPA can interact with.

### Web application tenants

The web adapter implements a multi-tenant application server, where tenants are controlled through a configuration file. A tenant represents a single application that may be used by any number of hostnames.

Tenant configuration is stored in the data directory's `tenants.json` file in the following format:

```json
{
	"app1": {
		"hostnames": {
			"foo.com": "site",
			"app.foo.com": "pwa"
		}
	},
	"app2": {
		"hostnames": {
			"bar.com": "site",
			"app.bar.com": "pwa"
		}
	}
}
```

The top-level keys (e.g. `"app1"`) describe a unique tenant name. This can be anything you like and is used to group things like hostname configuration for a tenant. It's also used as the folder name that will be created in the data directory for that tenant.

The `"hostnames"` key within a tenant object is used to describe all of the hostnames that should resolve to the parent tenant. Hostnames are described as key/value pairs where the key is the hostname and the value is the type of application the hostname serves.

The application type (e.g. `"site"`, `"pwa"`) is used during tenant setup at runtime to configure routers etc. The `"site"` tenant will setup routes specific to a more traditional MPA, whereas `"pwa"` will setup routes that make sense for an SPA style PWA.

```go
switch tenant.Kind {
case "site":
	mux.Handle("/", NewSiteRouter(h))

case "pwa":
	mux.Handle("/", NewPWARouter(h))
}
```

Any number of hostnames can be associated with a tenant.

It's important to remember that each hostname associated with a tenant will use the same repositories and other shared data structures internally. What this means is that JSON API calls from PWA hostnames will work with the same data as the main web application that site hostnames have access to. This has the nice side effect that when you write APIs that resolve to the same tenant as other hostnames, you don't need to deal with CORS settings, since you can just call the API endpoints on the application's own hostname and know that it's working with the same data internally.

Currently, the `tenants.json` file is only read once at application startup, so any changes require a restart of the whole application.

### Running the web application locally

Running locally requires a basic `tenants.json` file to be setup that can resolve any local development domains you want to use to any number of tenants you'd like to serve.

A simple starting point for a single tenant using `localhost` for the main MPA style web application, and a custom `app.local.com` domain for the SPA style PWA would be:

```json
{
	"app1": {
		"hostnames": {
			"localhost": "site",
			"app.local.com": "pwa"
		}
	}
}
```

On the first request this will create a folder in the data directory called `app1` where all of the tenant's data will be stored.

By default the application will use secure settings; cookies, for example, will have the `secure` flag set on them.

If you want to run locally with insecure HTTP, rather than HTTPS, you'll need to use the `-insecure` flag when running the web binary.

If you'd like to run locally with HTTPS then you'll need `cert.pem` and `key.pem` files to be available in the data directory. You can do this simply by running the following command after navigating to the data directory, changing the path and comma separated `-host` flag value to match your hostnames:

```sh
go run <go-install-path>/src/crypto/tls/generate_cert.go -rsa-bits=2048 -host=localhost,app.local.com
```

If you have an HTTP server like Caddy setup as a reverse proxy, then you might need to keep things like secure cookie flags, but allow an insecure HTTP connection. In that case you can use the `-behind-secure-proxy` flag instead. This will allow an insecure HTTP connection for the reverse proxy, but allow the main application to stick with the secure defaults internally, like secure cookies.

Setting the `-log-style` flag to `dev` will enable more readable log output than the default JSON style.

Choosing an address for the `-debug-addr` flag will enable Go's built-in debug endpoints. These endpoints do not use the standard library's default serve mux, and run on a separate serve mux from the main application to avoid accidentally exposing the endpoints publicly.

Finally you should pass the `-dev` flag, which will do things like disabling HTML template caching.

There is also a target in the `Makefile` that will set all of these flags for you; you can run that with `make run/web`. If you are running an HTTP server that's setup to reverse proxy to the application then you can run `make run/web BEHIND_SECURE_PROXY=1` instead.

### Password hashing parameters

Password hashing parameters are detected for the hardware you're running the application on to reach a target duration of 1 second hashing time.

When you first run the application it will detect the correct parameters for your hardware and cache the results in the `argon2_params.json` file in the data directory.

If you need to detect new password hashing parameters, due an upgrade in hardware or a move to another machine, deleting the `argon2_params.json` file will trigger detection again the next time you start the application.

### Proxies and rate limiting

The main web adapter application implements a simple leaky bucket style rate limiter which is based on IP addresses. Since IP addresses are used to keep track of the number of remaining tokens this means you'll need to tell the application about any trusted proxy IP addresses that may show up in a request.

By default the IP addresses `::1` and `127.0.0.1` are always implicitly trusted, so these never need to be defined, but if you know that you'll need to ignore certain other proxy IP addresses you can do that by passing a space separated list to the application through the `-trusted-proxies` flag.

Doing this will allow the rate limiter middleware to skip past those IPs when looking for the real IP to use for tracking the token count.
