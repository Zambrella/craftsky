import 'dart:io';

import 'package:craftsky_app/feed/widgets/caption_uri_resource_handle.dart';
import 'package:path_provider/path_provider.dart';

Future<CaptionUriResource> createCaptionUriResource(String data) async {
  final directory = await getTemporaryDirectory();
  final file = File(
    '${directory.path}/craftsky-caption-'
    '${DateTime.now().microsecondsSinceEpoch}-${identityHashCode(data)}.vtt',
  );
  await file.writeAsString(data, flush: true);
  return ManagedCaptionUriResource(file.uri, () async {
    try {
      await file.delete();
    } on FileSystemException {
      // The OS may already have reclaimed the temporary file.
    }
  });
}
