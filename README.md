# srclib-bash

**srclib-bash** is a [srclib](https://srclib.org)
toolchain that performs Bash (and POSIX Shell) code analysis: linking command names to man pages.

It enables this functionality in any client application whose code analysis is
powered by srclib, including [Sourcegraph.com](https://sourcegraph.com).

## Installation

This toolchain is not a standalone program; it provides additional functionality
to applications that use [srclib](https://srclib.org).

First,
[install the `srclib` program (see srclib installation instructions)](https://sourcegraph.com/sourcegraph/srclib).

Then run:

```
# download and fetch dependencies
go get -v sourcegraph.com/sourcegraph/srclib-bash
cd $GOPATH/src/sourcegraph.com/sourcegraph/srclib-bash

# build the srclib-bash program in .bin/srclib-bash (this is currently required by srclib to discover the program)
make

# link this toolchain in your SRCLIBPATH (default ~/.srclib) to enable it
```

To verify that installation succeeded, run:

```
srclib toolchain list
```

You should see this srclib-bash toolchain in the list.

Now that this toolchain is installed, any program that relies on srclib will support Bash.

## Known issues

srclib-bash is alpha-quality software. It powers code analysis on
[Sourcegraph.com](https://sourcegraph.com) but has not been widely tested or
adapted for other use cases.
