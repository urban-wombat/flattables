# `flattablesc` - FlatBuffers code generator

`flattablesc [-h] [-v] [-d] -f <gotables-file> -n <namespace> -p <package> [-o <out-dir>] [-s <out-dir-main>]`

Run `flattablesc -h` to see usage.

Try especially `-d` dry-run (also turns on verbose) to find out ahead of time what and where generated code will be written.

Generates `FlatBuffers` and `FlatTables` code (to call from your programs) to write and read `FlatBuffers []byte` arrays.

See [urban-wombat/flattables](https://github.com/urban-wombat/flattables#getting-started-with-google-flatbuffers-via-flattables)
for some of the how and why of `flattablesc`
