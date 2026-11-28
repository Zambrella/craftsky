# Instagram archive reader

Only `connections/followers_and_following/following.json` is read. Archive paths
are never resolved as filesystem paths. Choosing an export starts local
processing. Only normalized usernames are sent to AppView after parsing.

The ZIP reader preflights the end records before allocating a directory, then
uses zip.js strict local/central-header, overlapping-entry, ZIP64 and CRC checks.
Limits are 100,000 directory entries, 64 MiB of directory data and 20 MiB of actual
following-list output. Directory entries cannot bypass the output limit by
lying about uncompressed sizes. The reader rejects symlinks and an encrypted
following-list entry.

Following exports are parsed in a local browser worker, released on completion,
cancellation or error. Stored, Deflate and Deflate64 use zip.js. ZIP method 12 uses a registered BZip2
codec with the pinned `seek-bzip` decoder. Its input/output adapters accept only
byte arrays, throw on truncated input, require exact decoded size, and cap
compressed input at 40 MB. The decoder verifies BZip2 block and stream CRCs;
zip.js separately verifies the ZIP CRC. The BZip2 decoder runs in its own module
worker and is terminated after completion, cancellation, error or a
30-second deadline. Only that Worker loads the browser `buffer` implementation
needed by the decoder's small internal tables; no Node filesystem or network
compatibility layer is used.

Implementation references: [zip.js custom codecs](https://gildas-lormeau.github.io/zip.js/api/functions/registerCodec.html),
[seek-bzip source and stream interface](https://github.com/cscott/seek-bzip),
[browser Buffer byte arrays](https://github.com/feross/buffer).

Tests use an actual Python-standard-library BZip2 ZIP fixture, exercise both
checksum layers, header mismatches, false size claims, truncation, cancellation
and deadline termination. The browser import scenario exercises the bundled
Worker; Node codec tests alone do not establish that browser integration.
