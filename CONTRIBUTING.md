# Contributing

This guide is meant to help you start contributing to the Symfony CLI by
providing some key hints and explaining specifics related to this project.

## Language choice

First-time contributors could be surprised by the fact that this project is
written in Go whereas it is highly related to the Symfony Framework which is
written in PHP.

Go has been picked because it is well suited for system development and has
close-to-zero runtime dependencies which make releasing quite easy. This is
ideal for a tool that is used on a wide range of platforms and potentially on
systems where the requirements to run Symfony are not met. Go is also usually
quite easy to apprehend for PHP developers having some similarities in their
approach.

## Setup Go

Contributing to the CLI, implies that one must first setup Go locally on their
machine. Instructions are available on the official
[Go website](https://golang.org/dl). Just pick the latest version available: Go
will automatically download the version currently in use in the project and
dependencies as required.

## Local setup

First fork this repository and clone it in some location of your liking. Next,
try to build and run the project:

```bash
$ go build .
```

If any error happen you must fix them before going on. If no error happen, this
should produce a binary in the project directory. By default, this binary is
named `symfony-cli` and suffixed with `.exe` on Windows.

You should be able to run it right away:

```bash
$ ./symfony-cli version
```

The binary is self-contained: you can copy it as-is to another system and/or
execute it without any installation process.

> *Tip:* This binary can be executed from anywhere by using it's absolute path.
> This is handy during development when you need to run it in a project
> directory and you don't want to overwrite your system-wide Symfony CLI.

Finally, before and after changing code you should ensure tests are passing:

```bash
$ go test ./...
```

## Coding style

The CLI follows the Go standard code formatting. To fix the code formatting one
can use the following command:

```bash
$ go fmt ./...
```

One can also uses the `go vet` command in order to fix common mistakes:

```bash
$ go vet ./...
```

## Cross compilation

By definition, the CLI has to support multiple platforms which means that at
some point you might need to compile the code for another platform than the one
your are using to develop.

This can be done using Go cross-platform compiling capabilities. For example
the following command will compile the CLI for Windows:

```bash
$ GOOS=windows go build .
```

`GOOS` and `GOARCH` environment variables are used to target another OS or CPU
architecture, respectively.

During development, please take into consideration (in particular in the
process and file management sections) that we currently support the following
platforms matrix:

- Linux / 386
- Linux / amd64
- Linux / arm
- Linux / arm64
- Darwin / amd64
- Darwin / arm64
- Windows / 386
- Windows / amd64

## Code generation

Part of the code is generated automatically. One should not need to regenerate
the code themselves because a GitHub Action is in-charge of it. In the
eventuality one would need to debug it, code generation can be run as follows:

```bash
$ go generate ./...
```

If you add a new code generation command, please also update the GitHub
workflow in `.github/workflows/go_generate_update.yml`.

## Additional repositories

Contrary to the Symfony PHP Framework which is a mono-repository, the CLI
tool is developed in multiple repositories. `symfony-cli/symfony-cli` is the
main repository where lies most of the logic and is the only repository
producing a binary.

Every other repository is mostly independent and it is highly unlikely that
you would need to have a look at them. However, in the eventuality where you
would have to, here is the description of each repository scope:
- `symfony-cli/phpstore` is a independent library in charge of the PHP
  installations discovery and the logic to match a specific version to a given
  version constraint.
- `symfony-cli/console` is a independent library created to ease the process
  of Go command-line application. This library has been created with the goal
  of mimicking the look and feel of the Symfony Console for the end-user.
- `symfony-cli/terminal` is a wrapper around the Input and Output in a command
  line context. It provides helpers around styling (output formatters and
  styling - à la Symfony) and interactivity (spinners and questions helpers)
- `symfony-cli/dumper` is a library similar to Symfony's VarDumper component
  providing a `Dump` function useful to introspect variables content and
  particularly useful in the strictly typed context of Go.

If you ever have to work on those package, you can setup your local working
copy of the CLI to work with a local copy of one of those package by using
`go work`:

```bash
$ go work init
$ go work use .
$ go work use /path/to/package/fork
# repeat last command for each package you want to work with
```
