package flattables

// This is a library of helper functions for utility: flattablesc

// See: https://github.com/urban-wombat/flattables#flattables-is-a-simplified-tabular-subset-of-flatbuffers

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/urban-wombat/gotables"
	"github.com/urban-wombat/util"
)

/*
Copyright (c) 2017 Malcolm Gorman

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

func init() {
	log.SetFlags(log.Lshortfile) // For var where
}

// var where = log.Print

/*
	Note: the following are synonyms:-
	synonyms: graphql GraphQL
	synonyms: flattables FlatTables flatbuffers FlatBuffers
*/

// FlatBuffers schema types: bool | byte | ubyte | short | ushort | int | uint | float | long | ulong | double | string
// From: https://google.github.io/flatbuffers/flatbuffers_grammar.html

// Built-in scalar types are:
//  8 bit: byte, ubyte, bool
// 16 bit: short, ushort
// 32 bit: int, uint, float
// 64 bit: long, ulong, double
// From: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html

var goToFlatBuffersTypes = map[string]string{
	"bool":    "bool",
	"int8":    "byte", // Signed.
	"int16":   "short",
	"int32":   "int", // (Go rune is an alias for Go int32. For future reference.)
	"int64":   "long",
	"byte":    "ubyte",   // Unsigned. Go byte is an alias for Go uint8.
	"[]byte":  "[ubyte]", // Unsigned. Go byte is an alias for Go uint8. NOTE: This [ubyte] IS NOT IMPLEMENTED IN FLATTABLES!
	"uint8":   "ubyte",
	"uint16":  "ushort",
	"uint32":  "uint",
	"uint64":  "ulong",
	"float32": "float",
	"float64": "double",
	"string":  "string",
	//	"int":     "long",	// Assume largest int size:  64 bit. NO, DON'T DO THIS AUTOMATICALLY. REQUIRE USER DECISION.
	//	"uint":    "ulong",	// Assume largest uint size: 64 bit. NO, DON'T DO THIS AUTOMATICALLY. REQUIRE USER DECISION.
}

var goToGraphQLTypes = map[string]string{
	"string":  "String",
	"bool":    "Boolean",
	"int32":   "Int",
	"float64": "Float",
	//	"int64":   "long",	// 64‐bit numeric non‐fractional value. Currently not implemented by Prisma.
	/*
	   	"int8":    "byte",	// Signed.
	   	"int16":   "short",
	   	"byte":    "ubyte",	// Unsigned. Go byte is an alias for Go uint8.
	   	"[]byte":  "[ubyte]",	// Unsigned. Go byte is an alias for Go uint8. NOTE: This [ubyte] IS NOT IMPLEMENTED IN FLATTABLES!
	   	"uint8":   "ubyte",
	   	"uint16":  "ushort",
	   	"uint32":  "uint",
	   	"uint64":  "ulong",
	   	"float32": "float",
	   //	"int":     "long",	// Assume largest int size:  64 bit. NO, DON'T DO THIS AUTOMATICALLY. REQUIRE USER DECISION.
	   //	"uint":    "ulong",	// Assume largest uint size: 64 bit. NO, DON'T DO THIS AUTOMATICALLY. REQUIRE USER DECISION.
	*/
}

var goFlatBuffersScalarTypes = map[string]string{
	"bool":    "bool", // Scalar from FlatBuffers point of view.
	"int8":    "byte", // Signed.
	"int16":   "short",
	"int32":   "int", // (Go rune is an alias for Go int32. For future reference.)
	"int64":   "long",
	"byte":    "ubyte", // Unsigned. Go byte is an alias for Go uint8.
	"uint8":   "ubyte",
	"uint16":  "ushort",
	"uint32":  "uint",
	"uint64":  "ulong",
	"float32": "float",
	"float64": "double",
}

const deprecated = "_deprecated_"

func schemaType(colType string) (string, error) {
	schemaType, exists := goToFlatBuffersTypes[colType]
	if exists {
		return schemaType, nil
	} else {
		// Build a useful error message.
		var suggestChangeTypeTo string
		switch colType {
		case "int":
			suggestChangeTypeTo = "int32 or int64"
		case "uint":
			suggestChangeTypeTo = "uint32 or uint64"
		default:
			return "", fmt.Errorf("no FlatBuffers-compatible Go type suggestion for Go type: %s", colType)
		}
		return "", fmt.Errorf("no FlatBuffers type available for Go type: %s (suggest change it to Go type: %s)",
			colType, suggestChangeTypeTo)
	}
}

func isDeprecated(colName string) bool {
	return strings.Contains(colName, deprecated)
}

func IsFlatBuffersScalar(colType string) bool {
	_, exists := goFlatBuffersScalarTypes[colType]
	return exists
}

// This is possibly unused.
func isScalar(table *gotables.Table, colName string) bool {
	colType, err := table.ColType(colName)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s [%s].%s ERROR: %v\n", util.FuncName(), table.Name(), colName, err)
		return false
	}

	isNumeric, err := gotables.IsNumericColType(colType)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s [%s].%s ERROR: %v\n", util.FuncName(), table.Name(), colName, err)
		return false
	}

	return isNumeric || colType == "bool"
}

func indentText(indent string, text string) string {
	var indentedText string = ""
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		indentedText += fmt.Sprintf("%s%s\n", indent, scanner.Text())
	}
	return indentedText
}

