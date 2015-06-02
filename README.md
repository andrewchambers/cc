# {"C", "Compiler", "WIP", "Minimalist"};

![](https://raw.githubusercontent.com/andrewchambers/cc-images/master/Gopher.png)

Artwork by [Egon Elbre](https://twitter.com/egonelbre) based on the [Go gopher](https://blog.golang.org/gopher) by [Renee French](http://reneefrench.blogspot.com/)

## Goals

- C11 Compatibility, GNU Compatibility.
- Aggressive removal of cruft.
- Windows and *nix as equals.
- Build from source in the blink of an eye.
- Easy to contribute.

## Status 
[![Build Status](https://travis-ci.org/andrewchambers/cc.svg?branch=master)](https://travis-ci.org/andrewchambers/cc)

- Currently only x86_64 is supported.
- Not much works yet, but this will change.

It is a work in progress and a hobby project. The test suite is the best idea of what currently works, though other features may be
partially implemented.

## Building

- Install the go compiler, and a working version of gcc.
- Ensure your ```$GOPATH``` environmental variable is setup correctly.
- Run the following in a terminal.
```
$ mkdir -p $GOPATH/src/github.com/andrewchambers/
$ cd $GOPATH/src/github.com/andrewchambers/
$ git clone https://github.com/andrewchambers/cc
$ sh ./test.sh
$ go install github.com/andrewchambers/cc/cmd/x64cc
$ x64cc -h
```
or just use go get.

## Contact

[![Join the chat at https://gitter.im/andrewchambers/cc](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewchambers/cc?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Hacking

The code is heavily inspired by https://github.com/rui314/8cc as well as http://bellard.org/tcc/. 
I recommend studying the source code of 8cc before contributing here, as 8cc is currently far more mature.

The compiler is implemented in Go. Go has excellent support for cross-platform code, tests, refactoring, analysis, documentation, code coverage, so we should try to use them.

The compiler currenty does no optimization, this is intentional. This may change in the future, but I would
prefer a slow, but working program, to a broken program. 100 percent test coverage is reachable with a
simple backend.

Contributions to this project are welcome, please discuss ideas on the project gitter before commencing work.
You will probably need to discuss progress/direction for various aspects of the compiler to avoid duplicate/wasted work.

## Bug reports

Most things don't work, this is currently the state of affairs, If in doubt, write message in gitter before creating an issue. Failing test cases will be welcome once expected failures are part of the test runner.

When the compiler is more mature, we can do automatic bug hunting using the following resources:

- https://embed.cs.utah.edu/csmith/ (https://github.com/csmith-project/csmith)
- https://github.com/gcc-mirror/gcc/tree/master/gcc/testsuite/gcc.c-torture
- https://github.com/rui314/8cc/tree/master/test

The bugs can then be automatically reduced to minimal form using http://embed.cs.utah.edu/creduce/ (https://github.com/csmith-project/creduce).

## Fun Ideas
- Concurrency using goroutines - C can be compiled a function at a time, so there is a lot of room for this.
- Compile toolchain to javascript using Gopherjs, cross compile using javascript.
- Implement a companion Go -> C compiler, then compile ourselves with it.
- Allow preprocessor include paths from archives to allow sdk's to be packaged as a single binary + archive.
- Implement a backend that is similar to llvm, expose this as a library for other language frontends.
- A companion assembler/linker to remove the dependence on binutils.
- An SSA optimizing backend AFTER feature completion.
- Compilation of Go1.4 so we can bootstrap ourselves.
