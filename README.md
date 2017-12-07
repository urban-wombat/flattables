# FlatTables

A simple and fast way to get started with Google FlatBuffers.

## Advantages

* The FlatBuffers schema is auto-generated from a self-describing gotables data file.

* Go code to read a conforming gotables data file and populate a FlatBuffers []byte array is auto-generated.

* Go code to read the FlatBuffers []byte array is auto-generated.

* The read step is VERY fast. There is no additional code between you and the auto-generated FlatBuffers code.
  [The read step at this stage is read-only.]

* You write only your own code to call these auto-generated methods, and denormalise the data from tabular to
  your prefered data structures.

## FlatTables is subset of Google FlatBuffers as a binary format for gotables Table objects.

* FlatTables is general purpose because it consists of tables, and your own data is probably capable of being
  normalised (in Ted Codd, C J Date fashion) to one or more tables ready for transmission and re-assembly
  at the receiving end.

## How the flattables_sample repository was auto-generated

1. Create directory `flattables_sample`

    $ mkdir flattables_sample

2. Create a file containing one or more gotables tables. The tables don't need to contain data, but let's include some
   data and use the same file for running our tests. We'll call it "tables.got" (.got is for gotables).

    # tables.got
    
    [MyAbcTable]
    a       b       c       d		e
    int64   byte    int16   float32	bool
    1       2       3       3		true
    11      22      33      3.3		true
    111     222     333     3.33	false
    2       4       8       8.0		false
    
    [MyXyzTable]
    x       y       z
    int64   int32   int64
    4       5       6
    44      55      66
    444     555     666
    4444    5555    6666
    16      32      64
    
    [MyStrTable]
    s               i       ss
    string          int8    string
    "Fred"          0       "The rain ..."
    "Wilma"         1       "in Spain ..."
    "Barney"        2       "falls mainly ..."
    "Betty"         3       "on the ..."
    "Bam Bam"       4       "plain."
    "Pebbles"       5       "Why?"
    "Grand Poobah"  6       "Why not!"
    "Dino"          7       "Dinosaur"

Check its validity with gotsyntax:

    $ gotsyntax tables.got
    tables.got (syntax okay)

The FlatTables utility `gotft` will also do a validity check, but you might as well get earlier feedback.

3. Run the FlatTables utility `gotft` (gotables flattables).

    $ gotft -f tables.got -n flattables_sample
    (1) Reading: tables.got (gotables file)
    (1) Setting gotables.TableSet name to "flattables_sample" (from -n flattables_sample)
    (2) FlatTables: Generating FlatBuffers schema file: ../flattables_sample/flattables_sample.fbs (from TableSet "flattables_sample")
    *** FlatTables: Adding table [MyAbcTable] to FlatBuffers schema
    *** FlatTables: Adding table [MyXyzTable] to FlatBuffers schema
    *** FlatTables: Adding table [MyStrTable] to FlatBuffers schema
    (3) flatc generating official Go FlatBuffers code: flatc --go -o .. ../flattables_sample/flattables_sample.fbs
    (3) Generated: dir ../flattables_sample (flatc generated Go code here for user-defined types)
    (4) FlatTables generating unofficial Go FlatTables code: ../flattables_sample/flattables_sample_flattables.go
    (4) FlatTables generating Go FlatTables test code: ../flattables_sample/flattables_sample_test.go
    (*) DONE

The following files are generated. Some by Google FlatBuffers flatc (which is called by gotft), and some by FlatTables,
mainly code to link gotables to flattables (a constrained flatbuffers).

    flattables_sample.fbs (from flattables)
    MyAbcTable.go (from flatc)
    MyXyzTable.go (from flatc)
    MyStrTable.go (from flatc)
    FlatTables.go (from flatc)
    flattables_sample_flattables.go (from flattables)
    flattables_sample_test.go (from flattables)

You did not have to write the .fbs flatbuffers schema `flattables_sample.fbs`. It was done for you.

You did not have to write the glue code to get data from `tables.got` to a flatbuffers []byte array.
And if you wish to populate the flatbuffers []byte array yourself, and not go via gotables, just
follow the setter calls in `flattables_sample_flattables.go` to get you going. In that case, you could use
the gotables `tables.got` file purely for generating the schema and setter methods. That would be faster.
