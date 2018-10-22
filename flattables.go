package flattables

import (
	"bytes"
	"bufio"
	"fmt"
	"github.com/urban-wombat/gotables"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
	"unicode"
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
var where = log.Print

// FlatBuffers schema types: bool | byte | ubyte | short | ushort | int | uint | float | long | ulong | double | string
// From: https://google.github.io/flatbuffers/flatbuffers_grammar.html

// Built-in scalar types are:
// 8 bit: byte, ubyte, bool
// 16 bit: short, ushort
// 32 bit: int, uint, float
// 64 bit: long, ulong, double
// From: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html

var goToFlatBuffersTypes = map[string]string {
	"bool":    "bool",
	"int8":    "byte",	// Signed.
	"int16":   "short",
	"int32":   "int",	// (Go rune is an alias for Go int32. For future reference.)
	"int64":   "long",
	"byte":    "ubyte",	// Unsigned. Go byte is an alias for Go uint8.
	"[]byte":  "[ubyte]",	// Unsigned. Go byte is an alias for Go uint8. NOTE: This [ubyte] IS NOT IMPLEMENTED IN FLATTABLES!
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

var goFlatBuffersScalarTypes = map[string]string {
	"bool":    "bool",	// Scalar from FlatBuffers point of view.
	"int8":    "byte",	// Signed.
	"int16":   "short",
	"int32":   "int",	// (Go rune is an alias for Go int32. For future reference.)
	"int64":   "long",
	"byte":    "ubyte",	// Unsigned. Go byte is an alias for Go uint8.
	"uint8":   "ubyte",
	"uint16":  "ushort",
	"uint32":  "uint",
	"uint64":  "ulong",
	"float32": "float",
	"float64": "double",
}

func funcName() string {
    pc, _, _, _ := runtime.Caller(1)
    nameFull := runtime.FuncForPC(pc).Name() // main.foo
    nameEnd := filepath.Ext(nameFull)        // .foo
    name := strings.TrimPrefix(nameEnd, ".") // foo
    return name
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
			case "int": suggestChangeTypeTo = "int32 or int64"
			case "uint": suggestChangeTypeTo = "uint32 or uint64"
			default: return "", fmt.Errorf("No FlatBuffers-compatible Go type suggestion for Go type: %s", colType)
		}
		return "", fmt.Errorf("No FlatBuffers type available for Go type: %s (suggest change it to Go type: %s)",
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
		fmt.Fprintf(os.Stderr, "%s [%s].%s ERROR: %v\n", funcName(), table.Name(), colName, err)
		return false
	}

	isNumeric, err := gotables.IsNumericColType(colType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s [%s].%s ERROR: %v\n", funcName(), table.Name(), colName, err)
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

func FlatBuffersSchemaFromTableSet(templateInfo TemplateInfo) (string, error) {

	var err error

	var buf *bytes.Buffer = bytes.NewBufferString("")

	const schemaFromTableSetTemplateFile = "../flattables/schema.template"
	// Use the file name as the template name so that file name appears in error output.
//	var tplate *template.Template = template.New("FlatTables Schema")
	var tplate *template.Template = template.New(schemaFromTableSetTemplateFile)

	// Add a user-defined function to schema tplate.
	tplate = tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	data, err := ioutil. ReadFile(schemaFromTableSetTemplateFile)
	if err != nil { return "", err }

	tplate, err = tplate.Parse(string(data))
	if err != nil { return "", err }

	err = tplate.Execute(buf, templateInfo)
	if err != nil { return "", err }

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

func FlatBuffersGoCodeFromTableSet(tableSet *gotables.TableSet, templateInfo TemplateInfo, fileNames []string) (tofbStr, fromfbStr, mainStr string, err error) {
	if tableSet == nil {
		return "", "", "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName())
	}

	// imports
	templateInfo.ToFbImports = []string {
		`flatbuffers "github.com/google/flatbuffers/go"`,
		`"github.com/urban-wombat/gotables"`,
		`"fmt"`,
//		`"log"`,
		`"path/filepath"`,
		`"runtime"`,
		`"strings"`,
	}

	// imports
	templateInfo.FromFbImports = []string {
		`"github.com/urban-wombat/gotables"`,
		`"fmt"`,
		`"log"`,
	}

	// imports
	templateInfo.MainImports = []string {
		`"fmt"`,
		`"log"`,
		`"github.com/urban-wombat/gotables"`,
	}

	templateInfo.ToFbCodeFileName = filepath.Base(fileNames[0])
	templateInfo.FromFbCodeFileName = filepath.Base(fileNames[1])
	templateInfo.MainCodeFileName = filepath.Base(fileNames[2])


	// (1) Generate NewFlatBuffersFromTableSet()

	var toFlatBuf *bytes.Buffer = bytes.NewBufferString("")

	const toFlatBuffersTemplateFile = "../flattables/NewFlatBuffersFromTableSet.template"
	// Use the file name as the template name so that file name appears in error output.
//	var tofbTplate *template.Template = template.New("TO FlatBuffers Go Code")
	var tofbTplate *template.Template = template.New(toFlatBuffersTemplateFile)

	tofbTplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tofbTplate.Funcs(template.FuncMap{"rowCount": rowCount})

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	tofbData, err := ioutil. ReadFile(toFlatBuffersTemplateFile)
	if err != nil { return }

	tofbTplate, err = tofbTplate.Parse(string(tofbData))
	if err != nil { return }

	err = tofbTplate.Execute(toFlatBuf, templateInfo)
	if err != nil { return }

	tofbStr = toFlatBuf.String()


	// (2) Generate NewTableSetFromFlatBuffers()
	var fromFlatBuf *bytes.Buffer = bytes.NewBufferString("")

	const fromFlatBuffersTemplateFile = "../flattables/NewTableSetFromFlatBuffers.template"
//	var fromTplate *template.Template = template.New("FROM FlatBuffers Go Code")
	var fromTplate *template.Template = template.New(fromFlatBuffersTemplateFile)

	fromTplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	fromTplate.Funcs(template.FuncMap{"firstCharToLower": firstCharToLower})
	fromTplate.Funcs(template.FuncMap{"rowCount": rowCount})

	// Open and read file explicitly to avoid calling fromTplate.ParseFile() which has problems.
	fromData, err := ioutil. ReadFile(fromFlatBuffersTemplateFile)
	if err != nil { return }

	fromTplate, err = fromTplate.Parse(string(fromData))
	if err != nil { return }

	err = fromTplate.Execute(fromFlatBuf, templateInfo)
	if err != nil { return }

	fromfbStr = fromFlatBuf.String()


	// (3) Generate main()
	var mainBuf *bytes.Buffer = bytes.NewBufferString("")

	const mainBuffersTemplateFile = "../flattables/main.template"
	// Use the file name as the template name so that file name appears in error output.
//	var mainTplate *template.Template = template.New("MAIN FlatBuffers Go Code")
	var mainTplate *template.Template = template.New(mainBuffersTemplateFile)

	mainTplate.Funcs(template.FuncMap{"firstCharToLower": firstCharToLower})
	mainTplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})

	// Open and read file explicitly to avoid calling mainTplate.ParseFile() which has problems.
	mainData, err := ioutil. ReadFile(mainBuffersTemplateFile)
	if err != nil { return }

	mainTplate, err = mainTplate.Parse(string(mainData))
	if err != nil { return }

	err = mainTplate.Execute(mainBuf, templateInfo)
	if err != nil { return }

	mainStr = mainBuf.String()

	return
}

func FlatBuffersTestGoCodeFromTableSet(tableSet *gotables.TableSet, templateInfo TemplateInfo) (string, error) {
	if tableSet == nil { return "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName()) }

	var err error

	var buf *bytes.Buffer = bytes.NewBufferString("")

	const testTemplateFile = "../flattables/FlatTablesTest.template"
	// Use the file name as the template name so that file name appears in error output.
	var tplate *template.Template = template.New(testTemplateFile)

	templateInfo.TestImports = []string {
		`"github.com/urban-wombat/gotables"`,
		`"testing"`,
		`"fmt"`,
	}

	// Add a user-defined function to Test Go code tplate.
	tplate = tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tplate = tplate.Funcs(template.FuncMap{"firstCharToLower": firstCharToLower})
	tplate = tplate.Funcs(template.FuncMap{"rowCount": rowCount})
//	tplate = tplate.Funcs(template.FuncMap{"getValAsStringByColIndex": getValAsStringByColIndex})

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	data, err := ioutil. ReadFile(testTemplateFile)
	if err != nil { return "", err }

	tplate, err = tplate.Parse(string(data))
	if err != nil { return "", err }

	err = tplate.Execute(buf, templateInfo)
	if err != nil { return "", err }

	return buf.String(), nil
}

// Compilation will fail if a user inadvertently uses a Go key word as a name.
var goKeyWords = map[string]string {
	"break":		"break",
	"default":		"default",
	"func":			"func",
	"interface":	"interface",
	"select":		"select",
	"case":			"case",
	"defer":		"defer",
	"go":			"go",
	"map":			"map",
	"struct":		"struct",
	"chan":			"chan",
	"else":			"else",
	"goto":			"goto",
	"package":		"package",
	"switch":		"switch",
	"const":		"const",
	"fallthrough":	"fallthrough",
	"if":			"if",
	"range":		"range",
	"type":			"type",
	"continue":		"continue",
	"for":			"for",
	"import":		"import",
	"return":		"return",
	"var":			"var",
}

func isGoKeyWord(name string) bool {
	nameLower := strings.ToLower(name)
	_, exists := goKeyWords[nameLower]
	return exists
}

// Could be tricky if a user inadvertently uses a word used in FlatBuffers schemas.
var flatBuffersOrFlatTablesKeyWords = map[string]string {
	"flattables":	"flattables",	// FlatTables is used as the root table name and root_type.
	"table":		"table",
	"namespace":	"namespace",
	"root_type":	"root_type",
	"ubyte":		"ubyte",
	"float":		"float",
	"long":			"long",
	"ulong":		"ulong",
	"short":		"short",
	"ushort":		"ushort",
	"double":		"double",
	"enum":			"enum",
	"union":		"union",
	"include":		"include",
}

func isFlatBuffersOrFlatTablesKeyWord(name string) bool {
	name = strings.ToLower(name)
	_, exists := flatBuffersOrFlatTablesKeyWords[name]
	return exists
}

type ColInfo struct {
	ColName string
	ColType string
	FbsType string
	ColIndex int
	IsScalar bool	// FlatBuffers Scalar includes bool
	IsString bool
	IsBool bool
	IsDeprecated bool
}

type Row []string

type TableInfo struct {
	Table *gotables.Table
	TableIndex int
	TableName string
	RowCount int
	ColCount int
	Cols []ColInfo
	Rows []Row
	ColNames []string
	ColTypes []string
}

type TemplateInfo struct {
	GeneratedFrom string
	UsingCommand string
	NameSpace string	// Included in PackageName.
	PackageName string	// Includes NameSpace
	Year string
	SchemaFileName string
	ToFbImports []string
	ToFbCodeFileName string
	FromFbImports []string
	FromFbCodeFileName string
	MainImports []string
	MainCodeFileName string
	TestCodeFileName string
	TestImports []string
	GotablesFileName string	// We want to replace this with the following TWO file names.
	TablesSchemaFileName string	// For generating flatbuffers schema and functions.
	TablesDataFileName string	// For main() to populate flatbuffers.
	TableSetMetadata string
	Tables []TableInfo
}

func (templateInfo TemplateInfo) Name(tableIndex int) string {
	return templateInfo.Tables[0].Table.Name()
}

func InitTemplateInfo(tableSet *gotables.TableSet, packageName string) (TemplateInfo, error) {

	var emptyTemplateInfo TemplateInfo

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return emptyTemplateInfo, err }

		if table.ColCount() >= 0 {
			fmt.Fprintf(os.Stderr, "  %d  Adding gotables table  to FlatBuffers schema: [%s] \n", tableIndex, table.Name())
		} else {
			// Skip tables with zero cols.
			fmt.Fprintf(os.Stderr, "--- FlatTables: Skip   table [%s] with zero cols\n", table.Name())
			continue
		}

		if startsWithLowerCase(table.Name()) {
			// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
			return emptyTemplateInfo, fmt.Errorf("FlatBuffers style guide requires UpperCamelCase table names. Rename [%s] to [%s]",
				table.Name(), firstCharToUpper(table.Name()))
		}

		if isGoKeyWord(table.Name()) {
			return emptyTemplateInfo,
				fmt.Errorf("Cannot use a Go key word as a table name, even if it's upper case. Rename [%s]", table.Name())
		}

		if isFlatBuffersOrFlatTablesKeyWord(table.Name()) {
			return emptyTemplateInfo,
				fmt.Errorf("Cannot use a FlatBuffers or FlatTables key word as a table name, even if it's merely similar. Rename [%s]", table.Name())
		}

		tables[tableIndex].Table = table	// An array of Table accessible as .Tables

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColNameByColIndex(colIndex)
			if err != nil { return emptyTemplateInfo, err }

			if startsWithUpperCase(colName) {
				// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
				return emptyTemplateInfo, fmt.Errorf("FlatBuffers style guide requires lowerCamelCase field names. In table [%s] rename %s to %s",
					table.Name(), colName, firstCharToLower(colName))
			}

			if isGoKeyWord(colName) {
				return emptyTemplateInfo, fmt.Errorf("Cannot use a Go key word as a col name, even if it's upper case. Rename [%s]", colName)
			}

			if isFlatBuffersOrFlatTablesKeyWord(colName) {
				return emptyTemplateInfo,
					fmt.Errorf("Cannot use a FlatBuffers or FlatTables key word as a col name, even if it's merely similar. Rename [%s]", colName)
			}

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return emptyTemplateInfo, err }

			cols[colIndex].IsDeprecated = isDeprecated(colName)
			if cols[colIndex].IsDeprecated {
				// Restore the col name by removing _DEPRECATED_ indicator.
				colName = strings.Replace(colName, deprecated, "", 1)
				fmt.Fprintf(os.Stderr, "*** FlatTables: Tagged table [%s] column %q is deprecated\n", table.Name(), colName)
			}

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].FbsType, err = schemaType(colType)
			if err != nil { return emptyTemplateInfo, err }
			cols[colIndex].ColIndex = colIndex
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType)	// FlatBuffers Scalar includes bool
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
				if err != nil { return emptyTemplateInfo, err }
				var isStringType bool
				isStringType, err = table.IsColTypeByColIndex(colIndex, "string")
				if err != nil { return emptyTemplateInfo, err }
				if isStringType {
					cell = fmt.Sprintf("%q", cell)	// Add delimiters.
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
			if err != nil { return emptyTemplateInfo, err }
			colNames[colIndex] = colName

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return emptyTemplateInfo, err }
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

	const copyRows = false	// i.e., don't copy rows.
	metadataTableSet, err := tableSet.Copy(copyRows)	// Accessible as 
	if err != nil { return emptyTemplateInfo, err }

	for tableIndex := 0; tableIndex < metadataTableSet.TableCount(); tableIndex++ {
		table, err := metadataTableSet.TableByTableIndex(tableIndex)
		if err != nil { return emptyTemplateInfo, err }

		err = table.SetStructShape(true)
		if err != nil { return emptyTemplateInfo, err }
	}

	tableSetMetadata := metadataTableSet.String()
	tableSetMetadata = indentText("\t\t", tableSetMetadata)

	var templateInfo = TemplateInfo {
		GeneratedFrom: generatedFrom(tableSet),
		UsingCommand: usingCommand(tableSet, packageName),
		GotablesFileName: tableSet.FileName(),
		Year: copyrightYear(),
		NameSpace: tableSet.Name(),
		PackageName: packageName,
		TableSetMetadata: tableSetMetadata,
		Tables: tables,
	}

	return templateInfo, nil
}

