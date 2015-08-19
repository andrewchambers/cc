# { "Minimalist", "C", "Compiler"}; /* WIP */

![](https://raw.githubusercontent.com/andrewchambers/cc-images/master/Gopher.png)

Artwork by [Egon Elbre](https://twitter.com/egonelbre) based on the [Go gopher](https://blog.golang.org/gopher) by [Renee French](http://reneefrench.blogspot.com/)

## Goals

- Aggressive removal of cruft.
- Fast.
- Simple.

## Status 

*NOT UNDER DEVELOPMENT*

This compiler has been ported to C here https://github.com/andrewchambers/c where development continues. This was done to allow self hosting far earlier, increase 
speed, increase portability, and decrease binary size. Interestingly, the C version is less lines of code too.

## Building

```go get github.com/andrewchambers/cc/cmd/x64cc```

## Contact

[![Join the chat at https://gitter.im/andrewchambers/cc](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewchambers/cc?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Hacking

The code is heavily inspired by https://github.com/rui314/8cc as well as http://bellard.org/tcc/. 
I recommend studying the source code of 8cc before contributing here, as 8cc is currently far more mature.

- The compiler is implemented in Go.
- The compiler *currenty* does no optimization, this is intentional.
- Contributions to this project are welcome, I will respond on gitter or via email.

## Bugs

Yes.

When the compiler is more mature, we can do automatic bug hunting using the following resources:

- https://embed.cs.utah.edu/csmith/ (https://github.com/csmith-project/csmith)
- https://github.com/gcc-mirror/gcc/tree/master/gcc/testsuite/gcc.c-torture
- https://github.com/rui314/8cc/tree/master/test

The bugs can then be automatically reduced to minimal form using http://embed.cs.utah.edu/creduce/ (https://github.com/csmith-project/creduce).

## Ideas
- Concurrency using goroutines - C can be compiled a function at a time, so there is a lot of room for this.
- Compile toolchain to javascript using Gopherjs, make a demo site.
- Implement a companion Go -> C compiler, then compile ourselves with it.
- Allow preprocessor include paths from archives to allow sdk's to be packaged as a single binary + archive.
- Implement a backend that is similar to llvm, expose this as a library for other language frontends.
- A companion assembler/linker to remove the dependence on binutils.
- An SSA optimizing backend.
- Compilation of Go1.4 so we can bootstrap ourselves.
