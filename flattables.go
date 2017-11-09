package flattables

import (
	"bytes"
	"bufio"
	"fmt"
	"github.com/urban-wombat/gotables"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
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

var GoFlatBuffersTypes = map[string]string {
	"bool":    "bool",
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
	"string":  "string",
//	"int":     "long",	// Assume largest int size:  64 bit.
//	"uint":    "ulong",	// Assume largest uint size: 64 bit.
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
	schemaType, exists := GoFlatBuffersTypes[colType]
	if exists {
		return schemaType, nil
	} else {
		var changeTypeTo string
		switch colType {
			case "int": changeTypeTo = "int64"
			case "uint": changeTypeTo = "uint64"
			default: return "", fmt.Errorf("No FlatBuffers-compatible Go type suggestion for Go type: %s", colType)
		}
		return "", fmt.Errorf("No FlatBuffers type available for Go type: %s (suggest change it to Go type: %s)",
			colType, changeTypeTo)
	}
}

func indentText(indent string, text string) string {
	var indentedText string = ""
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		indentedText += fmt.Sprintf("%s%s\n", indent, scanner.Text())
	}
	return indentedText
}

//func MakeSchemaTable(table *gotables.Table, schemaFileName string) (string, error) {
//	var err error
//	if table == nil {
//		return "", fmt.Errorf("%s(table): table is <nil>", funcName())
//	}
//
//	tableName := table.Name()
//
//const templateString =
//`
///*
//	DO NOT MODIFY
//	{{.AutomaticallyFrom}}
//{{.TableString -}}
//*/
//
//namespace {{.NameSpace}};
//
//table {{.TableName}} {
//	{{range .TableFields}}
//	{{- .}}
//	{{end}}
//}
//
//root_type {{.RootType}};
//`
//
//	type SchemaInfo struct {
//		SchemaFileName string
//		AutomaticallyFrom string
//		TableString string
//		NameSpace string
//		TableName string
//		TableFields []string
//		RootType string
//	}
//
//	// More-complex assignments
//	automatically := fmt.Sprintf("FlatBuffers schema automatically generated %s from gotables.Table:",
//		time.Now().Format("3:04 PM Monday 2 Jan 2006"))
//	fieldNames, err := fieldNames(table)
//	if err != nil { return "", err }
//
//	// Populate schema struct.
//	var schemaInfo = SchemaInfo{
//		SchemaFileName: schemaFileName,
//		AutomaticallyFrom: automatically,
//		TableString: fmt.Sprintf("%s", indentText("\t\t", table.String())),
//		NameSpace: tableName,
//		TableName: tableName,
//		TableFields: fieldNames,
//		RootType: tableName,
//	}
//
//
//	var buf *bytes.Buffer = bytes.NewBufferString("")
//
//	t := template.New("fbs schema")
//
//	t, err = t.Parse(templateString)
//	if err != nil { return "", err }
//
//	err = t.Execute(buf, schemaInfo)
//	if err != nil { return "", err }
//
//	return buf.String(), nil
//}

func MakeSchemaTableSet(tableSet *gotables.TableSet, schemaFileName string) (string, error) {
	var err error
	if tableSet == nil {
		return "", fmt.Errorf("%s(tableSet): tableSet is <nil>", funcName())
	}

	type SchemaInfo struct {
		SchemaFileName string
		AutomaticallyFrom string
		TableString string
		NameSpace string
		RootType string
	}

	// More-complex assignments
	automatically := fmt.Sprintf("FlatBuffers schema automatically generated %s from a gotables.TableSet",
		time.Now().Format("3:04 PM Monday 2 Jan 2006"))

	// Populate schema struct.
	var schemaInfo = SchemaInfo {
		SchemaFileName: schemaFileName,
		AutomaticallyFrom: automatically,
		NameSpace: tableSet.Name(),
		RootType: tableSet.Name(),
	}

const beginTemplate =
`
/*
	{{.SchemaFileName}}
	DO NOT MODIFY
	{{.AutomaticallyFrom}}
{{.TableString -}}
*/

namespace {{.NameSpace}};
`

	type TableSetInfo struct {
		TableSetName string
		TableNames []string
	}

	tableNames, err := tableNames(tableSet)
	if err != nil { return "", err }

	var tableSetInfo = TableSetInfo {
		TableSetName: tableSet.Name(),
		TableNames: tableNames,
	}

	type TableInfo struct {
		TableName string
		TableFields []string
	}
	var tableInfo TableInfo

const tableSetTemplate =
`
// root table
table {{.TableSetName}} {
	{{range .TableNames}}
	{{- .}}
	{{end}}
}
`

const tableTemplate =
`
// data table
table {{.TableName}} {
	{{range .TableFields}}
	{{- .}}
	{{end}}
}
`

const endTemplate =
`
root_type {{.RootType}};
`

	var buf *bytes.Buffer = bytes.NewBufferString("")

	tplate := template.New("fbs schema")

	// Generate beginning of schema.
	tplate, err = tplate.Parse(beginTemplate)
	if err != nil { return "", err }
	err = tplate.Execute(buf, schemaInfo)
	if err != nil { return "", err }

	// Generate root table of gotables.Table instances.
	fmt.Fprintf(os.Stderr, "Adding table [%s] to FlatBuffers schema (as the schema root table)\n", tableSet.Name())
	tplate, err = tplate.Parse(tableSetTemplate)
	if err != nil { return "", err }
	err = tplate.Execute(buf, tableSetInfo)
	if err != nil { return "", err }

	// Generate gotables.Table instances.
	tplate, err = tplate.Parse(tableTemplate)
	if err != nil { return "", err }
	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return "", err }

		if table.ColCount() > 0 {
			fmt.Fprintf(os.Stderr, "Adding table [%s] to FlatBuffers schema\n", table.Name())
		} else {
			fmt.Fprintf(os.Stderr, "Skip   table [%s] with zero cols\n", table.Name())
			continue
		}

		fieldNames, err := fieldNames(table)
		if err != nil { return "", err }

		tableInfo = TableInfo {
			TableName: table.Name(),
			TableFields: fieldNames,
		}
		err = tplate.Execute(buf, tableInfo)
		if err != nil { return "", err }
	}

	// Generate end of schema.
	tplate, err = tplate.Parse(endTemplate)
	if err != nil { return "", err }
	err = tplate.Execute(buf, schemaInfo)
	if err != nil { return "", err }

	return buf.String(), nil
}

func tableNames(tableSet *gotables.TableSet) ([]string, error) {
	var tableNames []string

	for tableIndex := 0; tableIndex < tableSet.TableCount(); tableIndex++ {
		table, err := tableSet.TableByTableIndex(tableIndex)
		if err != nil { return nil, err }

		if table.ColCount() == 0 {
			// Skip tables with zero cols.
			continue
		}

		field := fmt.Sprintf("%s: %s;", table.Name(), table.Name())

		tableNames = append(tableNames, field)
	}

	return tableNames, nil
}

func fieldNames(table *gotables.Table) ([]string, error) {

	var fields []string = make([]string, table.ColCount())

	for colIndex := 0; colIndex < table.ColCount(); colIndex++ {

		colName, err := table.ColName(colIndex)
		if err != nil {
			return nil, err
		}

		colType, err := table.ColType(colName)
		if err != nil {
			return nil, err
		}

		schemaType, err := schemaType(colType)
		if err != nil {
			return nil, err
		}

		field := fmt.Sprintf("%s: [%s]; // Go type []%s", colName, schemaType, colType)

		fields[colIndex] = field
	}

	return fields, nil
}
