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

var where = log.Print

func funcName() string {
    pc, _, _, _ := runtime.Caller(1)
    nameFull := runtime.FuncForPC(pc).Name() // main.foo
    nameEnd := filepath.Ext(nameFull)        // .foo
    name := strings.TrimPrefix(nameEnd, ".") // foo
    return name
}

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

func IsFlatBuffersScalar(colType string) bool {
	_, exists := goFlatBuffersScalarTypes[colType]
	return exists
}

func isScalar(table *gotables.Table, colName string) bool {
	colType, err := table.ColType(colName)
	if err != nil { log.Fatal(err) }

	isNumeric, err := gotables.IsNumericColType(colType)
	if err != nil { log.Fatal(err) }

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

func FlatBuffersSchemaFromTableSet(tableSet *gotables.TableSet, schemaFileName string) (string, error) {
	if tableSet == nil {
		return "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName())
	}

	var err error
	var buf *bytes.Buffer = bytes.NewBufferString("")
	var tplate *template.Template = template.New("FlatTables Schema")

	type ColInfo struct {
		ColName string
		ColType string
		FbsType string
	}

	type TableInfo struct {
		Table *gotables.Table
		TableIndex int
		TableName string
		Cols []ColInfo
	}

	type SchemaInfo struct {
		SchemaFileName string
		TableSetFileName string
		GeneratedFrom string
		TableString string
		TableSetName string	// These three have the same value.
		NameSpace string	// These three have the same value.
		RootType string		// These three have the same value.
		Tables []TableInfo
	}

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", err }

		if table.ColCount() >= 0 {
			fmt.Fprintf(os.Stderr, "*** FlatTables: Adding table [%s] to FlatBuffers schema\n", table.Name())
		} else {
			// Skip tables with zero cols.
			fmt.Fprintf(os.Stderr, "--- FlatTables: Skip   table [%s] with zero cols\n", table.Name())
			continue
		}

		if startsWithLowerCase(table.Name()) {
			// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
			return "", fmt.Errorf("FlatBuffers style guide requires UpperCamelCase table names. Rename [%s] to [%s]",
				table.Name(), firstCharToUpper(table.Name()))
		}

		if isGoKeyWord(table.Name()) {
			return "", fmt.Errorf("Cannot use a Go key word as a table name, even if it's upper case. Rename [%s]", table.Name())
		}

		if isFlatTablesKeyWord(table.Name()) {
			return "", fmt.Errorf("Cannot use a FlatBuffers key word as a table name, even if it's merely similar. Rename [%s]", table.Name())
		}
	
		tables[tableIndex].Table = table

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColNameByColIndex(colIndex)
			if err != nil { return "", err }

			if startsWithUpperCase(colName) {
				// See: https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html
				return "", fmt.Errorf("FlatBuffers style guide requires lowerCamelCase field names. In table [%s] rename %s to %s",
					table.Name(), colName, firstCharToLower(colName))
			}

			if isGoKeyWord(colName) {
				return "", fmt.Errorf("Cannot use a Go key word as a col name, even if it's upper case. Rename [%s]", colName)
			}

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return "", err }

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].FbsType, err = schemaType(colType)
			if err != nil { return "", err }
		}

		tables[tableIndex].Cols = cols
		tables[tableIndex].TableIndex = tableIndex
	}

	// More-complex assignments
	var generatedFrom string
	if tableSet.FileName() != "" {
		generatedFrom = fmt.Sprintf("FlatBuffers schema generated %s from file: %s",
			time.Now().Format("3:04 PM Monday 2 Jan 2006" ), tableSet.FileName())
	} else {
		generatedFrom = fmt.Sprintf("FlatBuffers schema generated %s from a gotables.TableSet",
			time.Now().Format("3:04 PM Monday 2 Jan 2006" ))
	}

	// Populate schema struct.
	var schemaInfo = SchemaInfo {
		SchemaFileName: filepath.Base(schemaFileName),
		GeneratedFrom: generatedFrom,
		TableSetName: tableSet.Name(),
		NameSpace: tableSet.Name(),
		RootType: tableSet.Name(),
		Tables: tables,
	}
// fmt.Println(schemaInfo)

	// Add a user-defined function to schema tplate.
	tplate = tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})

	const templateFile = "../flattables/schema.template"

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	data, err := ioutil. ReadFile(templateFile)
	if err != nil { log.Fatal(err) }

	tplate, err = tplate.Parse(string(data))
	if err != nil { log.Fatal(err) }

	err = tplate.Execute(buf, schemaInfo)
	if err != nil { log.Fatal(err) }

	return buf.String(), nil
}

func startsWithLowerCase(s string) bool {
    rune0 := rune(s[0])
	return unicode.IsLower(rune0)
}

func startsWithUpperCase(s string) bool {
    rune0 := rune(s[0])
	return unicode.IsUpper(rune0)
}

