import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource_handle.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource_native.dart'
    as platform;

export 'local_video_preview_resource_handle.dart';

Future<LocalVideoPreviewResource> createLocalVideoPreviewResource(
  LocalVideoSelection selection,
) async {
  final uri = selection.localUri;
  if (uri != null && const {'file', 'content'}.contains(uri.scheme)) {
    return LocalVideoPreviewResource(uri, () async {});
  }
  return await platform.materializeLocalVideo(selection);
}
