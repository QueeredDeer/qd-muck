# QD MUCK

## What?

A reverse-engineered version of the [tinyMUCK/fuzzball server](https://github.com/fuzzball-muck/fuzzball) in Golang. Since I'm looking to use this as a way to learn more idiomatic Go, the main implementation resources will be the tinyMUCK [docs](https://www.realityfault.org/programmer/docs/index.html) and the [MINK](https://fuzzball-muck.github.io/muckman/) guide to MPI and general server behavior, rather than the C sources of the original project.

## Why?

Mostly as an experiment in adding more modern features to the MUCK ecosystem, and to have an excuse for playing around with an end-to-end deployment of a Golang application, from code to CI/CD to deployed image.

## How?

Primarily in Golang, seeing just how much I can exploit goroutines and channels to distribute the design. Luckily, Go comes with lots of great packages to help get web apps and services running, so we'll make good use of [crypto/tls](https://pkg.go.dev/crypto/tls), [net/http](https://pkg.go.dev/net/http), and handy third-party libraries to make the dream a reality.

## What's Different?

TBA!
