# Tofu

Tofu is a base reference project for a hex architecture implementation in Go.
It is still a work in progress and subject to a lot of change.

## Building

To build and run the project you can use all of the normal Go tool commands, but there is also a `build.go` file in the root of the project for convenience.
To use `build.go` just run it with `go run` and pass any flags you would like to use, for example...
```
go run build.go -help
```
...will display the help text for the file.

To run the project with Go vet, tests, and file watching run:
```
go run build.go -debug -watch -clear -after "./web -addr :8080 -log-style text"
```

Omitting the `-debug` flag will build the project in "release" mode which does things like stripping debug symbols, among other things.
