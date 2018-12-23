package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/urban-wombat/flattables"
	"github.com/urban-wombat/gotables"
	"io/ioutil"
	"log"
	"path/filepath"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type Flags struct {
	f string	// BOTH schema AND data file name
	r string	// relations file name for generating GraphQL schema
	n string	// namespace (also sets TableSet name)
	p string	// package name
	m bool		// mutable	// Note: mutable (non-const) FlatBuffers apparently unavailable in Go
	b bool		// generate FlatBuffers"	// The b is for buffers because f and t are already, at least cognitively, taken.
	B bool		// generate FlatBuffers and NOT GraphQL"
	g bool		// generate GraphQL"
	G bool		// generate GraphQL and NOT FlatBuffers"
	v bool		// verbose
	h bool		// help
}
var flags Flags

var gotablesFileName string		// from flags.f
var relationsFileName string	// from flags.r
var nameSpace string			// from flags.n
var packageName string			// from flags.p
var outDir string
var outDirMain string
var mutableFlag string			// Pass to flatc. Note: mutable (non-const) FlatBuffers apparently unavailable in Go.
var utilName string				// "flattablesc" or "graphqlc"

func init() {
	log.SetFlags(log.Lshortfile) // For var where

//	flag.Usage = printUsage	// Override the default flag.Usage variable.
//	initFlags()
}
var where = log.Print


