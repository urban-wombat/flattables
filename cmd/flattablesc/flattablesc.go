package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	//	"path"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/urban-wombat/flattables"
	"github.com/urban-wombat/gotables"
	"github.com/urban-wombat/util"
)

//	import "github.com/davecgh/go-spew/spew"

type Flags struct {
	f string // BOTH schema AND data file name
	n string // <namespace> (also sets TableSet name)
	p string // <package-name>
	o string // <out-dir-package>
	O string // <out-dir-package>
	s string // <out-dir-main>	defaults to <out-dir-package>/cmd/<package-name>.go
	m bool   // mutable	// Note: mutable (non-const) FlatBuffers apparently unavailable in Go
	v bool   // verbose
	d bool   // Dry Run
	h bool   // help
}

var flags Flags

var globalGotablesFileNameAbsolute string // from flags.f via filepath.Abs()
// var globalRelationsFileName string        // from flags.r
var globalNameSpace string          // from flags.n
var globalPackageName string        // from flags.p
var globalOutDirAbsolute string     // from (optional) flags.o or flags.O via filepath.Abs()
var globalFlagOWarnOnly bool        // if flags.O (capital O) is set
var globalOutDirMainAbsolute string // from (optional) flags.s via filepath.Abs()
var globalMutableFlag string        // Pass to flatc. Note: mutable (non-const) FlatBuffers apparently unavailable in Go.
var globalUtilName string           // "flattablesc"
var globalUtilDir string            // "flattablesc"

func init() {
	log.SetFlags(log.Lshortfile) // For var where
}

var where = log.Print

// Custom flag.
// Note: For custom flags that satisfy the Value interface, the default value is just the initial value of the variable.
// See: https://golang.org/pkg/flag
type stringFlag struct {
	set bool
	val string
}

// Custom flag.
func (sf *stringFlag) Set(x string) error {
	sf.val = x
	sf.set = true
	return nil
}

// Custom flag.
func (sf *stringFlag) String() string {
	return sf.val
}

var tablesTemplateInfo flattables.TablesTemplateInfoType

