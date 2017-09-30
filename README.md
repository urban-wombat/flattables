# gotablesflatbuffers
For experiments with gotables and FlatBuffers

A subset of Google FlatBuffers as a binary format for gotables Table objects.

A FlatBuffers schema is generated internally from the self-describing gotables Table format.

A Go interface is implemented by the fbt package and the matching implementations of gotables to provide
a simple API for programmers to work with gotables objects and fbt objects without having to change
mental gears.