func firstCharToUpper(s string) string {
    rune0 := rune(s[0])
	return string(unicode.ToUpper(rune0)) + s[1:]
}

func firstCharToLower(s string) string {
    rune0 := rune(s[0])
	return string(unicode.ToLower(rune0)) + s[1:]
}

func tableName(table *gotables.Table) string {
	return "// " + table.Name()
}

func rowCount(table *gotables.Table) int {
	return table.RowCount()
}

func FlatBuffersGoCodeFromTableSet(tableSet *gotables.TableSet, fileNames []string) (tofbString, fromfbString string, err error) {
	if tableSet == nil {
		return "", "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName())
	}

	type ColInfo struct {
		ColName string
		ColType string
		IsScalar bool
		IsString bool
		IsBool bool
	}

	type TableInfo struct {
		Table *gotables.Table
		Cols []ColInfo
	}

	type GoCodeInfo struct {
		PackageName string
		ToFbImports []string
		FromFbImports []string
//		FlatTablesCodeFileName string
		ToFbCodeFileName string
		FromFbCodeFileName string
		GeneratedFrom string
		Year string
		Tables []TableInfo
//		TableNames []string
		TableSetMetadata string
	}

	var generatedFrom string
	if tableSet.FileName() != "" {
		generatedFrom = fmt.Sprintf("FlatBuffers Go code generated %s from file: %s",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"), tableSet.FileName())
	} else {
		generatedFrom = fmt.Sprintf("FlatBuffers Go code generated %s from a gotables.TableSet",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"))
	}

	year := fmt.Sprintf("%s", time.Now().Format("2006"))

	// Remove data (which we don't use anyway) from tables so we are left with metadata.
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", "", err }

		err = table.DeleteRowsAll()
		if err != nil { return "", "", err }

		err = table.SetStructShape(true)
		if err != nil { return "", "", err }
	}
	tableSetMetadata := tableSet.String()
	tableSetMetadata = indentText("\t\t", tableSetMetadata)
	// fmt.Println(tableSetMetadata)

	tofbImports := []string {
		`flatbuffers "github.com/google/flatbuffers/go"`,
		`"github.com/urban-wombat/gotables"`,
//		`"github.com/urban-wombat/flattables"`,
		`"fmt"`,
		`"log"`,
		`"path/filepath"`,
		`"runtime"`,
		`"strings"`,
	}

	fromfbImports := []string {
//		`flatbuffers "github.com/google/flatbuffers/go"`,
		`"github.com/urban-wombat/gotables"`,
//		`"github.com/urban-wombat/flattables"`,
		`"fmt"`,
		`"log"`,
//		`"path/filepath"`,
//		`"runtime"`,
//		`"strings"`,
	}

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
//	var tableNames []string = make([]string, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", "", err }
	
		tables[tableIndex].Table = table
//		tableNames[tableIndex] = table.Name()

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColName(colIndex)
			if err != nil { return "", "", err }

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return "", "", err }

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType)
			cols[colIndex].IsString = colType == "string"
			cols[colIndex].IsBool = colType == "bool"
		}

		tables[tableIndex].Cols = cols
	}

	var goCodeInfo = GoCodeInfo {
		PackageName: tableSet.Name(),
		ToFbImports: tofbImports,
		FromFbImports: fromfbImports,
//		FlatTablesCodeFileName: filepath.Base(flatTablesCodeFileName),
		ToFbCodeFileName: filepath.Base(fileNames[0]),
		FromFbCodeFileName: filepath.Base(fileNames[1]),
		GeneratedFrom: generatedFrom,
		Year: year,
		Tables: tables,
//		TableNames: tableNames,
		TableSetMetadata: tableSetMetadata,
	}

	// Add a user-defined function to Go code tplate.
//	tplate = tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
//	tplate = tplate.Funcs(template.FuncMap{"rowCount": rowCount})


	// (1) Generate NewFlatTablesFlatBuffersFromTableSet()

//	const templateFile = "../flattables/GetTableSetAsFlatBuffers.template"
//	const templateFile = "../flattables/FlatBuffersFromTableSet.template"
//	const templateFile = "../flattables/FlatTablesFromTableSet.template"
//	const templateFile = "../flattables/NewFlatTablesFlatBuffersFromTableSet.template"
	const toFlatBuffersTemplateFile = "../flattables/NewFlatTablesFlatBuffersFromTableSet.template"
	var toFlatBuf *bytes.Buffer = bytes.NewBufferString("")
	var tofbTplate *template.Template = template.New("TO FlatBuffers Go Code")
	tofbTplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tofbTplate.Funcs(template.FuncMap{"rowCount": rowCount})

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	tofbData, err := ioutil. ReadFile(toFlatBuffersTemplateFile)
	if err != nil { log.Fatal(err) }

	tofbTplate, err = tofbTplate.Parse(string(tofbData))
	if err != nil { log.Fatal(err) }

