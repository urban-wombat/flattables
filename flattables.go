package flattables

import (
	"bytes"
	"bufio"
	"fmt"
	"github.com/urban-wombat/gotables"
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
	"int8":    "byte",	// Signed
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
}


func funcName() string {
    pc, _, _, _ := runtime.Caller(1)
    nameFull := runtime.FuncForPC(pc).Name() // main.foo
    nameEnd := filepath.Ext(nameFull)        // .foo
    name := strings.TrimPrefix(nameEnd, ".") // foo
    return name
}

func MakeSchema1(table *gotables.Table, gotablesFileName string, schemaFileName string) (string, error) {
	if table == nil {
		return "", fmt.Errorf("%s(table): table is <nil>", funcName())
	}

	tableName := table.Name()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("/*\n"))
	buf.WriteString(fmt.Sprintf("\t%s\n", schemaFileName))
	buf.WriteString(fmt.Sprintf("\tDO NOT MODIFY\n"))
	buf.WriteString(fmt.Sprintf("\tFlatBuffers schema automatically generated %s from:\n",
		time.Now().Format("3:04 PM Monday 2 Jan 2006")))
	buf.WriteString(fmt.Sprintf("\t\tgotables file:\n%s", indentText("\t\t\t", gotablesFileName)))
	buf.WriteString(fmt.Sprintf("\t\tgotables.Table:\n%s", indentText("\t\t\t", table.String())))
	buf.WriteString(fmt.Sprintf("*/\n\n"))

	buf.WriteString(fmt.Sprintf("namespace %s;\n", tableName))
	buf.WriteByte('\n')

	buf.WriteString(fmt.Sprintf("table %s {\n", tableName))

	for colIndex := 0; colIndex < table.ColCount(); colIndex++ {

		colName, err := table.ColName(colIndex)
		if err != nil {
			return "", err
		}

		colType, err := table.ColType(colName)
		if err != nil {
			return "", err
		}

		schemaType, err := schemaType(colType)
		if err != nil {
			return "", err
		}

		buf.WriteString(fmt.Sprintf("\t%s:[%s];\t// Go type []%s\n", colName, schemaType, colType))

	}

	buf.WriteString("}\n")

	buf.WriteByte('\n')
	buf.WriteString(fmt.Sprintf("root_type %s;\n", tableName))

	return buf.String(), nil
}

func schemaType(colType string) (string, error) {
	schemaType, exists := GoFlatBuffersTypes[colType]
	if exists {
		return schemaType, nil
	} else {
		return "", fmt.Errorf("No FlatBuffers type available for Go type: %s", colType)
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

func MakeSchema(table *gotables.Table, schemaFileName string) (string, error) {
	var err error
	if table == nil {
		return "", fmt.Errorf("%s(table): table is <nil>", funcName())
	}

	tableName := table.Name()

const templateString =
`
/*
	{{.SchemaFileName}}
	DO NOT MODIFY
	{{.AutomaticallyFrom}}
{{.TableString -}}
*/

namespace {{.NameSpace}};

table {{.TableName}} {
	{{range .TableFields}}
	{{- .}}
	{{end}}
}

root_type {{.RootType}};
`

	type SchemaInfo struct {
		SchemaFileName string
		AutomaticallyFrom string
		TableString string
		NameSpace string
		TableName string
		TableFields []string
		RootType string
	}

	// More-complex assignments
	automatically := fmt.Sprintf("FlatBuffers schema automatically generated %s from gotables.Table:",
		time.Now().Format("3:04 PM Monday 2 Jan 2006"))
	tableFields, err := flatBuffersTableFields(table)
	if err != nil { return "", err }

	// Populate schema struct.
	var schemaInfo = SchemaInfo{
		SchemaFileName: schemaFileName,
		AutomaticallyFrom: automatically,
		TableString: fmt.Sprintf("%s", indentText("\t\t", table.String())),
		NameSpace: tableName,
		TableName: tableName,
		TableFields: tableFields,
		RootType: tableName,
	}


	var buf *bytes.Buffer = bytes.NewBufferString("")

	t := template.New("fbs schema")

	t, err = t.Parse(templateString)
	if err != nil { return "", err }

	err = t.Execute(buf, schemaInfo)
	if err != nil { return "", err }

	return buf.String(), nil
}

func flatBuffersTableFields(table *gotables.Table) ([]string, error) {

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

		field := fmt.Sprintf("%s:[%s];\t// Go type []%s", colName, schemaType, colType)

		fields[colIndex] = field
	}

	return fields, nil
}