func FlatBuffersSchemaFromTableSet(tablesTemplateInfo TablesTemplateInfoType) (string, error) {

	var err error

	var buf *bytes.Buffer = bytes.NewBufferString("")

	const FlatBuffersSchemaFromTableSetTemplateFile = "../flattables/FlatBuffersSchema.template"
	// Use the file name as the template name so that file name appears in error output.
	// We still use the file name for diagnostics, even though the template is now embedded in flattables_templates.go
	// Although no longer used to OPEN the file, it is still used in err to locate the original (non-embedded) file source.
	var tplate *template.Template = template.New(FlatBuffersSchemaFromTableSetTemplateFile)

	// Add a user-defined function to schema tplate.
	tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tplate.Funcs(template.FuncMap{"yearRangeFromFirstYear": yearRangeFromFirstYear})

	// From embedded template in flattables_templates.go
	var data []byte = FlatBuffersSchema_template
	/*
		NOTE: This []byte slice may be what egonelbre is referring to when he says:
		This https://github.com/urban-wombat/flattables/blob/master/flattables.go#L202 breaks with unicode.
	*/

	tplate, err = tplate.Parse(string(data))
	if err != nil {
		return "", err
	}

	err = tplate.Execute(buf, tablesTemplateInfo)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GraphQLSchemaFromTableSet(tablesTemplateInfo TablesTemplateInfoType) (string, error) {

	var err error

	var buf *bytes.Buffer = bytes.NewBufferString("")

	const GraphQLSchemaFromTableSetTemplateFile = "../graphql/GraphQLSchema.template"
	// Use the file name as the template name so that file name appears in error output.
	var tplate *template.Template = template.New(GraphQLSchemaFromTableSetTemplateFile)

	// Add a user-defined function to schema tplate.
	tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tplate.Funcs(template.FuncMap{"yearRangeFromFirstYear": yearRangeFromFirstYear})

	// From embedded template in flattables_templates.go
	var data []byte = GraphQLSchema_template

	tplate, err = tplate.Parse(string(data))
	if err != nil {
		return "", err
	}

	err = tplate.Execute(buf, tablesTemplateInfo)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func startsWithLowerCase(s string) bool {
	if len(s) > 0 {
		rune0 := rune(s[0])
		return unicode.IsLower(rune0)
	} else {
		return false
	}
}

func startsWithUpperCase(s string) bool {
	if len(s) > 0 {
		rune0 := rune(s[0])
		return unicode.IsUpper(rune0)
	} else {
		return false
	}
}

func firstCharToUpper(s string) string {
	var upper string
	if len(s) > 0 {
		rune0 := rune(s[0])
		upper = string(unicode.ToUpper(rune0)) + s[1:]
	} else {
		upper = ""
	}
	return upper
}

func firstCharToLower(s string) string {
	var lower string
	if len(s) > 0 {
		rune0 := rune(s[0])
		lower = string(unicode.ToLower(rune0)) + s[1:]
	} else {
		lower = ""
	}
	return lower
}

func tableName(table *gotables.Table) string {
	return "// " + table.Name()
}

func rowCount(table *gotables.Table) int {
	return table.RowCount()
}

// Information specific to each generated function.
type GenerationInfo struct {
	TemplateType string
	FuncName     string   // Used as basename of *.template and *.go files. Not always a function name.
	Imports      []string // imports for this template.
	TemplateText []byte
}

var generations = []GenerationInfo{
	{TemplateType: "flattables",
		FuncName:     "README",
		TemplateText: README_template,
		Imports:      []string{},
	},
	{TemplateType: "flattables",
		FuncName:     "test", // Not really a function name.
		TemplateText: test_template,
		Imports: []string{
			`"bytes"`,
			`"fmt"`,
			`"github.com/urban-wombat/gotables"`,
			`"reflect"`,
			`"testing"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "helpers",
		TemplateText: helpers_template,
		Imports: []string{
			`"path/filepath"`,
			`"runtime"`,
			`"strings"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "NewFlatBuffersFromSlice",
		TemplateText: NewFlatBuffersFromSlice_template,
		Imports: []string{
			`flatbuffers "github.com/google/flatbuffers/go"`,
			`"fmt"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "NewFlatBuffersFromTableSet",
		TemplateText: NewFlatBuffersFromTableSet_template,
		Imports: []string{
			`flatbuffers "github.com/google/flatbuffers/go"`,
			`"github.com/urban-wombat/gotables"`,
			`"fmt"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "NewSliceFromFlatBuffers",
		TemplateText: NewSliceFromFlatBuffers_template,
		Imports: []string{
			`"fmt"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "NewTableSetFromFlatBuffers",
		TemplateText: NewTableSetFromFlatBuffers_template,
		Imports: []string{
			`"github.com/urban-wombat/gotables"`,
			`"fmt"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "OldSliceFromFlatBuffers",
		TemplateText: OldSliceFromFlatBuffers_template,
		Imports: []string{
			`"fmt"`,
		},
	},
	/*
	   	{	TemplateType: "graphql",
	   		FuncName: "polyglot_main",	// From: https://www.thepolyglotdeveloper.com/2018/05/getting-started-graphql-golang
	   		Imports: []string {
	   			`"encoding/json"`,
	   			`"fmt"`,
	   			`"net/http"`,
	   //			`"github.com/urban-wombat/gotables"`,
	   			`"github.com/graphql-go/graphql"`,
	   		},
	   	},
	*/
	/*
	   	{	TemplateType: "graphql",
	   		FuncName: "hafiz_main",	// From: https://wehavefaces.net/learn-golang-graphql-relay-1-e59ea174a902
	   		Imports: []string {
	   //			`"encoding/json"`,
	   //			`"fmt"`,
	   			`"net/http"`,
	   //			`"github.com/urban-wombat/gotables"`,
	   			`"github.com/graphql-go/graphql"`,
	   			`"github.com/graphql-go/graphql-go-handler"`,
	   		},
	   	},
	*/
	/*
	   	{	TemplateType: "graphql",
	   		FuncName: "raboy_main",	// From: https://wehavefaces.net/learn-golang-graphql-relay-1-e59ea174a902
	   		TemplateText: raboy_main_template,
	   		Imports: []string {
	   			`"encoding/json"`,
	   			`"fmt"`,
	   //			`"log"`,
	   			`"net/http"`,
	   			`"github.com/graphql-go/graphql"`,
	   //			`"github.com/urban-wombat/gotables"`,
	   		},
	   	},
	*/
	{TemplateType: "graphql",
		FuncName:     "gqlgen_main", // From: https://github.com/99designs/gqlgen
		TemplateText: gglgen_main_template,
		Imports: []string{
			`"encoding/json"`,
			`"fmt"`,
			//			`"log"`,
			`"net/http"`,
			`"github.com/graphql-go/graphql"`,
			//			`"github.com/urban-wombat/gotables"`,
		},
	},
	{TemplateType: "flattables",
		FuncName:     "main", // Not really a function name.
		TemplateText: main_template,
		Imports: []string{
			`"github.com/urban-wombat/gotables"`,
			`"fmt"`,
		},
	},
}

// func GenerateAll(tablesTemplateInfo TablesTemplateInfoType, nameSpace string, verbose bool, dryRun bool, genFlatBuffers bool, genGraphQL bool) error {
func GenerateAll(tablesTemplateInfo TablesTemplateInfoType, verbose bool, dryRun bool, genFlatBuffers bool, genGraphQL bool) error {
	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	for _, generation := range generations {
		if (generation.TemplateType == "flattables" && genFlatBuffers) || (generation.TemplateType == "graphql" && genGraphQL) {
			// tablesTemplateInfo is global.
			//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
			//			err := generateGoCodeFromTemplate(generation, tablesTemplateInfo, nameSpace, verbose, dryRun)
			err := generateGoCodeFromTemplate(generation, tablesTemplateInfo, verbose, dryRun)
			if err != nil {
				return err
			}
			//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
		}
	}

	return nil
}

// func generateGoCodeFromTemplate(generationInfo GenerationInfo, tablesTemplateInfo TablesTemplateInfoType, nameSpace string, verbose bool, dryRun bool) (err error) {
func generateGoCodeFromTemplate(generationInfo GenerationInfo, tablesTemplateInfo TablesTemplateInfoType, verbose bool, dryRun bool) (err error) {
	//gotables.PrintCaller()

	var templateFile string
	var outDir string
	var generatedFile string

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	// Calculate input template file name.
	// Although no longer used to OPEN the file, it is still used in err to locate the original (non-embedded) file source.
	templateFile = fmt.Sprintf("../%s/%s.template", generationInfo.TemplateType, generationInfo.FuncName)

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	/*
	   where("USING")
	   where(tablesTemplateInfo.OutDirMainAbsolute)
	   where(tablesTemplateInfo.OutDir)
	*/
	// Calculate output dir name.
	if strings.Contains(generationInfo.FuncName, "main") {
		//		outDir = "../" + nameSpace + "_main"	// main is in its own directory
		outDir = tablesTemplateInfo.OutDirMainAbsolute // main is in its own directory
		//where(fmt.Sprintf("outDir = %s", outDir))
	} else {
		//		outDir = "../" + nameSpace				// put it in with all the rest
		outDir = tablesTemplateInfo.OutDirAbsolute // put it in with all the rest
		//where(fmt.Sprintf("outDir = %s", outDir))
	}

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	// Calculate output file name.
	switch generationInfo.FuncName {
	case "README": // README is a markdown .md file
		generatedFile = outDir + "/" + generationInfo.FuncName + ".md"
	default: // For both function files and main files. Retain FuncName for main functions to differentiate multiple mains.
		generatedFile = outDir + "/" + tablesTemplateInfo.NameSpace + "_" + generationInfo.FuncName + ".go"
	}
	if verbose {
		fmt.Printf("     Generating: %-12s %s\n", fmt.Sprintf("(%s)", generationInfo.TemplateType), generatedFile)
	}

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	tablesTemplateInfo.SchemaFileName = outDir + "/" + tablesTemplateInfo.NameSpace + ".fbs"
	tablesTemplateInfo.GeneratedFile = generatedFile
	tablesTemplateInfo.FuncName = generationInfo.FuncName
	tablesTemplateInfo.Imports = generationInfo.Imports

	/*
	   fmt.Printf("\n")
	   fmt.Printf("%#v\n", generationInfo)
	   fmt.Printf("templateFile = %s\n", templateFile)
	   fmt.Printf("outDir = %s\n", outDir)
	   fmt.Printf("generatedFile = %s\n", generatedFile)
	   fmt.Printf("\n")
	*/

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	var stringBuffer *bytes.Buffer = bytes.NewBufferString("")

	// Use the file name as the template name so that file name appears in error output.
	// We still use the file name for diagnostics, even though the template is now embedded in flattables_templates.go
	// Although no longer used to OPEN the file, it is still used in err to locate the original (non-embedded) file source.
	var tplate *template.Template = template.New(templateFile)

	// Add functions.
	tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tplate.Funcs(template.FuncMap{"firstCharToLower": firstCharToLower})
	tplate.Funcs(template.FuncMap{"rowCount": rowCount})
	tplate.Funcs(template.FuncMap{"yearRangeFromFirstYear": yearRangeFromFirstYear})

	/*
		// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
		templateText, err := ioutil.ReadFile(templateFile)
		if err != nil { return }
	*/

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	// Template from embedded templates in flattables_templates.go
	var templateText []byte = generationInfo.TemplateText

	tplate, err = tplate.Parse(string(templateText))
	if err != nil {
		return
	}

	err = tplate.Execute(stringBuffer, tablesTemplateInfo)
	if err != nil {
		return
	}

	var goCode string = stringBuffer.String()

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	// The code generator has a lot of quirks (such as extra lines and tabs) which are hard to
	// eliminate within the templates themselves. Use gofmt to tidy up Go code.

	// We don't want gofmt to mess with non-Go files (such as README.md which it crunches).
	if strings.HasSuffix(generatedFile, ".go") {
		goCode = RemoveExcessTabsAndNewLines(goCode) // handwritten formatter
		//		goCode, err = util.GoFmtProgramString(goCode)	// Run the gofmt command on input string goCode
		var goCodeBytes []byte
		goCodeBytes, err = format.Source([]byte(goCode))
		if err != nil {
			// gofmt is better, but make do with my handwritten formatter if gofmt is unavailable.
			// Just in case the gofmt command is unavailable or inaccessible on this system.
			_, _ = fmt.Fprintln(os.Stderr, "     Cannot access gofmt utility right now. Using handwritten formatter instead.")
		}
		goCode = string(goCodeBytes)
	}

	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))
	if dryRun {
		fmt.Printf(" *** -d dry-run: Would have written file: %s\n", generatedFile)
	} else {
		err = ioutil.WriteFile(generatedFile, []byte(goCode), 0644)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(30)
		}
	}
	//where(fmt.Sprintf("WHAT? %s", tablesTemplateInfo.OutDirMainAbsolute))

	return
}

// Compilation will fail if a user inadvertently uses a Go key word as a name.
var goKeyWordsDEPRECATED = map[string]string{
	"break":       "break",
	"default":     "default",
	"func":        "func",
	"interface":   "interface",
	"select":      "select",
	"case":        "case",
	"defer":       "defer",
	"go":          "go",
	"map":         "map",
	"struct":      "struct",
	"chan":        "chan",
	"else":        "else",
	"goto":        "goto",
	"package":     "package",
	"switch":      "switch",
	"const":       "const",
	"fallthrough": "fallthrough",
	"if":          "if",
	"range":       "range",
	"type":        "type",
	"continue":    "continue",
	"for":         "for",
	"import":      "import",
	"return":      "return",
	"var":         "var",
}

func isGoKeywordDEPRECATED(name string) bool {
	nameLower := strings.ToLower(name)
	_, exists := goKeyWordsDEPRECATED[nameLower]
	return exists
}

// See https://www.reddit.com/r/golang/comments/9umtp2/beta_release_of_flattables_go_flatbuffers/e95iffn/?context=3
// This avoids manually providing a lookup map.
func isGoKeyword(name string) bool {
	nameLower := strings.ToLower(name)
	var isKeyword bool = token.Lookup(name).IsKeyword()
	return isKeyword
}

// Could be tricky if a user inadvertently uses a word used in FlatBuffers schemas.
var flatBuffersOrFlatTablesKeyWords = map[string]string{
	"flattables": "flattables", // FlatTables is used as the root table name and root_type.
	"table":      "table",
	"namespace":  "namespace",
	"root_type":  "root_type",
	"ubyte":      "ubyte",
	"float":      "float",
	"long":       "long",
	"ulong":      "ulong",
	"short":      "short",
	"ushort":     "ushort",
	"double":     "double",
	"enum":       "enum",
	"union":      "union",
	"include":    "include",
}

func isFlatBuffersOrFlatTablesKeyWord(name string) bool {
	name = strings.ToLower(name)
	_, exists := flatBuffersOrFlatTablesKeyWords[name]
	return exists
}

type ColInfo struct {
	ColName      string
	ColType      string
	FbsType      string
	ColIndex     int
	IsScalar     bool // FlatBuffers Scalar includes bool
	IsString     bool
	IsBool       bool
	IsDeprecated bool
}

type Row []string

type TableInfo struct {
	Table      *gotables.Table
	TableIndex int
	TableName  string
	RowCount   int
	ColCount   int
	Cols       []ColInfo
	Rows       []Row
	ColNames   []string
	ColTypes   []string
	// These are for GraphQL:
	ObjectType  string // e.g. "type", "input"
	ColTypeGQL  string
	FieldName   []string
	FieldType   []string // e.g. "String", "Int" (Go int32), "Float" (Go float64), "Boolean", "ID" (string), and user defined types.
	NonNullable []bool
}

type TablesTemplateInfoType struct {
	GeneratedDateFromFile string
	GeneratedFromFile     string
	UsingCommand          string
	UsingCommandMinusG    string
	NameSpace             string // Included in PackageName.
	PackageName           string // Includes NameSpace
	Year                  string
	OutDirAbsolute        string
	OutDirMainAbsolute    string
	SchemaFileName        string
	GeneratedFile         string
	FuncName              string
	Imports               []string
	//	GotablesFileName string	// We want to replace this with the following TWO file names.
	GotablesFileNameAbsolute string
	TableSetMetadata         string
	TableSetData             string
	Tables                   []TableInfo
}

var TablesTemplateInfo TablesTemplateInfoType

func (tablesTemplateInfo TablesTemplateInfoType) Name(tableIndex int) string {
	return tablesTemplateInfo.Tables[0].Table.Name()
}

func DeleteEmptyTables(tableSet *gotables.TableSet) error {

	for tableIndex := tableSet.TableCount() - 1; tableIndex >= 0; tableIndex-- {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil {
			return err
		}

		if table.ColCount() == 0 {
			err = tableSet.DeleteTableByTableIndex(tableIndex)
			if err != nil {
				return err
			}
			return fmt.Errorf("table has zero cols: [%s] (remove or populate)", table.Name())
		}
	}

	return nil
}

// Assumes flattables.RemoveEmptyTables() has been called first.
func InitTablesTemplateInfo(tableSet *gotables.TableSet, packageName string, genFlatBuffers bool, genGraphQL bool) (TablesTemplateInfoType, error) {

	var emptyTemplateInfo TablesTemplateInfoType
	var tablesTemplateInfo TablesTemplateInfoType

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil {
			return emptyTemplateInfo, err
		}

		if table.ColCount() > 0 {
			if genFlatBuffers && genGraphQL {
				_, _ = fmt.Fprintf(os.Stderr, "     Adding gotables table %d to FlatBuffers and GraphQL schema: [%s] \n", tableIndex, table.Name())
			} else if genFlatBuffers {
				_, _ = fmt.Fprintf(os.Stderr, "     Adding gotables table %d to FlatBuffers schema: [%s] \n", tableIndex, table.Name())
			} else if genGraphQL {
				_, _ = fmt.Fprintf(os.Stderr, "     Adding gotables table %d to GraphQL schema: [%s] \n", tableIndex, table.Name())
			}
		} else {
			// Skip tables with zero cols.
			return emptyTemplateInfo, fmt.Errorf("--- FlatTables: table [%s] has no col", table.Name())
		}

		if genFlatBuffers && !genGraphQL {
			if startsWithLowerCase(table.Name()) {
				// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
				return emptyTemplateInfo, fmt.Errorf("the FlatBuffers style guide requires UpperCamelCase table names. Rename [%s] to [%s]",
					table.Name(), firstCharToUpper(table.Name()))
			}

			if isGoKeyword(table.Name()) {
				return emptyTemplateInfo,
					fmt.Errorf("cannot use a Go key word as a table name, even if it's upper case. Rename [%s]", table.Name())
			}

			if isFlatBuffersOrFlatTablesKeyWord(table.Name()) {
				return emptyTemplateInfo,
					fmt.Errorf("cannot use a FlatBuffers or FlatTables key word as a table name, even if it's merely similar. Rename [%s]",
						table.Name())
			}

			// I don't see documentation on this, but undescores in field names affect code generation.
			if strings.ContainsRune(table.Name(), '_') {
				return emptyTemplateInfo,
					fmt.Errorf("cannot use underscores '_' in table names. Rename [%s]", table.Name())
			}
		}

		tables[tableIndex].Table = table // An array of Table accessible as .Tables

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColNameByColIndex(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}

			// Relax FlatBuffers requirements when generating GraphQL, which has a conflicting naming style.
			// i.e. if generating FlatBuffers and NOT GraphQL, enforce FlatBuffers style guide.
			if genFlatBuffers && !genGraphQL {
				if startsWithUpperCase(colName) {
					// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
					return emptyTemplateInfo, fmt.Errorf("the FlatBuffers style guide requires lowerCamelCase field names. In table [%s] rename %s to %s",
						table.Name(), colName, firstCharToLower(colName))
				}

				if isGoKeyword(colName) {
					return emptyTemplateInfo, fmt.Errorf("cannot use a Go key word as a col name, even if it's upper case. Rename: %s", colName)
				}

				if isFlatBuffersOrFlatTablesKeyWord(colName) {
					return emptyTemplateInfo,
						fmt.Errorf("cannot use a FlatBuffers or FlatTables key word as a col name, even if it's merely similar. Rename: %s", colName)
				}

				// I don't see documentation on this, but undescores in field names affect code generation.
				if strings.ContainsRune(colName, '_') {
					return emptyTemplateInfo,
						fmt.Errorf("cannot use underscores '_' in col names. Rename %s", colName)
				}
			}

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}

			cols[colIndex].IsDeprecated = isDeprecated(colName)
			if cols[colIndex].IsDeprecated {
				// Restore the col name by removing _DEPRECATED_ indicator.
				colName = strings.Replace(colName, deprecated, "", 1)
				_, _ = fmt.Fprintf(os.Stderr, "*** FlatTables: Tagged table [%s] column %q is deprecated\n", table.Name(), colName)
			}

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].FbsType, err = schemaType(colType)
			if err != nil {
				return emptyTemplateInfo, err
			}
			cols[colIndex].ColIndex = colIndex
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType) // FlatBuffers Scalar includes bool
			cols[colIndex].IsString = colType == "string"
			cols[colIndex].IsBool = colType == "bool"
		}

		// Populate Rows with a string representation of each table cell.
		var rows []Row = make([]Row, table.RowCount())
		// where(fmt.Sprintf("RowCount = %d", table.RowCount()))
		for rowIndex := 0; rowIndex < table.RowCount(); rowIndex++ {
			var row []string = make([]string, table.ColCount())
			for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
				var cell string
				cell, err = table.GetValAsStringByColIndex(colIndex, rowIndex)
				if err != nil {
					return emptyTemplateInfo, err
				}
				var isStringType bool
				isStringType, err = table.IsColTypeByColIndex(colIndex, "string")
				if err != nil {
					return emptyTemplateInfo, err
				}
				if isStringType {
					cell = fmt.Sprintf("%q", cell) // Add delimiters.
				}
				row[colIndex] = cell
			}
			rows[rowIndex] = row
			// where(fmt.Sprintf("row[%d] = %v", rowIndex, rows[rowIndex]))
		}

		var colNames []string = make([]string, table.ColCount())
		var colTypes []string = make([]string, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColName(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}
			colNames[colIndex] = colName

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}
			colTypes[colIndex] = colType
		}

		tables[tableIndex].Cols = cols
		tables[tableIndex].TableIndex = tableIndex
		tables[tableIndex].TableName = table.Name()
		tables[tableIndex].RowCount = table.RowCount()
		tables[tableIndex].ColCount = table.ColCount()
		tables[tableIndex].Rows = rows
		tables[tableIndex].ColNames = colNames
		tables[tableIndex].ColTypes = colTypes
	}

	// Get tableset metadata.
	// Make a copy of the tables and use them as metadata-only.
	// We end up with 2 instances of TableSet:-
	// (1) tableSet which contains data.            Is accessible in templates as: .Tables           (an array of Table)
	// (2) metadataTableSet which contains NO data. Is accessible in templates as: .TableSetMetadata (a TableSet)

	const copyRows = false                           // i.e., don't copy rows.
	metadataTableSet, err := tableSet.Copy(copyRows) // Accessible as
	if err != nil {
		return emptyTemplateInfo, err
	}

	for tableIndex := 0; tableIndex < metadataTableSet.TableCount(); tableIndex++ {
		table, err := metadataTableSet.TableByTableIndex(tableIndex)
		if err != nil {
			return emptyTemplateInfo, err
		}

		err = table.SetStructShape(true)
		if err != nil {
			return emptyTemplateInfo, err
		}
	}

	tableSetMetadata := metadataTableSet.String()
	tableSetMetadata = indentText("\t\t", tableSetMetadata)

	tableSetData := tableSet.String()
	//	tableSetData = indentText("\t", tableSetData)

	tablesTemplateInfo = TablesTemplateInfoType{
		GeneratedDateFromFile:    generatedDateFromFile(tableSet),
		GeneratedFromFile:        generatedFromFile(tableSet),
		UsingCommand:             usingCommand(tableSet, packageName),
		UsingCommandMinusG:       usingCommandMinusG(tableSet, packageName),
		GotablesFileNameAbsolute: tableSet.FileName(),
		Year:                     copyrightYear(),
		NameSpace:                tableSet.Name(),
		PackageName:              packageName,
		TableSetMetadata:         tableSetMetadata,
		TableSetData:             tableSetData,
		Tables:                   tables,
	}

	return tablesTemplateInfo, nil
}

