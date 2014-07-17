Status WIP. 

The goal of the project is to create a small cross platform toolchain for working with C code in windows and linux.
This includes preprocessing, code analysis and assembly generation. It should work on any platform Go supports and support trivial cross platform C compilation.

I'm investigating some features that aren't typically in C compilers that are made easier by some of Go's functionality. 

Some examples might be:

- Multiple cores in internal compiler pipeline using channels where it makes sense and simplifies design.
- Emphasis on simple code (helped by garbage collection), with good enough performance.
- Emphasis on readability for people learning compiler internals, comments should assume less internal knowledge than other compilers. Not elitist, and less complicated than a C++ compiler by supporting a bunch of cruft not needed by C.
- Pluggable #include mechanisms (like remote code or archived code)
- All crosscompilation is done with the same compiler binaries with intuitive configuration. 
- Whole compiler will build icredibly fast, seconds not minutes or hours. (GCC has taken me over 30 minutes just for the C frontent.)
- Anything else that might be fun.

I also want to put emphasis on good documentation, good code coverage tooling and profiling, and avoiding code bloat by limiting the scope to C and keeping parts as importable modules so external tools can do more fancy features.

The project is designed as a set of Go libraries which can all be imported individually, used in multithreaded environments, and used in servers. My original intentions are for a small well documented
compiler for hobbyists but I don't want to limit it and thus desire high quality and modularity.

A lexer, a preprocessor, C parser, C semantic analysis and assembly generation are planned.

A possibility is that the AST will be able to be dumped to a simple and portable serialization format so that multiple languages can be used (OCAML and Haskell excel at working with trees), This probably is useful for research and experimental backends if well designed.

Building:

set the GOPATH to the root. On linux I use export GOPATH=`pwd`
then execute:
go build cc

More binaries such as a standalone preprocessor, or a gcc compatible compiler driver may be added.

Tests can be executed via:
go test cc
go test cc/cpp
etc...

Using the builtin tools allow the standard Go code coverage, and benchmarking to work uniformly, which is extremely useful and high quality.

Dev environment:

Go compilation is very standard so any Go IDE should just work with minimal work. I personally use GoSublime which
gives me full autocompletion support with the only setup being a single environment variable.

Notes:
Architecture support will be limited to X86_64 and possible X86 initially, but I would love the toolchain to work for embedded C code on windows. ABI compatibility with gcc is a goal.

The initial assembly generation will not be optimizing and will follow the same techniques as TCC and 8cc
https://github.com/rui314/8cc. This is to limit project scope. If the project is successful, an SSA based IR can be added later with low impact on the frontend.

A seperate portable assembler and linker which supports ELF and PE binaries is possible, but not currently a priority.
