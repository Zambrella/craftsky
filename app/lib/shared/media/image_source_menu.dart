import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

bool supportsCameraImageSource({
  required bool isWeb,
  required TargetPlatform platform,
}) =>
    isWeb ||
    platform == TargetPlatform.android ||
    platform == TargetPlatform.iOS;

bool get cameraImageSourceSupported =>
    supportsCameraImageSource(isWeb: kIsWeb, platform: defaultTargetPlatform);

Future<void> showImageSourceMenu(
  BuildContext context, {
  required Future<void> Function()? onChoosePhotos,
  required Future<void> Function()? onTakePhoto,
  Future<void> Function()? onChooseVideo,
  String keyPrefix = 'image-source',
  bool? cameraSupported,
}) {
  final l10n = AppLocalizations.of(context);
  final showCamera =
      (cameraSupported ?? cameraImageSourceSupported) && onTakePhoto != null;
  if (!showCamera && onChooseVideo == null) {
    return onChoosePhotos?.call() ?? Future<void>.value();
  }
  return showCraftskyContextMenu(
    context,
    position: craftskyContextMenuAnchorPosition(context),
    groups: [
      CraftskyContextMenuGroup(
        items: [
          if (showCamera)
            CraftskyContextMenuItem(
              key: Key('$keyPrefix-take-photo'),
              text: l10n.postComposeTakePhoto,
              icon: Icons.photo_camera_outlined,
              onPressed: onTakePhoto,
            ),
          CraftskyContextMenuItem(
            key: Key('$keyPrefix-choose-photos'),
            text: l10n.postComposeChoosePhotos,
            icon: Icons.photo_library_outlined,
            onPressed: onChoosePhotos,
          ),
          if (onChooseVideo != null)
            CraftskyContextMenuItem(
              key: Key('$keyPrefix-choose-video'),
              text: l10n.postComposeChooseVideo,
              icon: Icons.video_library_outlined,
              onPressed: onChooseVideo,
            ),
        ],
      ),
    ],
  );
}
