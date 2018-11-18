# Getting started with `Google FlatBuffers` via `FlatTables`

## Install and Test

If you hit a wall or feel that something is missing or unclear, email to: `urban.wombat.burrow@gmail.com`

1. Install FlatBuffers

	```
    go get -u github.com/google/flatbuffers/go
	```

	For more information:
    * [How-To: Install FlatBuffers](https://rwinslow.com/posts/how-to-install-flatbuffers) by [Robert Winslow](https://rwinslow.com)
    * [FlatBuffers Go Documentation](https://google.github.io/flatbuffers/flatbuffers_guide_use_go.html)

2. Install `gotables`, `FlatTables` and `gotablesutils`

	```
    go get -u github.com/urban-wombat/gotables

    go get -u github.com/urban-wombat/flattables

    go get -u github.com/urban-wombat/gotablesutils

    $ cd gotablesutils

    go install flattablesc.go

    go install gotsyntax.go
	```

	Relationships between the packages:
	* `flattables` uses `gotables`
	* `flattablesc` uses `flattables` and `gotables`

3. Create your directory `my_package`

	```
    $ mkdir my_package
	```

	`my_package` (or whatever you decide to call it) will be your namespace and package name.

4. Create your `FlatTables` schema/data file

    It doesn't matter where you create it or what you call it. But for simplicity, let's call it `tables.got`
	and create it in your newly-created directory `my_package`.

	The table names, column names and types are used to generate the `FlatBuffers` schema file `*.fbs`

	The data in the tables is used in the auto-generated bench tests. So add some dummy data for testing.

	You can come up with your own tables of data, or can copy and paste the tables below into `tables.got`

	The `gotables` syntax is self-evident and most `Go` types are supported.
	
	**Not** supported are
	* `int` and `uint` (their size is machine-dependent, and `FlatBuffers` has only fixed-size)
	* `complex` (not supported by gotables)
	* `rune` (doesn't seem to be supported by `FlatBuffers`, perhaps because its size varies)

	If you just want to get started and not deal with creating your own `gotables` schema right now, just copy
	and paste the tables below into `tables.got` and proceed to the next step. You can go back later
	and whip up data that looks like your own.

```
    [MyXyzTable]
        x       y       z
    int16 float32 float64
        4       5       6
       44      55      66
      444     555     666
     4444    5555    6666
       16      32      64
    
    [StringsAndThings]
    flintstones nums spain          female unsigned
    string      int8 string         bool     uint32
    "Fred"         0 "The rain"     false         0
    "Wilma"        1 "in Spain"     true         11
    "Barney"       2 "stays mainly" false        22
    "Betty"        3 "in"           true         33
    "Bam Bam"      4 "the"          false        44
    "Pebbles"      5 "plain."       true         55
    
    [Wombats]
    housingPolicy string = "burrow"
    topSpeedKmH int8 = 40
    species string = "Vombatus"
    class string = "Mammalia"
    wild bool = true
```


5. Check its validity with `gotsyntax`

    ```
    $ gotsyntax tables.got
    ```

	The `FlatTables` utility `flattablesc` will also do a syntax check, but you might as well get earlier feedback with `gotsyntax`.

	`flattablesc` also invokes the Google `FlatTables` `flatc` code generator. It doesn't seem to police the 
	[FlatBuffers style guide](https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html)
	but `flattablesc` does. `flattablesc` also guards against some gotchas specific to generating `Go` code.

`FlatTables` is also a little more strict than `gotables` syntax:
* Table names must start with an uppercase character -- [FlatBuffers style guide](https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html)
* Column names must start with a lowercase character -- [FlatBuffers style guide](https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html)
* Table names or column names that so much as **look** like `Go` key words are not permitted. Table and column names end up as
function and variable names in generated `Go` code, and the `Go` compiler doesn't like key words as names. So we don't risk it.
* Transfers between `Go` slices and `FlatBuffers` require the field names to be exported (hence uppercase) which is
done by code generation. So there's a (managed) difference between the
[FlatBuffers style guide](https://google.github.io/flatbuffers/flatbuffers_guide_writing_schema.html)
and the need to export `Go` fields. Languages such as Java convert field names to lowerCamelCase, which is what `FlatTables`
requires here, consistent with `Go` unexported fields. But they are exported as UpperCamelCase where needed in raw `Go` structs
used by `FlatTables`, namely:
```
type RootTableSlice struct {...}
```

See a sample RootTableSlice definition in [flattables_sample_NewSliceFromFlatBuffers.go](https://github.com/urban-wombat/flattables_sample/blob/master/flattables_sample_NewSliceFromFlatBuffers.go)

`type RootTableSlice` is generated for you based on your `tables.got` schema file and the `*fbs` schema file.

6. From within dir `my_package` run the `FlatTables` utility `flattablesc`

    ```
    $ flattablesc -f ../my_package/tables.got -n my_package -p github.com/your-github-name/my_package
    ```

    `flattablesc` creates a flatbuffers schema `*.fbs` file and a number of `Go` source files in `../my_package`.

7. Run the tests

    ```
    $ go test -bench=.
    ```


## `FlatTables` is a simplified tabular subset of `FlatBuffers`

Have a look at the Google `FlatBuffers` official documentation to see
why you should seriously consider `FlatBuffers` (and `FlatTables`)
for **very fast** binary data transfer:
* [Google FlatBuffers official documentation](https://google.github.io/flatbuffers)

If your data is tabular (or can be easily normalised to tabular) then `FlatTables`
may be right for your project.

The `FlatTables` utility `flattablesc` will generate all the code needed to convert
from [gotables](https://github.com/urban-wombat/gotables#gotables) tabular format to
FlatBuffers and back again.

`flattablesc` also generates a `Go` main program with constructor and getter methods
specific to your `FlatBuffers` schema.

The generated code includes:
* conversion functions (which include all the code generated by the `FlatBuffers` utility `flatc`)
* test code
* test Example code (an Example for each of the key functions)
* benchmark tests which will run with the data you put into your `tables.got` schema file.
* a main program with sample code

* There is a [sample implementation](https://godoc.org/github.com/urban-wombat/flattables_sample)
using a `gotables` file
[tables.got](https://github.com/urban-wombat/flattables_sample/blob/master/tables.got) as input to the `flattablesc` utility.
The same way you would create your own implementation.
It is called `flattables_sample`. It is an actual implementation, and not just a toy sample.

When you run `flattablesc` on your own `gotables` schema file, it will generate
your raw `Go` struct tables, functions, examples and benchtests.

Have a look at [urban-wombat/flattables_sample](https://github.com/urban-wombat/flattables_sample)
which is a sample of `FlatBuffers` code generated entirely by `flatc` (FlatBuffers utility)
and `flattablesc` (gotables `FlatTables` utility).
The `flattablesc` utility is at [gotablesutils](https://github.com/urban-wombat/gotablesutils).

Here is the GoDoc of all 90 or so `Go` functions generated by the Google `flatc` utility and `gotables` `flattablesc` utility:
* [https://godoc.org/github.com/urban-wombat/flattables_sample](https://godoc.org/github.com/urban-wombat/flattables_sample)
* [https://godoc.org/github.com/urban-wombat/flattables_sample_main](https://godoc.org/github.com/urban-wombat/flattables_sample_main)

And test and benchmark code:
* [https://github.com/urban-wombat/flattables_sample/blob/master/flattables_sample_test.go](https://github.com/urban-wombat/flattables_sample/blob/master/flattables_sample_test.go)

The main function in [urban-wombat/flattables_sample_main](https://github.com/urban-wombat/flattables_sample_main)
is the simplest possible conversion code, if you don't want to get into
the weeds of moving data into and out of `FlatBuffers`.

ALL of the code, including the `FlatBuffers` schema and all `Go` code,
was generated automatically from `flatc` and `flattablesc`.

When you download and run `flattablesc` (referencing a simple
[gotables](https://github.com/urban-wombat/gotables) file you write yourself)
you can run the tests and benchtest and see the speed of `FlatBuffers`.

## Advantages

### Auto-Generation of schema and `Go` code:

* `FlatTables` auto-generates the `FlatBuffers` schema from your data.
  All you need to do is write a very simple self-describing `gotables` data file (sample below).
  This means normalising your objects to one or more tables (tabular tables, like database tables).

  `FlatBuffers` and `FlatTables` use 'table' in a slightly different sense, but if you see them as tabular
  tables, it makes sense.

  `gotables` is the supporting library and file format used by `FlatTables`.

* `FlatBuffers` utility `flatc` generates `Go` code to write (and read) data conforming to the `FlatBuffers` schema.

* `FlatTables` generates `Go` code to write `gotables`-formatted data to a `FlatBuffers` []byte array.

* `FlatTables` generates `Go` code to test that data has been written to `FlatBuffers` correctly.

* The read step is **very fast**. There is no additional code between you and the auto-generated `FlatBuffers` code.
  (Note: the read step at this stage is read-only. This may get better with the implementation of mutable tables)

* You write only your own code to call these auto-generated methods, and denormalise the data from tables to
  your prefered data structures.

* `FlatTables` uses a subset of Google `FlatBuffers` as a binary format for `gotables` Table objects.

* `FlatTables` is general purpose because it consists of tables, and your own data is probably capable of being
  normalised (in Ted Codd, C J Date fashion) to one or more relational tables ready for transmission and re-assembly
  at the receiving end.

* The Example functions in the `*.test.go` file will get you started coding data transfers.

* You don't have to write the `*.fbs` flatbuffers schema `flattables_sample.fbs`. It is done for you.

* You don't have to write the glue code to get data from `tables.got` to a flatbuffers []byte array.
