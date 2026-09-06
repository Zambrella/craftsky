import 'dart:js_interop';

import 'package:craftsky_app/feed/widgets/caption_uri_resource_handle.dart';
import 'package:web/web.dart' as web;

Future<CaptionUriResource> createCaptionUriResource(String data) async {
  final blob = web.Blob(
    <web.BlobPart>[data.toJS].toJS,
    web.BlobPropertyBag(type: 'text/vtt'),
  );
  final value = web.URL.createObjectURL(blob);
  return ManagedCaptionUriResource(Uri.parse(value), () async {
    web.URL.revokeObjectURL(value);
  });
}