// Assumes flattables.RemoveEmptyTables() has been called first.
// THIS NEEDS TO ADD TO, NOT REPLACE, EXISTING TEMPLATE INFORMATION.
func InitRelationsTemplateInfo(tableSet *gotables.TableSet, packageName string, genFlatBuffers bool, genGraphQL bool) (TablesTemplateInfoType, error) {

	var emptyTemplateInfo TablesTemplateInfoType
	var tablesTemplateInfo TablesTemplateInfoType

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil {
			return emptyTemplateInfo, err
		}

		if table.ColCount() > 0 {
			if genFlatBuffers && genGraphQL {
				_, _ = fmt.Fprintf(os.Stderr, "     Adding gotables table %d to FlatBuffers and GraphQL schema: [%s] \n", tableIndex, table.Name())
			} else if genFlatBuffers {
				_, _ = fmt.Fprintf(os.Stderr, "     Adding gotables table %d to FlatBuffers schema: [%s] \n", tableIndex, table.Name())
			} else if genGraphQL {
				_, _ = fmt.Fprintf(os.Stderr, "     Adding gotables table %d to GraphQL schema: [%s] \n", tableIndex, table.Name())
			}
		} else {
			// Skip tables with zero cols.
			return emptyTemplateInfo, fmt.Errorf("--- FlatTables: table [%s] has no col", table.Name())
		}

		if genFlatBuffers && !genGraphQL {
			if startsWithLowerCase(table.Name()) {
				// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
				return emptyTemplateInfo, fmt.Errorf("the FlatBuffers style guide requires UpperCamelCase table names. Rename [%s] to [%s]",
					table.Name(), firstCharToUpper(table.Name()))
			}

			if isGoKeyword(table.Name()) {
				return emptyTemplateInfo,
					fmt.Errorf("cannot use a Go key word as a table name, even if it's upper case. Rename [%s]", table.Name())
			}

			if isFlatBuffersOrFlatTablesKeyWord(table.Name()) {
				return emptyTemplateInfo,
					fmt.Errorf("cannot use a FlatBuffers or FlatTables key word as a table name, even if it's merely similar. Rename [%s]",
						table.Name())
			}

			// I don't see documentation on this, but undescores in field names affect code generation.
			if strings.ContainsRune(table.Name(), '_') {
				return emptyTemplateInfo,
					fmt.Errorf("cannot use underscores '_' in table names. Rename [%s]", table.Name())
			}
		}

		tables[tableIndex].Table = table // An array of Table accessible as .Tables

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColNameByColIndex(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}

			// Relax FlatBuffers requirements when generating GraphQL, which has a conflicting naming style.
			// i.e. if generating FlatBuffers and NOT GraphQL, enforce FlatBuffers style guide.
			if genFlatBuffers && !genGraphQL {
				if startsWithUpperCase(colName) {
					// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
					return emptyTemplateInfo, fmt.Errorf("the FlatBuffers style guide requires lowerCamelCase field names. In table [%s] rename %s to %s",
						table.Name(), colName, firstCharToLower(colName))
				}

				if isGoKeyword(colName) {
					return emptyTemplateInfo, fmt.Errorf("cannot use a Go key word as a col name, even if it's upper case. Rename: %s", colName)
				}

				if isFlatBuffersOrFlatTablesKeyWord(colName) {
					return emptyTemplateInfo,
						fmt.Errorf("cannot use a FlatBuffers or FlatTables key word as a col name, even if it's merely similar. Rename: %s", colName)
				}

				// I don't see documentation on this, but undescores in field names affect code generation.
				if strings.ContainsRune(colName, '_') {
					return emptyTemplateInfo,
						fmt.Errorf("cannot use underscores '_' in col names. Rename %s", colName)
				}
			}

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}

			cols[colIndex].IsDeprecated = isDeprecated(colName)
			if cols[colIndex].IsDeprecated {
				// Restore the col name by removing _DEPRECATED_ indicator.
				colName = strings.Replace(colName, deprecated, "", 1)
				_, _ = fmt.Fprintf(os.Stderr, "*** FlatTables: Tagged table [%s] column %q is deprecated\n", table.Name(), colName)
			}

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].FbsType, err = schemaType(colType)
			if err != nil {
				return emptyTemplateInfo, err
			}
			cols[colIndex].ColIndex = colIndex
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType) // FlatBuffers Scalar includes bool
			cols[colIndex].IsString = colType == "string"
			cols[colIndex].IsBool = colType == "bool"
		}

		// Populate Rows with a string representation of each table cell.
		var rows []Row = make([]Row, table.RowCount())
		// where(fmt.Sprintf("RowCount = %d", table.RowCount()))
		for rowIndex := 0; rowIndex < table.RowCount(); rowIndex++ {
			var row []string = make([]string, table.ColCount())
			for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
				var cell string
				cell, err = table.GetValAsStringByColIndex(colIndex, rowIndex)
				if err != nil {
					return emptyTemplateInfo, err
				}
				var isStringType bool
				isStringType, err = table.IsColTypeByColIndex(colIndex, "string")
				if err != nil {
					return emptyTemplateInfo, err
				}
				if isStringType {
					cell = fmt.Sprintf("%q", cell) // Add delimiters.
				}
				row[colIndex] = cell
			}
			rows[rowIndex] = row
			// where(fmt.Sprintf("row[%d] = %v", rowIndex, rows[rowIndex]))
		}

		var colNames []string = make([]string, table.ColCount())
		var colTypes []string = make([]string, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColName(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}
			colNames[colIndex] = colName

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil {
				return emptyTemplateInfo, err
			}
			colTypes[colIndex] = colType
		}

		tables[tableIndex].Cols = cols
		tables[tableIndex].TableIndex = tableIndex
		tables[tableIndex].TableName = table.Name()
		tables[tableIndex].RowCount = table.RowCount()
		tables[tableIndex].ColCount = table.ColCount()
		tables[tableIndex].Rows = rows
		tables[tableIndex].ColNames = colNames
		tables[tableIndex].ColTypes = colTypes
	}

	// Get tableset metadata.
	// Make a copy of the tables and use them as metadata-only.
	// We end up with 2 instances of TableSet:-
	// (1) tableSet which contains data.            Is accessible in templates as: .Tables           (an array of Table)
	// (2) metadataTableSet which contains NO data. Is accessible in templates as: .TableSetMetadata (a TableSet)

	const copyRows = false                           // i.e., don't copy rows.
	metadataTableSet, err := tableSet.Copy(copyRows) // Accessible as
	if err != nil {
		return emptyTemplateInfo, err
	}

	for tableIndex := 0; tableIndex < metadataTableSet.TableCount(); tableIndex++ {
		table, err := metadataTableSet.TableByTableIndex(tableIndex)
		if err != nil {
			return emptyTemplateInfo, err
		}

		err = table.SetStructShape(true)
		if err != nil {
			return emptyTemplateInfo, err
		}
	}

	tableSetMetadata := metadataTableSet.String()
	tableSetMetadata = indentText("\t\t", tableSetMetadata)

	tableSetData := tableSet.String()
	//	tableSetData = indentText("\t", tableSetData)

	tablesTemplateInfo = TablesTemplateInfoType{
		GeneratedDateFromFile:    generatedDateFromFile(tableSet),
		GeneratedFromFile:        generatedFromFile(tableSet),
		UsingCommand:             usingCommand(tableSet, packageName),
		UsingCommandMinusG:       usingCommandMinusG(tableSet, packageName),
		GotablesFileNameAbsolute: tableSet.FileName(),
		Year:                     copyrightYear(),
		NameSpace:                tableSet.Name(),
		PackageName:              packageName,
		TableSetMetadata:         tableSetMetadata,
		TableSetData:             tableSetData,
		Tables:                   tables,
	}

	return tablesTemplateInfo, nil
}