func copyrightYear() (copyrightYear string) {
	firstYear := "2017"	// See github dates.
	copyrightYear = fmt.Sprintf("%s-%s", firstYear, time.Now().Format("2006"))
	return
}

func generatedFrom(tableSet *gotables.TableSet) string {
	var generatedFrom string

	if tableSet.FileName() != "" {
		generatedFrom = fmt.Sprintf("Generated %s from your gotables file %s",
			time.Now().Format("Monday 2 Jan 2006"), tableSet.FileName())
	} else {
		generatedFrom = fmt.Sprintf("Generated %s from your gotables.TableSet",
			time.Now().Format("Monday 2 Jan 2006"))
	}

	return generatedFrom
}

func usingCommand(tableSet *gotables.TableSet, packageName string) string {
	var usingCommand string

	// Sample:
	// flattablesc -f ../flattables_sample/tables.got -n flattables_sample

	nameSpace := tableSet.Name()
	fileName := filepath.Base(tableSet.FileName())

	indent := "\t"
	usingCommand = "using the following command:\n"
	usingCommand += indentText(indent, fmt.Sprintf("$ cd %s\t# Where you defined your tables in file %s\n", nameSpace, fileName))
	usingCommand += indentText(indent, fmt.Sprintf("$ flattablesc -f ../%s/%s -n %s -p %s\n",
		nameSpace, fileName, nameSpace, packageName))
	usingCommand += indentText(indent, "See instructions at: https://github.com/urban-wombat/flattables")

	return usingCommand
}
