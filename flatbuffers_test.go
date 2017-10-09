package flatbuffers

import (
	"fmt"
    "log"
    "testing"
	"github.com/urban-wombat/gotables"
	"github.com/urban-wombat/flatbuffers/users"
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

func BenchmarkFlatBuffersMakeUser(b *testing.B) {
	initialSize := 0
	buffer := flatbuffers.NewBuilder(initialSize)
	/*
	name, id := ReadUser(buf)
	fmt.Printf("%s has id %d. The encoded data is %d bytes long.\n", name, id, len(buf))
	*/

    for i := 0; i < b.N; i++ {
		MakeUser(buffer, []byte("Arthur Dent"), 42)
    }
}

func BenchmarkFlatBuffersReadUser(b *testing.B) {
	initialSize := 0
	buffer := flatbuffers.NewBuilder(initialSize)
	buf := MakeUser(buffer, []byte("Arthur Dent"), 42)
	/*
	name, id := ReadUser(buf)
	fmt.Printf("%s has id %d. The encoded data is %d bytes long.\n", name, id, len(buf))
	*/

    for i := 0; i < b.N; i++ {
		ReadUser(buf)
    }
}

func BenchmarkGotablesMakeUser(b *testing.B) {
	s :=
	`[User]
	name string
	id   uint64
	`
	table, err := gotables.NewTableFromString(s)
	if err != nil {
		b.Error(err)
	}

    for i := 0; i < b.N; i++ {
		_, err = TypeStructSliceFromTable_User(table)
		if err != nil {
			b.Error(err)
		}
    }
}

func BenchmarkGotablesReadUser(b *testing.B) {
	s :=
	`[User]
	name string
	id   uint64
	`
	table, err := gotables.NewTableFromString(s)
	if err != nil {
		b.Error(err)
	}

	var user []User
	user, err = TypeStructSliceFromTable_User(table)
	if err != nil {
		b.Error(err)
	}

    for i := 0; i < b.N; i++ {
		_, err = TypeStructSliceToTable_User(user)
		if err != nil {
			b.Error(err)
		}
    }
}

func BenchmarkGotablesReadUser_OLD_WAY(b *testing.B) {
	s :=
	`[User]
	name string
	id   uint64
	`
	table, err := gotables.NewTableFromString(s)
	if err != nil {
		b.Error(err)
	}

	var user []User
	user, err = TypeStructSliceFromTable_User(table)
	if err != nil {
		b.Error(err)
	}

    for i := 0; i < b.N; i++ {
		_, err = TypeStructSliceToTable_User_OLD_WAY(user)
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
// where(fmt.Sprintf("name_position = %v\n", name_position))

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

/*
	Automatically generated source code. DO NOT MODIFY. Generated 5:48 PM Saturday 30 Sep 2017.

	type User struct generated from *gotables.Table [User] for including in your code.
*/
type User struct {
	name string
	id uint64
}
/*
	Automatically generated source code. DO NOT MODIFY. Generated 5:48 PM Saturday 30 Sep 2017.
	Generate a slice of type User struct from *gotables.Table [User] for including in your code.
*/
func TypeStructSliceFromTable_User(table *gotables.Table) ([]User, error) {
	if table == nil {
		return nil, fmt.Errorf("TypeStructSliceFromTable_User(slice []User) slice is <nil>")
	}

	var User []User = make([]User, table.RowCount())

	for rowIndex := 0; rowIndex < table.RowCount(); rowIndex++ {
		name, err := table.GetString("name", rowIndex)
		if err != nil {
			return nil, err
		}
		User[rowIndex].name = name

		id, err := table.GetUint64("id", rowIndex)
		if err != nil {
			return nil, err
		}
		User[rowIndex].id = id
	}

	return User, nil
}
/*
	Automatically generated source code. DO NOT MODIFY. Generated 5:48 PM Saturday 30 Sep 2017.

	Generate a gotables Table [User] from a slice of type struct []User for including in your code.
*/
func TypeStructSliceToTable_User_OLD_WAY(slice []User) (*gotables.Table, error) {
	if slice == nil {
		return nil, fmt.Errorf("TypeStructSliceToTable_User(slice []User) slice is <nil>")
	}

	var err error

	var seedTable string = `
	[User]
	name string
	id uint64
	`
	var table *gotables.Table
	table, err = gotables.NewTableFromString(seedTable)
	if err != nil {
		return nil, err
	}

	for rowIndex := 0; rowIndex < len(slice); rowIndex++ {
		err = table.AppendRow()
		if err != nil {
			return nil, err
		}

		err = table.SetString("name", rowIndex, slice[rowIndex].name)
		if err != nil {
			return nil, err
		}

		err = table.SetUint64("id", rowIndex, slice[rowIndex].id)
		if err != nil {
			return nil, err
		}
	}

	return table, nil
}

/*
    Automatically generated source code. DO NOT MODIFY. Generated 9:12 PM Saturday 30 Sep 2017.

    Generate a gotables Table [User] from a slice of type struct []User for including in your code.
*/
func TypeStructSliceToTable_User(slice []User) (*gotables.Table, error) {
    if slice == nil {
        return nil, fmt.Errorf("TypeStructSliceToTable_User(slice []User) slice is <nil>")
    }

    var err error

    var table *gotables.Table
    var tableName string = "User"
    var colNames []string = []string{"name","id"}
    var colTypes []string = []string{"string","uint64"}
    table, err = gotables.NewTableFromMetadata(tableName, colNames, colTypes)
    if err != nil {
        return nil, err
    }

    for rowIndex := 0; rowIndex < len(slice); rowIndex++ {
        err = table.AppendRow()
        if err != nil {
            return nil, err
        }

        err = table.SetString("name", rowIndex, slice[rowIndex].name)
        if err != nil {
            return nil, err
        }

        err = table.SetUint64("id", rowIndex, slice[rowIndex].id)
        if err != nil {
            return nil, err
        }
    }

    return table, nil
}

var forGob string =
`[Table]
name string = "Arthur Dent"
id   uint64 = 42
`

func BenchmarkGobEncode(b *testing.B) {
    // Set up for benchmark.
    table, err := gotables.NewTableFromString(forGob)
    if err != nil {
        b.Error(err)
    }
// fmt.Printf("\nBenchmarkGobEncode\n%s\n", table)

    for i := 0; i < b.N; i++ {
        _, err := table.GobEncode()
        if err != nil {
            b.Error(err)
        }
    }
}

func BenchmarkGobDecode(b *testing.B) {
    // Set up for benchmark.
    var err error
    var table *gotables.Table
    table, err = gotables.NewTableFromString(forGob)
    if err != nil {
        b.Error(err)
    }

    // Set up for benchmark.
    gobEncodedTable, err := table.GobEncode()
    if err != nil {
        b.Error(err)
    }

// var table2 *gotables.Table
    for i := 0; i < b.N; i++ {
        _, err = gotables.GobDecodeTable(gobEncodedTable)
        if err != nil {
            b.Error(err)
        }
    }
// fmt.Printf("\nBenchmarkGobDecode\n%s\n", table2)
}
