# FlatTables - A Simple Subset of FlatBuffers

A simple and fast way to get started with Google FlatBuffers.

Have a look at the [Google FlatBuffers official documentation](https://google.github.io/flatbuffers) to see
why you should seriously consider FlatBuffers (and by implication FlatTables) for VERY fast binary
data transfer.

## Advantages

### Auto-Generation of schema and Go code:

* FlatTables auto-generates the FlatBuffers schema from your data.
  All you need to do is write a very simple self-describing gotables data file (sample below).
  This means normalising your objects to one or more tables (tabular tables, like database tables).

  FlatBuffers and FlatTables use 'table' in a slightly different sense, but if you see them as tabular
  tables, it makes sense.

* FlatBuffers utility `flatc` generates Go code to write (and read) data conforming to the FlatBuffers schema.

* FlatTables generates Go code to write gotables-formatted data to a FlatBuffers []byte array.

* FlatTables generates Go code to test that data has been written to FlatBuffers correctly.

* The read step is VERY fast. There is no additional code between you and the auto-generated FlatBuffers code.
  (Note: the read step at this stage is read-only. This may get better.)

* You write only your own code to call these auto-generated methods, and denormalise the data from tabular to
  your prefered data structures.

* FlatTables uses a subset of Google FlatBuffers as a binary format for gotables Table objects.

* FlatTables is general purpose because it consists of tables, and your own data is probably capable of being
  normalised (in Ted Codd, C J Date fashion) to one or more tables ready for transmission and re-assembly
  at the receiving end.

* You don't HAVE to use gotables data format to write to a FlatBuffers []byte array. Once you have followed the simple
steps (described below) to generate the schema and code, you can take guidance from the generated code
to write directly from your data objects to FlatBuffers. Perhaps use gotables during initial
development, and write directly to FlatBuffers later for the highest possible speeds.

## How the flattables_sample repository was auto-generated - you can do it too

1. Install FlatBuffers

    * See [How-To: Install FlatBuffers](https://rwinslow.com/posts/how-to-install-flatbuffers) by Robert Winslow.

    * And the [FlatBuffers Go Documentation](https://google.github.io/flatbuffers/flatbuffers_guide_use_go.html)

    * `go get github.com/google/flatbuffers/go`

2. Install gotables and gotablesutils

    `go get github.com/urban-wombat/flattables`

	`go get github.com/urban-wombat/gotablesutils`

3. Create directory `flattables_sample`

    `$ mkdir flattables_sample`

4. In dir flattables_sample create a file containing one or more gotables tables. The tables don't need to contain data,
but let's include some data and use the same file for writing to a FlatBuffers []byte array and running our tests.
We'll call it "tables.got" (.got is for gotables).

```
    [MyAbcTable]
        a    b     c       d e           f    u8
    int64 byte int16 float32 bool  float64 uint8
        1    2     3    3.0  true    111.1     1
       11   22    33    3.3  false   222.2     2
      111  222   333    3.33 true    333.3     3
        2    4     8    8.0  false   444.4     4
 
    [MyXyzTable]
        x     y     z
    int64 int32 int64
        4     5     6
       44    55    66
      444   555   666
     4444  5555  6666
       16    32    64
    
    [MyStrTable]
    s1                i s2
    string         int8 string
    "Fred"            0 "The rain ..."
    "Wilma"           1 "in Spain ..."
    "Barney"          2 "falls mainly ..."
    "Betty"           3 "on the ..."
    "Bam Bam"         4 "plain."
    "Pebbles"         5 "Why?"
    "Grand Poobah"    6 "Why not!"
    "Dino"            7 "Dinosaur"
    
    [Tabular]
        a     b    c
    int16 int32 int8
        1     2    3
    
    [Structural]
    x uint8 = 1
    y uint16 = 2
    z uint64 = 3
```

Check its validity with gotsyntax:

    $ gotsyntax tables.got
    tables.got (syntax okay)

The FlatTables utility `gotft` will also do a validity check, but you might as well get earlier feedback with `gotsyntax`.

Note: FlatTables is a little more strict than gotables syntax:
* Table names must start with an uppercase character.
* Column names must start with a lowercase character.
* Table names or column names that so much as look like Go key words are not permitted. Table and column names end up as
variable names in generated Go code, and the compiler can get annoyed seeing key words used as variables.

3. Run the FlatTables utility `gotft` (gotables flattables).

```
    $ gotft -f tables.got -n flattables_sample
    (1) Reading: tables.got (gotables file)
    (1) Setting gotables.TableSet name to "flattables_sample" (from -n flattables_sample)
    (2) Dir <outdir> already exists: ../flattables_sample (good)
    (2) FlatTables: Generating FlatBuffers schema file: ../flattables_sample/flattables_sample.fbs (from tables.got)
    *** FlatTables: Adding table [MyAbcTable] to FlatBuffers schema
    *** FlatTables: Adding table [MyXyzTable] to FlatBuffers schema
    *** FlatTables: Adding table [MyStrTable] to FlatBuffers schema
    *** FlatTables: Adding table [Tabular] to FlatBuffers schema
    *** FlatTables: Adding table [Structural] to FlatBuffers schema
    (3) flatc:      Generating FlatBuffers Go code with cmd: flatc --go -o .. ../flattables_sample/flattables_sample.fbs
    (4) FlatTables: Generating FlatTables Go code: ../flattables_sample/flattables_sample_flattables.go
    (4) FlatTables: Generating FlatTables test Go code: ../flattables_sample/flattables_sample_test.go
    (*) DONE
```

The following files are generated. Some by Google FlatBuffers flatc (which is called by gotft), and some by FlatTables,
mainly code to link gotables to flattables (a constrained flatbuffers).

```
    flattables_sample.fbs (by flattables)
    MyAbcTable.go (by flatc)
    MyXyzTable.go (by flatc)
    MyStrTable.go (by flatc)
    Tabular.go (by flatc)
    Structural.go (by flatc)
    FlatTables.go (by flatc)
    flattables_sample_flattables.go (by flattables)
    flattables_sample_test.go (by flattables)
```

You did not have to write the .fbs flatbuffers schema `flattables_sample.fbs`. It was done for you.

You did not have to write the glue code to get data from `tables.got` to a flatbuffers []byte array.

And if you wish to populate the flatbuffers []byte array yourself, and not go via gotables, just
follow the setter calls in `flattables_sample_flattables.go` to get you going. In that case, you could use
the gotables `tables.got` file purely for generating the schema and setter methods. That would run faster.

4. Run the tests

```
    go test
    go test -bench=.
```

That's it!
