# Instagram archive reader

Only `connections/followers_and_following/following.json` is read. Archive paths
are never resolved as filesystem paths and no imported content is sent anywhere
until the owner reviews the usernames and explicitly starts an import.

The ZIP reader preflights the end records before allocating a directory, then
uses zip.js strict local/central-header, overlapping-entry, ZIP64 and CRC checks.
Limits are 100,000 directory entries, 64 MB of directory data and 20 MB of actual
following-list output. Directory entries cannot bypass the output limit by
lying about uncompressed sizes. Encrypted entries and symlinks are rejected.

Stored, Deflate and Deflate64 use zip.js. ZIP method 12 uses a registered BZip2
codec with the pinned `seek-bzip` decoder. Its input/output adapters accept only
byte arrays, throw on truncated input, require exact decoded size, and cap
compressed input at 40 MB. The decoder verifies BZip2 block and stream CRCs;
zip.js separately verifies the ZIP CRC. A dedicated module Worker keeps parsing
off the main thread and is terminated after completion, cancellation, error or a
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
