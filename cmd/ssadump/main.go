// antha-tools/cmd/ssadump/main.go: Part of the Antha language
// Copyright (C) 2014 The Antha authors. All rights reserved.
// 
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
// 
// For more information relating to the software or licensing issues please
// contact license@antha-lang.org or write to the Antha team c/o 
// Synthace Ltd. The London Bioscience Innovation Centre
// 1 Royal College St, London NW1 0NH UK


// ssadump: a tool for displaying and interpreting the SSA form of Go programs.
package main

import (
	"flag"
	"fmt"
	"github.com/antha-lang/antha/build"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/antha-lang/antha-tools/antha/loader"
	"github.com/antha-lang/antha-tools/antha/ssa"
	"github.com/antha-lang/antha-tools/antha/ssa/interp"
	"github.com/antha-lang/antha-tools/antha/types"
)

var buildFlag = flag.String("build", "", `Options controlling the SSA builder.
The value is a sequence of zero or more of these letters:
C	perform sanity [C]hecking of the SSA form.
D	include [D]ebug info for every function.
P	log [P]ackage inventory.
F	log [F]unction SSA code.
S	log [S]ource locations as SSA builder progresses.
G	use binary object files from gc to provide imports (no code).
L	build distinct packages seria[L]ly instead of in parallel.
N	build [N]aive SSA form: don't replace local loads/stores with registers.
`)

var testFlag = flag.Bool("test", false, "Loads test code (*_test.go) for imported packages.")

var runFlag = flag.Bool("run", false, "Invokes the SSA interpreter on the program.")

var interpFlag = flag.String("interp", "", `Options controlling the SSA test interpreter.
The value is a sequence of zero or more more of these letters:
R	disable [R]ecover() from panic; show interpreter crash instead.
T	[T]race execution of the program.  Best for single-threaded programs!
`)

const usage = `SSA builder and interpreter.
Usage: ssadump [<flag> ...] <args> ...
Use -help flag to display options.

Examples:
% ssadump -build=FPG hello.go            # quickly dump SSA form of a single package
% ssadump -run -interp=T hello.go        # interpret a program, with tracing
% ssadump -run -test unicode -- -test.v  # interpret the unicode package's tests, verbosely
` + loader.FromArgsUsage +
	`
When -run is specified, ssadump will run the program.
The entry point depends on the -test flag:
if clear, it runs the first package named main.
if set, it runs the tests of each package.
`

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func init() {
	// If $GOMAXPROCS isn't set, use the full capacity of the machine.
	// For small machines, use at least 4 threads.
	if os.Getenv("GOMAXPROCS") == "" {
		n := runtime.NumCPU()
		if n < 4 {
			n = 4
		}
		runtime.GOMAXPROCS(n)
	}
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintf(os.Stderr, "ssadump: %s.\n", err)
		os.Exit(1)
	}
}

func doMain() error {
	flag.Parse()
	args := flag.Args()

	conf := loader.Config{
		Build:         &build.Default,
		SourceImports: true,
	}
	// TODO(adonovan): make antha/types choose its default Sizes from
	// build.Default or a specified *build.Context.
	var wordSize int64 = 8
	switch conf.Build.GOARCH {
	case "386", "arm":
		wordSize = 4
	}
	conf.TypeChecker.Sizes = &types.StdSizes{
		MaxAlign: 8,
		WordSize: wordSize,
	}

	var mode ssa.BuilderMode
	for _, c := range *buildFlag {
		switch c {
		case 'D':
			mode |= ssa.GlobalDebug
		case 'P':
			mode |= ssa.LogPackages | ssa.BuildSerially
		case 'F':
			mode |= ssa.LogFunctions | ssa.BuildSerially
		case 'S':
			mode |= ssa.LogSource | ssa.BuildSerially
		case 'C':
			mode |= ssa.SanityCheckFunctions
		case 'N':
			mode |= ssa.NaiveForm
		case 'G':
			conf.SourceImports = false
		case 'L':
			mode |= ssa.BuildSerially
		default:
			return fmt.Errorf("unknown -build option: '%c'", c)
		}
	}

	var interpMode interp.Mode
	for _, c := range *interpFlag {
		switch c {
		case 'T':
			interpMode |= interp.EnableTracing
		case 'R':
			interpMode |= interp.DisableRecover
		default:
			return fmt.Errorf("unknown -interp option: '%c'", c)
		}
	}

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Profiling support.
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Use the initial packages from the command line.
	args, err := conf.FromArgs(args, *testFlag)
	if err != nil {
		return err
	}

	// The interpreter needs the runtime package.
	if *runFlag {
		conf.Import("runtime")
	}

	// Load, parse and type-check the whole program.
	iprog, err := conf.Load()
	if err != nil {
		return err
	}

	// Create and build SSA-form program representation.
	prog := ssa.Create(iprog, mode)
	prog.BuildAll()

	// Run the interpreter.
	if *runFlag {
		var main *ssa.Package
		pkgs := prog.AllPackages()
		if *testFlag {
			// If -test, run all packages' tests.
			if len(pkgs) > 0 {
				main = prog.CreateTestMainPackage(pkgs...)
			}
			if main == nil {
				return fmt.Errorf("no tests")
			}
		} else {
			// Otherwise, run main.main.
			for _, pkg := range pkgs {
				if pkg.Object.Name() == "main" {
					main = pkg
					if main.Func("main") == nil {
						return fmt.Errorf("no func main() in main package")
					}
					break
				}
			}
			if main == nil {
				return fmt.Errorf("no main package")
			}
		}

		if runtime.GOARCH != build.Default.GOARCH {
			return fmt.Errorf("cross-interpretation is not yet supported (target has GOARCH %s, interpreter has %s)",
				build.Default.GOARCH, runtime.GOARCH)
		}

		interp.Interpret(main, interpMode, conf.TypeChecker.Sizes, main.Object.Path(), args)
	}
	return nil
}