// We might implement separation of schema and data input files sometime in the future.
func initFlags() {
	flag.StringVar(&flags.f, "f", "",    fmt.Sprintf("<infile> of schema/data tables"))
	flag.StringVar(&flags.r, "r", "",    fmt.Sprintf("<infile> of GraphQL schema tables"))
	flag.StringVar(&flags.n, "n", "",    fmt.Sprintf("<namespace> (sets tableset name, root_type, root_table)"))
	flag.StringVar(&flags.p, "p", "",    fmt.Sprintf("<package> (sets package name)"))
	flag.BoolVar(  &flags.b, "b", false, fmt.Sprintf("generate FlatBuffers"))	// flatbuffers
	flag.BoolVar(  &flags.B, "B", false, fmt.Sprintf("generate FlatBuffers"))	// flatbuffers ONLY
	flag.BoolVar(  &flags.g, "g", false, fmt.Sprintf("generate GraphQL"))		// graphql
	flag.BoolVar(  &flags.G, "G", false, fmt.Sprintf("generate GraphQL"))		// graphql ONLY
	flag.BoolVar(  &flags.m, "m", false, fmt.Sprintf("generate additional non-const accessors for mutating FlatBuffers in-place"))
	flag.BoolVar(  &flags.v, "v", false, fmt.Sprintf("verbose"))
	flag.BoolVar(  &flags.h, "h", false, fmt.Sprintf("print flattables usage"))

	flag.Parse()

	if flags.h {
		// help
		printUsage()
		os.Exit(1)
	}

	/*
		Executable file name is a proxy for a flag.
		If the file name contains flattablesc then -b (flattables) flag is turned on and -b on the commandline is unnecwssary to turn it on.
		If the file name contains graphqlc    then -g (graphql)    flag is turned on and -g on the commandline is unnecwssary to turn it on.
		if -B (flatbuffers-only) is set then -g is turned off.
		if -G (graphql-only)     is set then -b is turned Off.
	*/
	if strings.Contains(os.Args[0], "flattablesc") {
		flags.b = true	// As good as -b
		utilName = "flattablesc"
	}
	if strings.Contains(os.Args[0], "graphqlc") {
		flags.g = true	// As good as -g
		utilName = "graphqlc"
	}
	if flags.B {	// flattables ONLY
		flags.b = true
		flags.g = false
	}
	if flags.G {	// graphql ONLY
		flags.g = true
		flags.b = false
	}
	if flags.B && flags.G {
		fmt.Fprintf(os.Stderr, "Cannot have -G AND -B. -G means ONLY graphql. -B means ONLY flattables.\n")
		printUsage()
		os.Exit(3)
	}
	if !flags.b && !flags.g {
		fmt.Fprintf(os.Stderr, "Cannot have NEITHER -g NOR -b. -g means graphql. -b means flattables.\n")
		printUsage()
		os.Exit(4)
	}
	// flags.b and flags.g indicate whether flatbuffers AND/OR graphql are turned on.

	// Input file of gotables tables to be used as a schema, and possibly data.
    // Compulsory flag.
    if flags.f == "" {
        // -f has been followed by an empty argument.
        fmt.Fprintf(os.Stderr, "Expecting -f <infile>\n")
        printUsage()
        os.Exit(5)
    }
    // The flags package doesn't seem to provide a required argument option.
    // Try to detect missing flag arguments.
    // If an argument is another flag, that's an error.
    if strings.HasPrefix("flags.f", "-") {
        // Expecting -f <infile> expecting an argument but has been followed by another flag.
        fmt.Fprintf(os.Stderr, "Expecting -f <infile> (not: %s)\n", flags.f)
        printUsage()
        os.Exit(6)
    }
	gotablesFileName = flags.f
	// where(fmt.Sprintf("gotablesFileName = %s\n", gotablesFileName))

	if flags.g {
		// Input file of relations tables to be used as a GraphQL schema.
	    // Compulsory flag.
	    if flags.r == "" {
	        // -r has been followed by an empty argument.
	        fmt.Fprintf(os.Stderr, "Expecting -r <infile>\n")
	        printUsage()
	        os.Exit(5)
	    }
	    // The flags package doesn't seem to provide a required argument option.
	    // Try to detect missing flag arguments.
	    // If an argument is another flag, that's an error.
	    if strings.HasPrefix("flags.r", "-") {
	        // Expecting -r <infile> expecting an argument but has been followed by another flag.
	        fmt.Fprintf(os.Stderr, "Expecting -r <infile> (not: %s)\n", flags.r)
	        printUsage()
	        os.Exit(6)
	    }
		relationsFileName = flags.f
	where(fmt.Sprintf("relationsFileName = %s\n", relationsFileName))
	}

	// Namespace
    // Compulsory flag.
    if flags.n == "" {
        // -n has been followed by an empty argument.
        fmt.Fprintf(os.Stderr, "Expecting -n <namespace>\n")
        printUsage()
        os.Exit(7)
    }
    // The flags package doesn't seem to provide a required argument option.
    // Try to detect missing flag arguments.
    // If an argument is another flag, that's an error.
    if strings.HasPrefix("flags.n", "-") {
        // -n expecting an argument but has been followed by another flag.
        fmt.Fprintf(os.Stderr, "Expecting -n <namespace> (not: %s)\n", flags.n)
        printUsage()
        os.Exit(8)
    }
	nameSpace = flags.n
	// nameSpace has the same validity criteria as gotables col names and table names.
	isValid, _ := gotables.IsValidColName(nameSpace)
	if !isValid {
        fmt.Fprintf(os.Stderr, "Error: non-alpha-numeric-underscore chars in -n <namespace>: %q\n", flags.n)
        fmt.Fprintf(os.Stderr, "Note:  <namespace> is not a file or dir name. Though it is used in file and dir names.\n")
        printUsage()
        os.Exit(9)
	}

	if flags.m {
		mutableFlag = "--gen-mutable"	// Generate additional non-const accessors to mutate FlatBuffers in-place.
	}

	outDir     = "../" + nameSpace	// Package level, where nameSpace is package name.
	outDirMain = outDir + "_main"

	// Package
    // Compulsory flag.
    if flags.p == "" {
        // -p has been followed by an empty argument.
        fmt.Fprintf(os.Stderr, "Expecting -p <package>\n")
        printUsage()
        os.Exit(10)
    }
    // The flags package doesn't seem to provide a required argument option.
    // Try to detect missing flag arguments.
    // If an argument is another flag, that's an error.
    if strings.HasPrefix("flags.p", "-") {
        // -p expecting an argument but has been followed by another flag.
        fmt.Fprintf(os.Stderr, "Expecting -p <package> (not: %s)\n", flags.p)
        printUsage()
        os.Exit(11)
    }
	packageName = flags.p
	// Package name must include namespace.
	if !strings.HasSuffix(packageName, nameSpace) {
        fmt.Fprintf(os.Stderr, "package name -p %q must include namespace -n %q\n", packageName, nameSpace)
        printUsage()
        os.Exit(12)
	}
}

func progName() string {
	return filepath.Base(os.Args[0])
}

