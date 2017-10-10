package flatbuffers

import (
	"bytes"
	"bufio"
	"fmt"
	"github.com/urban-wombat/gotables"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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

func MakeSchema(table *gotables.Table, gotableFileName string, schemaFileName string) (string, error) {
	if table == nil {
		return "", fmt.Errorf("%s(table): table is <nil>", funcName())
	}

	tableName := table.Name()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("// %s\n", schemaFileName))
	buf.WriteString(fmt.Sprintf("// DO NOT MODIFY. FlatBuffers schema automatically generated %s from:\n",
		time.Now().Format("3:04 PM Monday 2 Jan 2006")))
	buf.WriteString(fmt.Sprintf("//   file:\n%s", indentText("//\t", gotableFileName)))
	buf.WriteString(fmt.Sprintf("//   table:\n%s\n", indentText("//\t", table.String())))

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

		buf.WriteString(fmt.Sprintf("\t%s : [%s] ; // Go type []%s\n", colName, schemaType, colType))

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