func initFlags() {
	/*
		1. variable pointer
		2. -flagname
		3. default value (except for custom flags that satisfy the Value interface, which default to the initial value of the variable)
		4. help message for flagname
	*/
	var err error

	flag.StringVar(&flags.f, "f", "", fmt.Sprintf("<infile> of schema/data tables"))
	flag.StringVar(&flags.n, "n", "", fmt.Sprintf("Sets schema file <namespace>.fbs, schema root_type, FlatBuffers namespace, TableSet name"))
	flag.StringVar(&flags.p, "p", "", fmt.Sprintf("<package-name> Sets package name"))
	flag.StringVar(&flags.o, "o", "", fmt.Sprintf("<out-dir> Default is ../<namespace>"))
	flag.StringVar(&flags.O, "O", "", fmt.Sprintf("<out-dir> Default is ../<namespace>"))
	flag.StringVar(&flags.s, "s", "", fmt.Sprintf("<sample-main-out-dir> Default is ../<out-dir>/cmd/<namespace>"))
	flag.BoolVar(&flags.m, "m", false, fmt.Sprintf("generate additional non-const accessors for mutating FlatBuffers in-place"))
	flag.BoolVar(&flags.v, "v", false, fmt.Sprintf("verbose"))
	flag.BoolVar(&flags.d, "d", false, fmt.Sprintf("dry run"))
	flag.BoolVar(&flags.h, "h", false, fmt.Sprintf("print flattables usage"))

	flag.Parse()

	if flags.h {
		// help
		printUsage()
		os.Exit(1)
	}

	const (
		compulsoryFlag = true
		optionalFlag   = false
	)

	var flagExists bool

	// Input file of gotables tables to be used as a schema, and possibly data.
	checkStringFlagReplaceWithUtilVersion("f", flags.f, compulsoryFlag)
	var globalGotablesFileName string = flags.f
	globalGotablesFileNameAbsolute, err = util.FilepathAbs(globalGotablesFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		printUsage()
		os.Exit(14)
	}
	// Change backslashes to forward slashes. Otherwise strings interpret them as escape chars.
	globalGotablesFileNameAbsolute = filepath.ToSlash(globalGotablesFileNameAbsolute)

	// Namespace
	// Compulsory flag.
	checkStringFlagReplaceWithUtilVersion("n", flags.n, compulsoryFlag)
	globalNameSpace = flags.n
	// globalNameSpace has the same validity criteria as gotables col names and table names.
	isValid, _ := gotables.IsValidColName(globalNameSpace)
	if !isValid {
		fmt.Fprintf(os.Stderr, "Error: non-alpha-numeric-underscore chars in -n <namespace>: %q\n", flags.n)
		fmt.Fprintf(os.Stderr, "Note:  <namespace> is not a file or dir name. Though it is used in file and dir names.\n")
		printUsage()
		os.Exit(9)
	}

	if flags.m {
		globalMutableFlag = "--gen-mutable" // Generate additional non-const accessors to mutate FlatBuffers in-place.
	}

	// Package
	checkStringFlagReplaceWithUtilVersion("p", flags.p, compulsoryFlag)
	globalPackageName = flags.p
	// Package name must include namespace.
	if !strings.HasSuffix(globalPackageName, globalNameSpace) {
		fmt.Fprintf(os.Stderr, "tail end of <package-name> -p %q must match <namespace> -n %q\n", globalPackageName, globalNameSpace)
		printUsage()
		os.Exit(12)
	}
	// Detect an easy package name error (looks like relative path name).
	if strings.HasPrefix(globalPackageName, ".") {
		fmt.Fprintf(os.Stderr, "invalid <package-name> -p %s (leading '.')\n", globalPackageName)
		printUsage()
		os.Exit(12)
	}

	// Set default outDir. May be provided (optionally) with -o <out-dir>
	var outDir string = "../" + globalNameSpace // Package level, where globalNameSpace is package name.
	flagExists = checkStringFlagReplaceWithUtilVersion("o", flags.o, optionalFlag)
	if flagExists { // Has been set explicitly with -o
		outDir = flags.o
	}
	// Capital -O means warnOnly.
	flagExists = checkStringFlagReplaceWithUtilVersion("O", flags.O, optionalFlag)
	if flagExists { // Has been set explicitly with -O
		outDir = flags.O
		globalFlagOWarnOnly = true
	}
	globalOutDirAbsolute, err = filepath.Abs(outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		printUsage()
		os.Exit(14)
	}
	// Change backslashes to forward slashes. Otherwise strings interpret them as escape chars.
	globalOutDirAbsolute = filepath.ToSlash(globalOutDirAbsolute)
	if inconsistent, err := inconsistentPackageAndOutDir(globalPackageName, globalOutDirAbsolute); inconsistent {
		if globalFlagOWarnOnly {
			fmt.Fprintf(os.Stderr, "WARNING: %v\n", err)
			fmt.Fprintf(os.Stderr, "         go test will work, but main will not be able to find its package\n")
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			printUsage()
			os.Exit(12)
		}
	}

	// Set default globalOutDirMainAbsolute. May be provided (optionally) with -s <out-dir-main>
	// <out-dir-main>  defaults to <out-dir>/cmd/<package-name>
	globalOutDirMainAbsolute = fmt.Sprintf("%s/cmd/%s", globalOutDirAbsolute, globalNameSpace)
	flagExists = checkStringFlagReplaceWithUtilVersion("s", flags.s, optionalFlag)
	if flagExists { // Has been set explicitly with -s
		globalOutDirMainAbsolute = flags.s
	}
	globalOutDirMainAbsolute, err = filepath.Abs(globalOutDirMainAbsolute)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		printUsage()
		os.Exit(14)
	}
	// Change backslashes to forward slashes. Otherwise strings interpret them as escape chars.
	globalOutDirMainAbsolute = filepath.ToSlash(globalOutDirMainAbsolute)
}

func progName() string {
	return filepath.Base(os.Args[0])
}

