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

## Tenants

This project implements a multi-tenant application server, where tenants are controlled through a configuration file.

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
The top-level keys (e.g. `"app1"`) describe a unique tenant name. This can be anything you like and is used to group things like hostname configuration for a tenant.

The `"hostnames"` key within a tenant object is used to describe all of the hostnames that should resolve to the parent tenant. Hostnames are described as key/value pairs where the key is the hostname and the value is the type of application the hostname serves.

The application type (e.g. "site", "pwa") is used during tenant setup at runtime to configure routers etc.

Currently, the `tenants.json` file is only read once at application startup, so any changes require a restart of the whole application.
