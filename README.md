# A simple and fast way to get started with `Google FlatBuffers`

1. Install FlatBuffers

    * See [How-To: Install FlatBuffers](https://rwinslow.com/posts/how-to-install-flatbuffers) by Robert Winslow.

    * And the [FlatBuffers Go Documentation](https://google.github.io/flatbuffers/flatbuffers_guide_use_go.html)

    * `go get -u github.com/google/flatbuffers/go`

2. Install gotables, flattables and gotablesutils

	`go get -u github.com/urban-wombat/gotables`

    `go get -u github.com/urban-wombat/flattables`

	`go get -u github.com/urban-wombat/gotablesutils`

    `go install flattablesc.go`

FlatTables uses [gotables](https://github.com/urban-wombat/gotables) as its underlying data format and library.

3. Create directory `flattables_sample`

    `$ mkdir flattables_sample`

4. In dir `flattables_sample` create a file containing one or more gotables tables. The tables don't need to contain data,
but let's include some data and use the same file for writing to a FlatBuffers []byte array and running our tests.
We'll call it "tables.got" (.got is for gotables).

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

The FlatTables utility `flattablesc` will also do a validity check, but you might as well get earlier feedback with `gotsyntax`.

Check its validity with gotsyntax:

    $ gotsyntax tables.got
    tables.got (syntax okay 3 tables)


Note: FlatTables is a little more strict than gotables syntax:
* Table names must start with an uppercase character.
* Column names must start with a lowercase character.
* Table names or column names that so much as look like Go key words are not permitted. Table and column names end up as
variable names in generated Go code, and the compiler gets very annoyed seeing key words used as variables.

3. From within dir `flattables_sample` run the FlatTables utility `flattablesc`

```
    $ flattablesc -f ../flattables_sample/tables.got -n flattables_sample -p github.com/urban-wombat/flattables_sample
     (1) Reading: ../flattables_sample/tables.got (gotables file)
     (2) Setting gotables.TableSet name to "flattables_sample" (from -n flattables_sample)
     (3) Setting package name to "github.com/urban-wombat/flattables_sample" (from -p github.com/urban-wombat/flattables_sample)
     (4) Dir <outdir>     already exists: ../flattables_sample (okay)
     (5) Dir <outdirmain> already exists: ../flattables_sample_main (okay)
     (6) Adding gotables tables to FlatBuffers tables schema:
      #0 FlatTables: Adding gotables table [MyXyzTable] to FlatBuffers schema
      #1 FlatTables: Adding gotables table [StringsAndThings] to FlatBuffers schema
      #2 FlatTables: Adding gotables table [Wombats] to FlatBuffers schema
     (7) FlatTables: Generating schema file: ../flattables_sample/flattables_sample.fbs (from ../flattables_sample/tables.got)
     (8) flatc:      Generating FlatBuffers Go code from schema. $ flatc --go -o .. ../flattables_sample/flattables_sample.fbs
     (9) FlatTables: Generating FlatTables user Go code: ../flattables_sample/flattables_sample_NewFlatBuffersFromTableSet.go
    (10) FlatTables: Generating FlatTables user Go code: ../flattables_sample/flattables_sample_NewTableSetFromFlatBuffers.go
    (11) FlatTables: Generating FlatTables user Go code: ../flattables_sample_main/flattables_sample_main.go
    (12) FlatTables: Generating FlatTables test Go code: ../flattables_sample/flattables_sample_test.go
    DONE
```

The following files are generated. Some by Google FlatBuffers `flatc` (which is called by `flattablesc`), and some by FlatTables,
mainly code to link gotables to flattables (a subset of flatbuffers).

```
    flattables_sample.fbs (by flattables)
    MyXyzTable.go (by FlatBuffers flatc)
    StringsAndThings.go (by FlatBuffers flatc)
    Wombats.go (by FlatBuffers flatc)
    FlatTables.go (by FlatBuffers flatc)
    flattables_sample_NewFlatBuffersFromTableSet.go (by flattables)
    flattables_sample_NewTableSetFromFlatBuffers.go (by flattables)
    flattables_sample_test.go (by flattables)
    flattables_sample/flattables_sample_main.go (by flattables)
```

`FlatTables` is a simple tabular subset of `FlatBuffers`.

Installation and code generation instructions are provided further below.

Have a look at the Google FlatBuffers official documentation to see
why you should seriously consider FlatBuffers (and FlatTables)
for VERY FAST binary data transfer:
* [Google FlatBuffers official documentation](https://google.github.io/flatbuffers)

If your data is tabular (or can be easily normalised to tabular) then FlatTables
may be right for your project.

The `FlatTables` utility `flattablesc` will generate all the code needed to convert
from [gotables](https://github.com/urban-wombat/gotables) tabular format to
FlatBuffers and back again.

`flattablesc` also generates a Go main program with constructor and getter methods
specific to your FlatBuffers schema.  Just to remind you: you didn't have to
generate the FlatBuffers schema (.fbs), it was done for you from your gotables file.

The generated code includes conversion functions (which include all the code
generated by the FlatBuffers utility `flatc`), test code, and a main program.

There is an Example test function for each of the key functions.

There is a suite of benchmark tests which will run with the data you put into
your gotables schema file.

There is a sample implementation (using a gotables file as input to the
`flattablesc` utility) the same way you would create your own implementation.
It is called `flattables_sample`. It is an actual implementation, not just a
toy sample.

When you run flattablesc on your own gotables schema file, it will generate
your raw Go struct tables, functions, examples and benchtests.

Have a look at [urban-wombat/flattables_sample](https://github.com/urban-wombat/flattables_sample)
which is a sample of FlatBuffers code generated entirely by `flatc` (FlatBuffers utility)
and `flattablesc` (gotables FlatTables utility).
The `flattablesc` utility is at [gotablesutils](https://github.com/urban-wombat/gotablesutils).

Here is the GoDoc of all 90 or so Go functions generated by the Google `flatc` utility and gotables `flattablesc` utility:
* [https://godoc.org/github.com/urban-wombat/flattables_sample](https://godoc.org/github.com/urban-wombat/flattables_sample)
* [https://godoc.org/github.com/urban-wombat/flattables_sample_main](https://godoc.org/github.com/urban-wombat/flattables_sample_main)

And test and benchmark code:
* [https://github.com/urban-wombat/flattables_sample/blob/master/flattables_sample_test.go](https://github.com/urban-wombat/flattables_sample/blob/master/flattables_sample_test.go)

The main function in [urban-wombat/flattables_sample_main](https://github.com/urban-wombat/flattables_sample_main)
is the simplest possible conversion code, if you don't want to get into
the weeds of moving data into and out of `FlatBuffers`.

ALL of the code, including the FlatBuffers schema and all Go code,
was generated automatically from `flatc` and `flattablesc`.

When you download and run flattablesc (referencing a simple
[gotables](https://github.com/urban-wombat/gotables) file you write yourself)
you can run the tests and benchtest and see the SPEED of FlatBuffers.

## Advantages

### Auto-Generation of schema and Go code:

* FlatTables auto-generates the FlatBuffers schema from your data.
  All you need to do is write a very simple self-describing gotables data file (sample below).
  This means normalising your objects to one or more tables (tabular tables, like database tables).

  FlatBuffers and FlatTables use 'table' in a slightly different sense, but if you see them as tabular
  tables, it makes sense.

  gotables is the supporting library and file format used by FlatTables.

* FlatBuffers utility `flatc` generates Go code to write (and read) data conforming to the FlatBuffers schema.

* FlatTables generates Go code to write gotables-formatted data to a FlatBuffers []byte array.

* FlatTables generates Go code to test that data has been written to FlatBuffers correctly.

* The read step is VERY FAST. There is no additional code between you and the auto-generated FlatBuffers code.
  (Note: the read step at this stage is read-only. This may get better.)

* You write only your own code to call these auto-generated methods, and denormalise the data from tables to
  your prefered data structures.

* FlatTables uses a subset of Google FlatBuffers as a binary format for gotables Table objects.

* FlatTables is general purpose because it consists of tables, and your own data is probably capable of being
  normalised (in Ted Codd, C J Date fashion) to one or more relational tables ready for transmission and re-assembly
  at the receiving end.

* You don't HAVE to use gotables data format to write to a FlatBuffers []byte array. Once you have followed the simple
steps (described below) to generate the schema and code, you can take guidance from the generated code
to write directly from your data objects to FlatBuffers. Perhaps use gotables during initial
development, and write directly to FlatBuffers later for the highest possible speeds.

You did not have to write the .fbs flatbuffers schema `flattables_sample.fbs`. It was done for you.

You did not have to write the glue code to get data from `tables.got` to a flatbuffers []byte array.

And if you wish to populate the flatbuffers []byte array yourself, and not go via gotables, just
follow the setter calls in the various Go source files to get you going. In that case, you could use
the gotables `tables.got` file purely for generating the schema and setter methods. That would run faster.

4. Run the tests

```
    $ go test -v
    === RUN   TestNewFlatBuffersFromTableSet
    --- PASS: TestNewFlatBuffersFromTableSet (0.00s)
    === RUN   TestNewTableSetFromFlatBuffers
    --- PASS: TestNewTableSetFromFlatBuffers (0.00s)
    PASS
    ok      github.com/urban-wombat/flattables_sample       0.123s
```

```
    $ go test -bench=.
    goos: windows
    goarch: amd64
    pkg: github.com/urban-wombat/flattables_sample
    BenchmarkGetFlatBuffersAndCompareWithGotables-4           300000              5367 ns/op
    BenchmarkGetFlatBuffersOnly-4                           10000000               121 ns/op
    PASS
    ok      github.com/urban-wombat/flattables_sample       3.194s
```

That's it!
