[![Join the chat at https://gitter.im/andrewchambers/cc](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewchambers/cc?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Project Goals

The goal of the project is to create a minimalist, useful, cross platform C compiler. I'd like to see...

- Compilation of Go1.4 so we can bootstrap ourselves.
- Both Windows and *nix work equally well.
- Zero config toolchain builds.
- Zero config cross compilation (Or as close as possible, this includes support libraries like libc).
- Toolchain builds in the blink of an eye (I usually take a nap/shower between GCC/Clang builds).
- Good documentation and a low learning curve.
- An SSA optimizing backend AFTER feature completion.
- A companion assembler/linker to remove the dependence on binutils.
- Aggressive removal of cruft.

## Status

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

## Fun Ideas
- Concurrency using goroutines - C can be compiled a function at a time, so there is a lot of room for this.
- Compile toolchain to javascript using Gopherjs, cross compile using javascript.
- Implement a companion Go -> C compiler, then compile ourselves with it.
- Allow preprocessor include paths from archives to allow sdk's to be packaged as a single binary + archive.
- Implement a backend that is similar to llvm, expose this as a library for other language frontends.
