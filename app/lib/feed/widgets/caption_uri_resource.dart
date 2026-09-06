import 'package:craftsky_app/feed/widgets/caption_uri_resource_handle.dart';
import 'package:craftsky_app/feed/widgets/caption_uri_resource_stub.dart'
    if (dart.library.io) 'caption_uri_resource_native.dart'
    if (dart.library.js_interop) 'caption_uri_resource_web.dart'
    as platform;

export 'caption_uri_resource_handle.dart';

Future<CaptionUriResource> createCaptionUriResource(String data) =>
    platform.createCaptionUriResource(data);
