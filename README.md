# Readme

## Building and running

Build with:
```sh
go run dev.go
```

Run locally with:
```sh
./web -dev -insecure -log-style dev -addr localhost:8080
```

This will start an insecure server over HTTP.

If you'd like to run locally with HTTPS then you'll need `cert.pem` and `key.pem` files to be available in the data directory. You can do this by running the following command after navigating to the data directory, changing the comma separated `-host` flag value to match the hosts you want to use:

```sh
go run $(go env GOROOT)/src/crypto/tls/generate_cert.go -rsa-bits 2048 -host "localhost,app.local.com"
```

On Windows replace `$(go env GOROOT)` with `%GOROOT%` if it's set, otherwise run `go env GOROOT` and copy the path into the command.

See `web -help` for more application options.

## Web tenant configuration

A tenant represents a single application that may be used by any number of hosts.

Tenant configuration is stored in the data directory's `tenants.json` file in the following format:

```json
{
	"app1": {
		"hosts": {
			"site": "foo.com",
			"pwa": "app.foo.com"
		},
		"aliases": {
			"site": ["alias.foo.com:8080"]
		}
	},
	"app2": {
		"hosts": {
			"site": "bar.com",
			"pwa": "app.bar.com"
		}
	}
}
```

The top-level keys (e.g. `"app1"`) describe a unique tenant name. This can be anything you like and is used to group things like host configuration for a tenant. It's also used as the folder name that will be created in the data directory for that tenant.

The `"hosts"` key within a tenant object is used to map a particular application type within the application to a host.

The application type (e.g. `"site"`, `"pwa"`) is used during tenant setup at runtime to configure routers etc. The `"site"` tenant will setup routes specific to a more traditional MPA, whereas `"pwa"` will setup routes that make sense for an SPA style PWA.

The `"aliases"` key is used to associate more hosts with an application type if needed and is completely optional.

Each tenant will share the same repositories and other data structures internally regardless of the host/alias. This means that a tenant accessed from any configured host/alias will use, for example, the same database.

## Password hashing parameters

Password hashing parameters are detected for the hardware you're running the application on to reach a target duration of 1 second hashing time.

When you first run the application it will detect the correct parameters for your hardware and cache the results in the `argon2_params.json` file in the data directory.

If you need to detect new password hashing parameters, due an upgrade in hardware or a move to another machine, deleting the `argon2_params.json` file will trigger detection again the next time you start the application.

## Proxies and rate limiting

The server uses a simple token bucket style rate limiter middleware which is based on IP addresses. Since IP addresses are used to keep track of the number of remaining tokens this means you'll need to tell the application about any trusted proxy IP addresses that may show up in a request.

By default the IP addresses `::1` and `127.0.0.1` are always implicitly trusted, so these never need to be defined, but if you know that you'll need to ignore certain other proxy IP addresses you can do that by passing a space separated list to the application through the `-trusted-proxies` flag.

Doing this will allow the rate limiter middleware to skip past those IPs when looking for the real IP to use for tracking the token count.