func printUsage() {
	var usageSlice []string = []string{
		"usage:       ${utilName} [-v] -f <gotables-file> -n <namespace> -p <package>",
		"purpose: (1) Generate a FlatBuffers schema file <namespace>.fbs from a set of tables.",
		"         (2) Generate official Flatbuffers Go code (from <namespace>.fbs) using flatc --go",
		"         (3) Generate additional Go code to read/write these specific table types from gotables objects.",
		"flags:   -f  Input text file containing one or more gotables tables (generates FlatBuffers schema).",
		"             See flattables_sample: https://github.com/urban-wombat/flattables_sample/blob/master/tables.got",
		"             Note: The file need not contain data. Only metadata (names and types) will be used for code generation.",
		"                   If there is data in the input file, it may be used for running tests, benchtests.",
		"                   Exception: the row count of each table is used to dimention table arrays.",
		"         -n  Namespace  Sets <namespace>.fbs, FlatBuffers namespace, TableSet name, schema root_type.",
		"             Note: Generated Go code will be (conveniently) placed adjacently at Go package level.",
		"                   This assumes you are running ${utilName} at package level.",
		"         -p  Package  Sets Go package name. Needs to include Namespace.",
//		"         -m  Mutable  Tells flatc to add mutable methods to its code generation: Mutate...()",
		"types:       Architecture-dependent Go types int and uint are not used. Instead use e.g. int64, uint32, etc.",
		"             Go types not implemented: complex.",
		"names:       Table names are UpperCamelCase, column names are lowerCamelCase, as per the FlatBuffers style guide.",
//		"deprecation: To deprecate a column, append its name with _DEPRECATED_ (warning: deprecation may break tests and old code).",
		"         -v  Verbose",
		"         -h  Help",
		"sample:      This sample assumes package name \"github.com/urban-wombat/flattables_sample\".",
		"             Make a Go package dir: $ mkdir flattables_sample",
		"             $ cd flattables_sample",
		"             Create a gotables file: tables.got",
		"             $ ${utilName}           -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample",
//		"             $ ${utilName}           -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample -m",
		"             $ go run ${utilName}.go -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample",
	}

	var usageString string
	for i := 0; i < len(usageSlice); i++ {
		usageString += usageSlice[i] + "\n"
	}

	// For debugging or new code, conditionally add provisional command line examples under development.
	user, _ := user.Current()
	if user.Username == "Malcolm-PC\\Malcolm" {
		// We are testing. Provide a useful sample. Does not appear in final product.
		usageString += "additional commands in development mode:\n"
		usageString += "             $ go run ${utilName}.go -v -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample\n"
		usageString += "             $ go run ${utilName}.go -v -G -f ../graphql_sample/tables.gt -n graphql_sample -p github.com/urban-wombat/graphql_sample\n"
		usageString += "             $ go install ${utilName}.go\n"
		usageString += "             $ ${utilName}           -v -G -f ../graphql_sample/tables.gt -n graphql_sample -p github.com/urban-wombat/graphql_sample\n"
	}

	usageString = strings.Replace(usageString, "${utilName}", utilName, -1)

	fmt.Fprintf(os.Stderr, "%s\n", usageString)
}

func main() {

	if strings.Contains(os.Args[0], "flattablesc") {
		utilName = "flattablesc"
	} else if strings.Contains(os.Args[0], "graphqlc") {
		utilName = "graphqlc"
	} else {
		fmt.Fprintf(os.Stderr, `expecting to be called something like "flattablesc" or "graphqlc", not %q`, os.Args[0])
		os.Exit(2)
	}

	if len(os.Args) == 1 {
		// No args.
		fmt.Fprintf(os.Stderr, "%s expects at least 1 argument\n", utilName)
		printUsage()
		os.Exit(2)
	}

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Expecting -f <infile> containing one or more gotables tables\n")
		printUsage()
		os.Exit(13)
	}

	flag.Usage = printUsage	// Override the default flag.Usage variable.
	initFlags()

	if flags.v { fmt.Printf(" (1) Reading gotables file: %s\n", gotablesFileName) }
	tableSet, err := gotables.NewTableSetFromFile(gotablesFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		printUsage()
		os.Exit(14)
	}

	if flags.v { fmt.Printf(" (2) Setting gotables.TableSet name to %q (from -n %s)\n", nameSpace, nameSpace) }
	tableSet.SetName(nameSpace)
	tableSet.SetFileName(gotablesFileName)

	if flags.v { fmt.Printf(" (3) Setting package name to %q (from -p %s)\n", packageName, packageName) }

	if !pathExists(outDir) {
		if flags.v { fmt.Printf(" (4) Creating dir <outdir>:     %s\n", outDir) }
		permissions := 0777
		err = os.Mkdir(outDir, os.FileMode(permissions))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(15)
		}
	} else {
		if flags.v { fmt.Printf(" (4) Dir <outdir>     already exists (okay) %s\n", outDir) }
	}

	if !pathExists(outDirMain) {
		if flags.v { fmt.Printf(" (5) Creating dir <outdirmain>: %s\n", outDirMain) }
		permissions := 0777
		err = os.Mkdir(outDirMain, os.FileMode(permissions))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(16)
		}
	} else {
		if flags.v { fmt.Printf(" (5) Dir <outdirmain> already exists (okay) %s\n", outDirMain) }
	}

	// Must be called before flattables.InitTemplateInfo()
	err = flattables.DeleteEmptyTables(tableSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(17)
	}

	// Template info for ALL the templates.
	if flags.v { fmt.Printf(" (6) Preparing tables for schema generation ...\n")  }
	var tablesTemplateInfo flattables.TablesTemplateInfo
	tablesTemplateInfo, err = flattables.InitTablesTemplateInfo(tableSet, packageName, flags.b, flags.g)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(18)
	}

