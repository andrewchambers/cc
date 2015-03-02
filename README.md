The goal of the project is to create a minimalist, cross platform C compiler which is accessible to
hobbyists, but can still compile serious applications.

It is implemented in Go because it is cross platform, fast, simple, and has a garbage collector.
Go applications run natively on windows/linux/mac with no effort, and are statically linked, so easy it's install.
Building the compiler is fast and simple, and I plan for it to be "usb key ready" by simply copying the static binary to the desired location.

It is a work in progress, and the test suite is the best idea of what currently works, though other features may be
partially implemented.

It is heavily inspired by https://github.com/rui314/8cc as well as http://bellard.org/tcc/. 
I recommend studying the source code of 8cc before contributing here, as 8cc is currently far more mature.

*Project Trajectory/Stretch Goals.*

 * Compiling non trivial C99/C11 programs with x86_64 with a non optimising backend on linux/windows/macos. Compiling Go1.4 would be a good test, as then we could bootstrap ourselves.
 * Simple optimizing backend, maintaining functionality.
 * Cross compiling and porting, make this compiler simple to port. Keeping the backend fairly simple may support this goal.
 * Companion assembler/linker to remove dependence on binutils.

*Building*

```
$ go get github.com/andrewchambers/cc
$ cd $GOPATH/src/github.com/andrewchambers/cc
$ go test -v
$ go build
$ ./cc -h
```

Collaborators for this project are welcome, please discuss ideas on the project gitter before commencing work.
You will need to discuss progress/direction for various aspects of the compiler to avoid duplicate/wasted work.

*Fun Ideas*

Allow preprocessor include paths from archives to allow sdk's to be packaged as a single binary + archive.
Concurrency using goroutines - C can be compiled a function at a time, so there is a lot of room for this.
Compile to javascript using Gopherjs.

[![Join the chat at https://gitter.im/andrewchambers/cc](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewchambers/cc?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