func copyrightYear() (copyrightYear string) {
	firstYear := "2017" // See github dates.
	copyrightYear = fmt.Sprintf("%s-%s", firstYear, thisYear())
	return
}

func yearRangeFromFirstYear(firstYear string) (yearRange string) {
	thisYear := thisYear()
	if firstYear == thisYear {
		yearRange = firstYear
	} else {
		yearRange = fmt.Sprintf("%s-%s", firstYear, thisYear)
	}
	return
}

func thisYear() (thisYear string) {
	thisYear = time.Now().Format("2006")
	return
}

func generatedDateFromFile(tableSet *gotables.TableSet) string {
	return fmt.Sprintf("Generated %s from your gotables file %s", time.Now().Format("Monday 2 Jan 2006"), tableSet.FileName())
}

func generatedFromFile(tableSet *gotables.TableSet) string {
	return tableSet.FileName()
}

func usingCommand(tableSet *gotables.TableSet, packageName string) string {
	var usingCommand string

	// Sample:
	// flattablesc -v -f ../flattables_sample/tables.got -n flattables_sample -p package_name

	nameSpace := tableSet.Name()
	fileName := filepath.Base(tableSet.FileName())

	indent := "\t"
	usingCommand = "using the following command:\n"
	usingCommand += indentText(indent, fmt.Sprintf("$ cd %s\t# Where you defined your tables in file %s\n", nameSpace, fileName))
	usingCommand += indentText(indent, fmt.Sprintf("$ flattablesc -v -f ../%s/%s -n %s -p %s\n",
		nameSpace, fileName, nameSpace, packageName))
	usingCommand += indentText(indent, "See instructions at: https://github.com/urban-wombat/flattables")

	return usingCommand
}