/*
	This function does naughty things:-
		- Does not return any values.
		- Has global side-effects: Calls os.Exit().
	It avoids heaps of boilerplate code testing flags.
	It can be called and:-
		- Compulsory flags can trust the existence of an argument.
		- Optional flags can test exists.
	The flag library itself does some global stuff: bails out if a flag does not have an argument.
*/
func checkStringFlagReplaceWithUtilVersion(name string, arg string, compulsory bool) (exists bool) {
	var hasArg bool

	if arg != "" {
		exists = true
	}

	// Try to detect missing flag argument.
	// If an argument is another flag, argument has not been provided.
	if exists && !strings.HasPrefix(arg, "-") {
		// Option expecting an argument but has been followed by another flag.
		hasArg = true
	}
	/*
		where(fmt.Sprintf("-%s compulsory = %t", name, compulsory))
		where(fmt.Sprintf("-%s exists     = %t", name, exists))
		where(fmt.Sprintf("-%s hasArg     = %t", name, hasArg))
		where(fmt.Sprintf("-%s value      = %s", name, arg))
	*/

	if compulsory && !exists {
		fmt.Fprintf(os.Stderr, "compulsory flag: -%s\n", name)
		printUsage()
		os.Exit(2)
	}

	if exists && !hasArg {
		fmt.Fprintf(os.Stderr, "flag -%s needs a valid argument (not: %s)\n", name, arg)
		printUsage()
		os.Exit(3)
	}

	return
}

func printUsage() {
	var usageSlice []string = []string{
		"usage:       ${globalUtilName} [-v] [-d] -f <gotables-file> -n <namespace> -p <package-name> [-o <out-dir>] [-s <out-dir-main>]",
		"purpose: (1) Generate a FlatBuffers schema file <namespace>.fbs from a set of tables.",
		"         (2) Generate official Flatbuffers Go code (from <namespace>.fbs) using flatc --go",
		"         (3) Generate additional Go code to read/write these specific table types from gotables objects.",
		"flags:   -f  Input text file containing one or more gotables tables (generates schema).",
		"             See flattables_sample: https://github.com/urban-wombat/flattables_sample/blob/master/tables.got",
		"             Note: The file need not contain data. Only metadata (names and types) will be used for Go code generation.",
		"                   If there is data in the input file, it will be used for running benchmarks.",
		"         -n  Namespace  Sets schema file <namespace>.fbs, schema root_type, FlatBuffers namespace, TableSet name.",
		"             Note: Generated Go code will be placed adjacently at Go package level.",
		"                   This assumes you are running ${globalUtilName} at package level.",
		"                   You may override this with -o <out-dir>",
		"         -p  Package  Sets Go package name. Needs to include Namespace.",
		"        [-o] <out-dir> Where to put generated Go code files. Default is ../<namespace>",
		"             Note: The tail end of <out-dir> must match -p <package-name>",
		"        [-O] <out-dir> Allow generated code to go where <out-dir> does NOT match -p <package-name> (will print WARNING)",
		"             Note: go test will work, but main will not be able to find its package",
		"        [-s] <out-dir-main> Where to put generated sample main Go code file. Default is <out-dir>/cmd/<package-name>",
		//		"         -m  Mutable  Tells flatc to add mutable methods to its Go code generation: Mutate...()",
		"types:       Architecture-dependent Go types int and uint are not used. Instead use e.g. int64, uint32, etc.",
		"             Go types not implemented: complex32 complex64.",
		//		"names:       Table names are UpperCamelCase, column names are lowerCamelCase, as per the FlatBuffers style guide.",
		//		"deprecation: To deprecate a column, append its name with _DEPRECATED_ (warning: deprecation may break tests and old code).",
		"        [-v] Verbose",
		"        [-d] Dry run (also turns on Verbose)",
		"        [-h] Help",
		"sample:      This sample assumes package name \"github.com/urban-wombat/flattables_sample\".",
		"             Make a Go package dir: $ mkdir flattables_sample",
		"             $ cd flattables_sample",
		"             Create a gotables file: tables.got",
		"             $ ${globalUtilName} -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample",
		//		"             $ ${globalUtilName} -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample -m",
		//		"             $ go run ${globalUtilName}.go -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample",
		"             $ Test: cd ../flattables_sample; go test -bench=.",
		"             $ Test: cd ../flattables_sample/cmd/flattables_sample; go run flattables_sample_main.go",
	}

	var usageString string
	for i := 0; i < len(usageSlice); i++ {
		usageString += usageSlice[i] + "\n"
	}

	// For debugging or new code, conditionally add provisional command line examples under development.
	if user, _ := user.Current(); user.Username == "Malcolm-PC\\Malcolm" {
		// We are testing. Provide a useful sample. Does not appear in final product.
		usageString += "additional commands in development mode:\n"
		usageString += "             $ go run ${globalUtilDir}/${globalUtilName}.go -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample\n"
		usageString += "             $ go install ${globalUtilDir}/${globalUtilName}.go\n"
		usageString += buildTime()
	}

	usageString = strings.Replace(usageString, "${globalUtilName}", globalUtilName, -1)
	usageString = strings.Replace(usageString, "${globalUtilDir}", globalUtilDir, -1)

	fmt.Fprintf(os.Stderr, "%s\n", usageString)
}

