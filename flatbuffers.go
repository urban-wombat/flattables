package flatbuffers

import (
	"fmt"
	"github.com/urban-wombat/gotables"
	"runtime"
	"path/filepath"
	"strings"
)

func funcName() string {
    pc, _, _, _ := runtime.Caller(1)
    nameFull := runtime.FuncForPC(pc).Name() // main.foo
    nameEnd := filepath.Ext(nameFull)        // .foo
    name := strings.TrimPrefix(nameEnd, ".") // foo
    return name
}

func MakeSchema(table *gotables.Table) (string, error) {
	if table == nil {
		return "", fmt.Errorf("%s(table): table is <nil>", funcName())
	}

	var schema string

	return schema, nil
}