/*
	if flags.g {
		// Template info for GraphQL the templates.
		if flags.v { fmt.Printf(" (6) Preparing tables for GraphQL schema generation ...\n")  }
		var relationsTemplateInfo flattables.RelationsTemplateInfo
		relationsTemplateInfo, err = flattables.InitRelationsTemplateInfo(tableSet, packageName, flags.b, flags.g)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(18)
		}
	}
*/

	if flags.b {
		var tableCount int = tableSet.TableCount()
		if flags.v { fmt.Printf("     Adding gotables tables  to FlatBuffers schema: (%d table%s):-\n", tableCount, plural(tableCount)) }

		flatBuffersSchemaFileName := outDir + "/" + nameSpace + ".fbs"
		tablesTemplateInfo.GeneratedFile = filepath.Base(flatBuffersSchemaFileName)
		if flags.v {
			fmt.Printf(" (7) Generating  FlatBuffers schema from gotables file %s ...\n", gotablesFileName)
			fmt.Printf("     Generating: %s\n", flatBuffersSchemaFileName)
		}
		flatBuffersSchema, err := flattables.FlatBuffersSchemaFromTableSet(tablesTemplateInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(19)
		}
	
		flatBuffersSchema = flattables.RemoveExcessTabsAndNewLines(flatBuffersSchema)
	
		err = ioutil.WriteFile(flatBuffersSchemaFileName, []byte(flatBuffersSchema), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(20)
		}

		// Note: each arg part needs to be passed to exec.Command separately.
		executable := "flatc"
		goFlag := "--go"
		outFlag := "-o"		// for flatc
		outDirFlatC := ".."	// flatc creates a subdir under this.
		if flags.v { fmt.Printf(" (8) From FlatBuffers schema %s generating standard generic FlatBuffers Go code:\n", flatBuffersSchemaFileName) }
		if flags.m {	// Mutable
			if flags.v { fmt.Printf("     $ %s %s %s %s %s %s\n", executable, goFlag, mutableFlag, outFlag, outDirFlatC, flatBuffersSchemaFileName) }
		} else {
			if flags.v { fmt.Printf("     $ %s %s %s %s %s\n",    executable, goFlag,              outFlag, outDirFlatC, flatBuffersSchemaFileName) }
		}
		var cmd *exec.Cmd
		if flags.m {	// Mutable
			cmd = exec.Command(executable, goFlag, mutableFlag, outFlag, outDirFlatC, flatBuffersSchemaFileName)
		} else {
			cmd = exec.Command(executable, goFlag,              outFlag, outDirFlatC, flatBuffersSchemaFileName)
		}
		var out bytes.Buffer
		cmd.Stdout = &out
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

	if flags.g {
		graphqlSchemaFileName := outDir + "/" + nameSpace + "_schema.graphql"
		tablesTemplateInfo.GeneratedFile = filepath.Base(graphqlSchemaFileName)
		if flags.v {
			fmt.Printf("     Generating  GraphQL schema from gotables file %s ...\n", gotablesFileName)
			fmt.Printf("     Generating: %s\n", graphqlSchemaFileName)
		}
		graphqlSchema, err := flattables.GraphQLSchemaFromTableSet(tablesTemplateInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(23)
		}

		graphqlSchema = flattables.RemoveExcessTabsAndNewLines(graphqlSchema)

		err = ioutil.WriteFile(graphqlSchemaFileName, []byte(graphqlSchema), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(24)
		}
	}
	
	// GenerateAll() chooses between flatbuffers and/or graphql internally.
	if flags.v { fmt.Printf(" (*) Generating user Go code ...\n") }
	err = flattables.GenerateAll(nameSpace, flags.v, flags.b, flags.g)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(22)
	}

	fmt.Println("DONE")
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