func main() {

	if strings.Contains(os.Args[0], "flattablesc") {
		globalUtilName = "flattablesc"
	} else {
		fmt.Fprintf(os.Stderr, `expecting to be called "flattablesc" not %q`, os.Args[0])
		os.Exit(2)
	}

	if len(os.Args) == 1 {
		// No args.
		fmt.Fprintf(os.Stderr, "%s expects at least 1 argument\n", globalUtilName)
		printUsage()
		os.Exit(2)
	}

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Expecting -f <infile> containing one or more gotables tables\n")
		printUsage()
		os.Exit(13)
	}

	flag.Usage = printUsage // Override the default flag.Usage variable.
	initFlags()

	if flags.d { // dry run, turn on verbose
		flags.v = true
		fmt.Printf(" *** -d DRY-RUN ***\n")
	}

	if flags.v {
		fmt.Printf(" (1) Reading gotables file: %s\n", globalGotablesFileNameAbsolute)
	}
	tableSet, err := gotables.NewTableSetFromFile(globalGotablesFileNameAbsolute)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		printUsage()
		os.Exit(14)
	}

	if flags.v {
		fmt.Printf(" (2) Setting gotables.TableSet name to %q (from -n %s)\n", globalNameSpace, globalNameSpace)
	}
	tableSet.SetName(globalNameSpace)
	tableSet.SetFileName(globalGotablesFileNameAbsolute)

	if flags.v {
		fmt.Printf(" (3) Setting package name to %q (from -p %s)\n", globalPackageName, globalPackageName)
	}

	if !pathExists(globalOutDirAbsolute) {
		if flags.v {
			fmt.Printf(" (4) Creating dir <out-dir>      %s\n", globalOutDirAbsolute)
		}
		permissions := 0777
		if flags.d {
			fmt.Printf(" *** -d dry-run: Would have created <out-dir> %s\n", globalOutDirAbsolute)
		} else {
			err = os.MkdirAll(globalOutDirAbsolute, os.FileMode(permissions))
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(15)
			}
		}
	} else {
		if flags.v {
			fmt.Printf(" (4) Dir <out-dir>      already exists (okay) %s\n", globalOutDirAbsolute)
		}
	}

	if !pathExists(globalOutDirMainAbsolute) {
		if flags.v {
			fmt.Printf(" (5) Creating dir <out-dir-main> %s\n", globalOutDirMainAbsolute)
		}
		permissions := 0777
		if flags.d {
			fmt.Printf(" *** -d dry-run: Would have created <out-dir-main> %s\n", globalOutDirMainAbsolute)
		} else {
			err = os.MkdirAll(globalOutDirMainAbsolute, os.FileMode(permissions))
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(16)
			}
		}
	} else {
		if flags.v {
			fmt.Printf(" (5) Dir <out-dir-main> already exists (okay) %s\n", globalOutDirMainAbsolute)
		}
	}

	// Must be called before flattables.InitTablesTemplateInfo()
	err = flattables.DeleteEmptyTables(tableSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(17)
	}

	//	spew.Dump(tablesTemplateInfo)

	// Template info for ALL the templates.
	if flags.v {
		fmt.Printf(" (6) Preparing tables for schema generation ...\n")
	}
	tablesTemplateInfo, err = flattables.InitTablesTemplateInfo(tableSet, globalPackageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(18)
	}

	// Make these assignments AFTER calling flattables.InitTablesTemplateInfo()
	tablesTemplateInfo.NameSpace = globalNameSpace
	tablesTemplateInfo.GotablesFileNameAbsolute = globalGotablesFileNameAbsolute

	tablesTemplateInfo.OutDirAbsolute = globalOutDirAbsolute
	tablesTemplateInfo.OutDirMainAbsolute = globalOutDirMainAbsolute

	//	spew.Dump(tablesTemplateInfo)

	var tableCount int = tableSet.TableCount()
	if flags.v {
		fmt.Printf("     Adding gotables tables *  to FlatBuffers schema: (%d table%s added)\n", tableCount, plural(tableCount))
	}

	flatBuffersSchemaFileName := globalOutDirAbsolute + "/" + globalNameSpace + ".fbs"
	tablesTemplateInfo.GeneratedFile = filepath.Base(flatBuffersSchemaFileName)
	if flags.v {
		fmt.Printf(" (7) Generating  FlatBuffers schema from gotables file %s    \n", globalGotablesFileNameAbsolute)
		fmt.Printf("     Generating: %s\n", flatBuffersSchemaFileName)
	}
	flatBuffersSchema, err := flattables.FlatBuffersSchemaFromTableSet(tablesTemplateInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(19)
	}

	flatBuffersSchema = flattables.RemoveExcessTabsAndNewLines(flatBuffersSchema)

	if flags.d {
		fmt.Printf(" *** -d dry-run: Would have written file: %s\n", flatBuffersSchemaFileName)
	} else {
		err = ioutil.WriteFile(flatBuffersSchemaFileName, []byte(flatBuffersSchema), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(20)
		}
	}

	// Note: each arg part needs to be passed to exec.Command separately.
	executable := "flatc"
	goFlag := "--go"
	outFlag := "-o" // for flatc
	//		outDirFlatC := ".."	// flatc creates a subdir under this.
	// flatc appends nameSpace to its outDirFlatC. So we need to snip nameSpace from end of outDirFlatC
	outDirFlatC := globalOutDirAbsolute[:len(globalOutDirAbsolute)-len(globalNameSpace)] // Snip off globalNameSpace

	// flatc creates subdir <namespace> under outDirFlatC
	if flags.v {
		fmt.Printf(" (8) From FlatBuffers schema %s\n", flatBuffersSchemaFileName)
	}
	if flags.v {
		fmt.Printf("         generating standard generic Google FlatBuffers Go code:\n")
	}
	if flags.v {
		fmt.Printf("         %s\n", flatBuffersSchemaFileName)
	}
	fmtString := "     $ %s %s %s %s %s\n         %s\n"
	if flags.m { // Mutable
		//			if flags.v { fmt.Printf("     $ %s %s %s %s %s %s\n", executable, goFlag, globalMutableFlag, outFlag, outDirFlatC, flatBuffersSchemaFileName) }
		if flags.v {
			fmt.Printf(fmtString, executable, goFlag, globalMutableFlag, outFlag, outDirFlatC, flatBuffersSchemaFileName)
		}
	} else {
		//			if flags.v { fmt.Printf("     $ %s %s %s %s %s\n",    executable, goFlag,              outFlag, outDirFlatC, flatBuffersSchemaFileName) }
		if flags.v {
			fmt.Printf(fmtString, executable, goFlag, "", outFlag, outDirFlatC, flatBuffersSchemaFileName)
		}
	}
	var cmd *exec.Cmd
	if flags.m { // Mutable
		cmd = exec.Command(executable, goFlag, globalMutableFlag, outFlag, outDirFlatC, flatBuffersSchemaFileName)
	} else {
		cmd = exec.Command(executable, goFlag, outFlag, outDirFlatC, flatBuffersSchemaFileName)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if flags.d {
		fmt.Printf(" *** -d dry-run: Would have run command: %s\n", cmd.Args)
	} else {
		err = cmd.Run()
		if err != nil {
			// Note: err contains the exit code. (?)
			//       out contains the actual error message. (?)
			fmt.Fprintf(os.Stderr, "(a) %s\n", err)
			fmt.Fprintf(os.Stderr, "(b) %s\n", out.String())
			fmt.Fprintf(os.Stderr, "(c) Have you installed flatc ?\n")
			printUsage()
			os.Exit(21)
		}
	}

	if flags.v {
		fmt.Printf(" (*) Generating user Go code ...\n")
	}
	err = flattables.GenerateAll(tablesTemplateInfo, flags.v, flags.d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(22)
	}

	if flags.d {
		fmt.Println(" *** -d DRY-RUN *** (Didn't do anything!)")
	} else {
		fmt.Println(" DONE")
		fmt.Printf("     Test: cd %s; go test -bench=.\n", globalOutDirAbsolute)
		fmt.Printf("     Test: cd %s; go run %s_main.go\n", globalOutDirMainAbsolute, globalNameSpace)
	}

	if user, _ := user.Current(); user.Username == "Malcolm-PC\\Malcolm" {
		fmt.Printf(" %s\n", buildTime())
	}
}

