[![Join the chat at https://gitter.im/andrewchambers/cc](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewchambers/cc?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Build Status](https://travis-ci.org/andrewchambers/cc.svg?branch=master)](https://travis-ci.org/andrewchambers/cc)

## Project Goals

The goal of the project is to create a minimalist, useful, cross platform C compiler. I'd like to see...

- C11 Compatibility.
- Both Windows and *nix work equally well.
- Zero config toolchain builds.
- Zero config cross compilation (Or as close as possible, this includes support libraries like libc).
- Toolchain builds in the blink of an eye (I usually take a nap between GCC/Clang builds).
- Good documentation and a low learning curve.
- A companion assembler/linker to remove the dependence on binutils.
- Aggressive removal of cruft.
- An SSA optimizing backend AFTER feature completion.
- Compilation of Go1.4 so we can bootstrap ourselves.

## Status

- Currently only x86_64 machines are supported.
- Currently I run tests on Windows with cygwin, and Arch Linux. Any Linux should work. 
- Not much works yet, but this will change.

It is a work in progress and a hobby project. The test suite is the best idea of what currently works, though other features may be
partially implemented.

## Building

- Install the go compiler, and a working version of gcc (gcc won't be needed in the future).
- Ensure your ```$GOPATH``` environmental variable is setup correctly.
- Run the following in a terminal.
```
$ go get github.com/andrewchambers/cc
$ cd $GOPATH/src/github.com/andrewchambers/cc
$ go test -v
$ go build
$ ./cc -h
```

## Hacking

The code is heavily inspired by https://github.com/rui314/8cc as well as http://bellard.org/tcc/. 
I recommend studying the source code of 8cc before contributing here, as 8cc is currently far more mature.

The compiler is implemented in Go, builds should be quick, and barriers to contribution
should be kept to a minimum. (If you treat Go as the successor to C while ignoring C++ ;), this makes sense.)
Go has excellent support for tests, refactoring, analysis, documentation, code coverage, so we should try to use them.

The compiler currenty does no optimization, this is intentional. This may change in the future, but I would
prefer a slow, but working program, to a broken program. 100 percent test coverage is reachable with a
simple backend.

Contributions to this project are welcome, please discuss ideas on the project gitter before commencing work.
You will probably need to discuss progress/direction for various aspects of the compiler to avoid duplicate/wasted work.
I will try to maintain a code review culture in order to maintain a high standard of work.

## Bug reports

The compiler is currently so immature that it is trivial to find bugs. I don't need bug reports unless
you are also willing to fix the bug yourself in a pull request or by emailing a diff. If you want to tackle an issue, be sure to add a
self contained snippet or file that reproduces the issue.

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