// where(goCodeInfo)
	err = tofbTplate.Execute(toFlatBuf, goCodeInfo)
	if err != nil { log.Fatal(err) }

	tofbString = toFlatBuf.String()


	// (2) Generate NewTableSetFromFlatBuffers()
	var fromFlatBuf *bytes.Buffer = bytes.NewBufferString("")
	const fromFlatBuffersTemplateFile = "../flattables/NewTableSetFromFlatBuffers.template"
	var fromTplate *template.Template = template.New("FROM FlatBuffers Go Code")
	fromTplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	fromTplate.Funcs(template.FuncMap{"firstCharToLower": firstCharToLower})
	fromTplate.Funcs(template.FuncMap{"rowCount": rowCount})

	// Open and read file explicitly to avoid calling fromTplate.ParseFile() which has problems.
	fromData, err := ioutil. ReadFile(fromFlatBuffersTemplateFile)
	if err != nil { log.Fatal(err) }

	fromTplate, err = fromTplate.Parse(string(fromData))
	if err != nil { log.Fatal(err) }

// where(goCodeInfo)
	err = fromTplate.Execute(fromFlatBuf, goCodeInfo)
	if err != nil { log.Fatal(err) }

	fromfbString = fromFlatBuf.String()

	return
}

func FlatBuffersTestGoCodeFromTableSet(tableSet *gotables.TableSet, flatTablesTestCodeFileName string) (string, error) {
	if tableSet == nil {
		return "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName())
	}

	var err error
	var buf *bytes.Buffer = bytes.NewBufferString("")
	var tplate *template.Template = template.New("FlatTables Test Go")

	type ColInfo struct {
		ColName string
		ColType string
		IsScalar bool
		IsString bool
		IsBool bool
	}

	type TableInfo struct {
		Table *gotables.Table
		Cols []ColInfo
	}

	type GoTestCodeInfo struct {
		PackageName string
		GotablesFileName string
		FlatTablesTestCodeFileName string
		GeneratedFrom string
		Year string
		Imports []string
		Tables []TableInfo
		TableNames []string
	}

	var generatedFrom string
	if tableSet.FileName() != "" {
		generatedFrom = fmt.Sprintf("FlatBuffers Test Go code generated %s from file: %s",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"), tableSet.FileName())
	} else {
		generatedFrom = fmt.Sprintf("FlatBuffers Test Go code generated %s from a gotables.TableSet",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"))
	}

	year := fmt.Sprintf("%s", time.Now().Format("2006"))

	imports := []string {
//		`flatbuffers "github.com/google/flatbuffers/go"`,
		`"github.com/urban-wombat/gotables"`,
//		`"github.com/urban-wombat/flattables"`,
//		`"fmt"`,
//		`"log"`,
//		`"path/filepath"`,
//		`"runtime"`,
//		`"strings"`,
		`"testing"`,
	}

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
//	var tableNames []string = make([]string, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", err }
	
		tables[tableIndex].Table = table
//		tableNames[tableIndex] = table.Name()

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColName(colIndex)
			if err != nil { return "", err }

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return "", err }

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType)
			cols[colIndex].IsString = colType == "string"
			cols[colIndex].IsBool = colType == "bool"
		}

		tables[tableIndex].Cols = cols
	}

	var goTestCodeInfo = GoTestCodeInfo {
		PackageName: tableSet.Name(),
		GotablesFileName: tableSet.FileName(),
		FlatTablesTestCodeFileName: filepath.Base(flatTablesTestCodeFileName),
		GeneratedFrom: generatedFrom,
		Year: year,
		Imports: imports,
		Tables: tables,
//		TableNames: tableNames,
	}

	// Add a user-defined function to Test Go code tplate.
	tplate = tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tplate = tplate.Funcs(template.FuncMap{"firstCharToLower": firstCharToLower})
	tplate = tplate.Funcs(template.FuncMap{"rowCount": rowCount})

	const templateFile = "../flattables/FlatTablesTest.template"

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	data, err := ioutil. ReadFile(templateFile)
	if err != nil { log.Fatal(err) }

	tplate, err = tplate.Parse(string(data))
	if err != nil { log.Fatal(err) }

// where(goTestCodeInfo)
	err = tplate.Execute(buf, goTestCodeInfo)
	if err != nil { log.Fatal(err) }

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
	name = strings.ToLower(name)
	_, exists := goKeyWords[name]
	return exists
}

// Could be tricky if a user inadvertently uses FlatTables as a table name.
var flatTablesKeyWords = map[string]string {
	"flattables":	"flattables",
}

func isFlatTablesKeyWord(name string) bool {
	name = strings.ToLower(name)
	_, exists := goKeyWords[name]
	return exists
}