// From: http://www.musingscafe.com/check-if-a-file-or-folder-exists-in-golang
// Also checks directories.
// fileExists
// dirExists
func pathExists(path string) (exists bool) {
	exists = true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exists = false
	}
	return
}

func plural(items int) string {
	if items == 1 || items == -1 {
		// Singular
		return ""
	} else {
		// Plural
		return "s"
	}
}

func buildTime() (buildTime string) {
	stat, err := os.Stat(os.Args[0])
	if err == nil {
		ago := time.Now().Sub(stat.ModTime()).Truncate(time.Second)
		// Can also use ago := time.Now().Sub(stat.ModTime()).Truncate(time.Second)
		executableName := os.Args[0]
		executableName = strings.Replace(executableName, ".exe", "", 1)
		executableName = filepath.Base(executableName)
		buildTime = fmt.Sprintf("    %s.go built %s (%v ago) installed %s\n",
			executableName, stat.ModTime().Format(time.UnixDate), ago, os.Args[0])
	}
	return
}

func inconsistentPackageAndOutDir(packageName string, outDir string) (consistent bool, err error) {
	// Convert outDir to absolute and forward slashes for valid comparison with packageName.
	absolute, err := filepath.Abs(outDir)
	if err != nil {
		return true, err
	}

	// Change backslashes to forward slashes. Otherwise strings interpret them as escape chars.
	absolute = filepath.ToSlash(absolute)

	// See if outDir even contains packageName. Deal-breaker if it doesn't.
	index := strings.Index(absolute, packageName)
	if index < 0 {
		err := fmt.Errorf("-p <package-name> %q is missing from (absolute) -o <out-dir> %s", packageName, absolute)
		return true, err
	}

	absolute = absolute[index:]
	if absolute != packageName {
		err := fmt.Errorf("-p <package-name> does not match END OF -o <out-dir>: -p %s -o %s (end of)", packageName, absolute)
		lenOutDir := len(absolute)
		lenPackageName := len(packageName)
		var length int
		if lenOutDir > lenPackageName {
			length = lenOutDir
		} else {
			length = lenPackageName
		}
		where(fmt.Sprintf("%-*s\n", length, packageName))
		where(fmt.Sprintf("%-*s\n", length, absolute))
		return true, err
	}

	return false, nil
}
