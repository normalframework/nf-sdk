
Decode WAL Segments
====


This utility can be used to decode the buffered data segments stored
in `/var/nf/data`.  In some situations after an outage, there may be a
lot of buffered data segments in this directly, and loading it
directly from the files may be faster than retrieving it via the API.
It also makes it possible to retrieve buffered data from offline instances.

To use, simply edit `bootstrap.sh` to refer to your log segment files
after installing the Normal Protobuf library as shown.

