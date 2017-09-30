package vanilla

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "log"
    "math"
    "math/rand"
    "sort"
    "strconv"
    "strings"
    "testing"
//	"github.com/urban-wombat/gotablesflatbuffers/users"
	flatbuffers "github.com/google/flatbuffers/go"
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

func BenchmarkMakeUser(b *testing.B) {
    var err error

	initialSize := 0
	buffer := flatbuffers.NewBuilder(initialSize)
	/*
	name, id := ReadUser(buf)
	fmt.Printf("%s has id %d. The encoded data is %d bytes long.\n", name, id, len(buf))
	*/

    for i := 0; i < b.N; i++ {
        _, err = NewTableSetFromString(tableSetString)
		buf := MakeUser(buffer, []byte("Arthur Dent"), 42)
        if err != nil {
            b.Error(err)
        }
    }
}

func init() {
	log.SetFlags(log.Lshortfile) // For var where
}

var where = log.Print

func MakeUser(b *flatbuffers.Builder, name []byte, id uint64) []byte {
	// re-use the already-allocated Builder:
	b.Reset()

	// create the name object and get its offset:
	name_position := b.CreateByteString(name)
where(fmt.Sprintf("name_position = %v\n", name_position))

	// write the User object:
	users.UserStart(b)
	users.UserAddName(b, name_position)
	users.UserAddId(b, id)
	user_position := users.UserEnd(b)

	// finish the write operations by our User the root object:
	b.Finish(user_position)

	// return the byte slice containing encoded data:
	return b.Bytes[b.Head():]
}

func ReadUser(buf []byte) (name []byte, id uint64) {
	// initialize a User reader from the given buffer:
	user := users.GetRootAsUser(buf, 0)
// where(fmt.Sprintf("user type = %T\n", user))
// where(fmt.Sprintf("user.Table = %#v\n", user.Table))

	// point the name variable to the bytes containing the encoded name:
	name = user.Name()

	// copy the user's id (since this is just a uint64):
	id = user.Id()

	return
}