func usingCommandMinusG(tableSet *gotables.TableSet, packageName string) string {
	var usingCommand string

	// Sample:
	// flattablesc -v -g -f ../flattables_sample/tables.got -n flattables_sample -p package_name

	nameSpace := tableSet.Name()
	fileName := filepath.Base(tableSet.FileName())

	indent := "\t"
	usingCommand = "using the following command:\n"
	usingCommand += indentText(indent, fmt.Sprintf("$ cd %s\t# Where you defined your tables in file %s\n", nameSpace, fileName))
	usingCommand += indentText(indent, fmt.Sprintf("$ flattablesc -v -g -f ../%s/%s -n %s -p %s\n",
		nameSpace, fileName, nameSpace, packageName))
	usingCommand += indentText(indent, "See instructions at: https://github.com/urban-wombat/flattables")

	return usingCommand
}

type removeStruct struct {
	replace string
	with    string
	id      string
	count   int
}

var rmstr = []removeStruct{
	{"\r\n", "\n", "rn", 0}, // Remove ^M
	//	{ "\r",                 "",         "r",  0 },	// Maybe replace by rn
	{"\n\n\n", "\n\n", "02", 0},
	{"\n\n\n", "\n\n", "03", 0},
	{"\n\t\n", "\n\n", "04", 0},
	{"\n\n}", "\n}", "05", 0},
	{"\n\n)", "\n)", "06", 0},
	{"\n\n\n\n", "\n\n", "07", 0},
	{"\n\n\n", "\n\n", "08", 0},
	{"\n\t\t\n", "\n\n", "09", 0},
	{"\n\t\t\t\n", "\n", "10", 0},
	{"\n\t\t\t\n", "\n", "11", 0},
	{"\t\n", "", "12", 0},
	{"\t\n", "", "13", 0},
	{"\n\n\n\n", "\n\n", "14", 0},
	{"\n\n\n\n", "\n\n", "15", 0},
	{"\n\n\n", "\n\n", "16", 0},
	{"\n\n}", "\n}", "17", 0},
	{"\n\n\t\t}", "\n\t}", "18", 0},
	{"\n\n\t}", "\n\t}", "19", 0},
	{"\n    \n)", "\n)", "20", 0},
	{"\t\t\t\t\t\t}", "\t\t\t}", "21", 0},
	{"{\n\n", "{\n", "22", 0},
	{"{\n\n", "{\n", "22", 0},
	{"\t\n}", "}", "23", 0}, // Why doesn't this do anything?
}

func RemoveExcessTabsAndNewLines(code string) string {
	// Use cat -A flattables_sample_flattables.go to detect non-printing characters.

	for i := 0; i < len(rmstr); i++ {
		var codeIn = code
		code = strings.Replace(code, rmstr[i].replace, rmstr[i].with, -1)
		if code != codeIn {
			rmstr[i].count++
		}
	}

	var verbose bool = false
	if verbose {
		fmt.Println()

		// Used
		for i := 0; i < len(rmstr); i++ {
			if rmstr[i].count > 0 {
				fmt.Printf("  Used %d times id %q\n", rmstr[i].count, rmstr[i].id)
			}
		}

		// Unused
		for i := 0; i < len(rmstr); i++ {
			if rmstr[i].count == 0 {
				fmt.Printf("UNUSED %d times id %q\n", rmstr[i].count, rmstr[i].id)
			}
		}
	}

	return code
}
