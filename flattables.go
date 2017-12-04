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

var goScalarTypes = map[string]string {
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
	_, exists := goScalarTypes[colType]
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
		AutomaticallyFrom string
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
	var automaticallyFrom string
	if tableSet.FileName() != "" {
		automaticallyFrom = fmt.Sprintf("FlatBuffers schema automatically generated %s from file: %s",
			time.Now().Format("3:04 PM Monday 2 Jan 2006" ), tableSet.FileName())
	} else {
		automaticallyFrom = fmt.Sprintf("FlatBuffers schema automatically generated %s from a gotables.TableSet",
			time.Now().Format("3:04 PM Monday 2 Jan 2006" ))
	}

	// Populate schema struct.
	var schemaInfo = SchemaInfo {
		SchemaFileName: filepath.Base(schemaFileName),
		AutomaticallyFrom: automaticallyFrom,
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

func FlatBuffersGoCodeFromTableSet(tableSet *gotables.TableSet, flatTablesCodeFileName string) (string, error) {
	if tableSet == nil {
		return "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName())
	}

	var err error
	var buf *bytes.Buffer = bytes.NewBufferString("")
	var tplate *template.Template = template.New("FlatTables Go")

	type ColInfo struct {
		ColName string
		ColType string
		IsScalar bool
	}

	type TableInfo struct {
		Table *gotables.Table
		Cols []ColInfo
	}

	type GoCodeInfo struct {
		PackageName string
		FlatTablesCodeFileName string
		AutomaticallyFrom string
		Year string
		Imports []string
		Tables []TableInfo
		TableNames []string
	}

	var automaticallyFrom string
	if tableSet.FileName() != "" {
		automaticallyFrom = fmt.Sprintf("FlatBuffers Go code automatically generated %s from file: %s",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"), tableSet.FileName())
	} else {
		automaticallyFrom = fmt.Sprintf("FlatBuffers Go code automatically generated %s from a gotables.TableSet",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"))
	}

	year := fmt.Sprintf("%s", time.Now().Format("2006"))

	imports := []string {
		`flatbuffers "github.com/google/flatbuffers/go"`,
		`"github.com/urban-wombat/gotables"`,
//		`"github.com/urban-wombat/flattables"`,
		`"fmt"`,
		`"log"`,
		`"path/filepath"`,
		`"runtime"`,
		`"strings"`,
	}

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
	var tableNames []string = make([]string, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", err }
	
		tables[tableIndex].Table = table
		tableNames[tableIndex] = table.Name()

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColName(colIndex)
			if err != nil { return "", err }

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return "", err }

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType)
		}

		tables[tableIndex].Cols = cols
	}

	var goCodeInfo = GoCodeInfo {
		PackageName: tableSet.Name(),
		FlatTablesCodeFileName: filepath.Base(flatTablesCodeFileName),
		AutomaticallyFrom: automaticallyFrom,
		Year: year,
		Imports: imports,
		Tables: tables,
		TableNames: tableNames,
	}

	// Add a user-defined function to Go code tplate.
	tplate = tplate.Funcs(template.FuncMap{"firstCharToUpper": firstCharToUpper})
	tplate = tplate.Funcs(template.FuncMap{"rowCount": rowCount})

//	const templateFile = "../flattables/GetTableSetAsFlatBuffers.template"
//	const templateFile = "../flattables/FlatBuffersFromTableSet.template"
	const templateFile = "../flattables/FlatTablesFromTableSet.template"

	// Open and read file explicitly to avoid calling tplate.ParseFile() which has problems.
	data, err := ioutil. ReadFile(templateFile)
	if err != nil { log.Fatal(err) }

	tplate, err = tplate.Parse(string(data))
	if err != nil { log.Fatal(err) }

// where(goCodeInfo)
	err = tplate.Execute(buf, goCodeInfo)
	if err != nil { log.Fatal(err) }

	return buf.String(), nil
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
	}

	type TableInfo struct {
		Table *gotables.Table
		Cols []ColInfo
	}

	type GoTestCodeInfo struct {
		PackageName string
		FlatTablesTestCodeFileName string
		AutomaticallyFrom string
		Year string
		Imports []string
		Tables []TableInfo
		TableNames []string
	}

	var automaticallyFrom string
	if tableSet.FileName() != "" {
		automaticallyFrom = fmt.Sprintf("FlatBuffers Test Go code automatically generated %s from file: %s",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"), tableSet.FileName())
	} else {
		automaticallyFrom = fmt.Sprintf("FlatBuffers Test Go code automatically generated %s from a gotables.TableSet",
			time.Now().Format("3:04 PM Monday 2 Jan 2006"))
	}

	year := fmt.Sprintf("%s", time.Now().Format("2006"))

	imports := []string {
		`flatbuffers "github.com/google/flatbuffers/go"`,
		`"github.com/urban-wombat/gotables"`,
//		`"github.com/urban-wombat/flattables"`,
		`"fmt"`,
		`"log"`,
//		`"path/filepath"`,
//		`"runtime"`,
//		`"strings"`,
		`"testing"`,
	}

	var tables []TableInfo = make([]TableInfo, tableSet.TableCount())
	var tableNames []string = make([]string, tableSet.TableCount())
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", err }
	
		tables[tableIndex].Table = table
		tableNames[tableIndex] = table.Name()

		var cols []ColInfo = make([]ColInfo, table.ColCount())
		for colIndex := 0; colIndex < table.ColCount(); colIndex++ {
			colName, err := table.ColName(colIndex)
			if err != nil { return "", err }

			colType, err := table.ColTypeByColIndex(colIndex)
			if err != nil { return "", err }

			cols[colIndex].ColName = colName
			cols[colIndex].ColType = colType
			cols[colIndex].IsScalar = IsFlatBuffersScalar(colType)
		}

		tables[tableIndex].Cols = cols
	}

	var goTestCodeInfo = GoTestCodeInfo {
		PackageName: tableSet.Name(),
		FlatTablesTestCodeFileName: filepath.Base(flatTablesTestCodeFileName),
		AutomaticallyFrom: automaticallyFrom,
		Year: year,
		Imports: imports,
		Tables: tables,
		TableNames: tableNames,
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
