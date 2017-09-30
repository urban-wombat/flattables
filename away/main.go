package main

import (
	"fmt"
	"log"
	"github.com/urban-wombat/gotablesflatbuffers/users"
	flatbuffers "github.com/google/flatbuffers/go"
)

func init() {
	log.SetFlags(log.Lshortfile) // For var where
}

var where = log.Print

func main() {
	initialSize := 0
	b := flatbuffers.NewBuilder(initialSize)
	buf := MakeUser(b, []byte("Arthur Dent"), 42)
	name, id := ReadUser(buf)
// where(fmt.Sprintf("id = %v\n", id))
	fmt.Printf("%s has id %d. The encoded data is %d bytes long.\n", name, id, len(buf))
}

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
where(fmt.Sprintf("user type = %T\n", user))
where(fmt.Sprintf("user.Table = %#v\n", user.Table))

	// point the name variable to the bytes containing the encoded name:
	name = user.Name()

	// copy the user's id (since this is just a uint64):
	id = user.Id()

	return
}
