import 'package:craftsky_app/chat/widgets/chat_recipient_picker.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:material_ui/material_ui.dart' show Theme;
import 'package:share_plus/share_plus.dart';

List<CraftskyContextMenuItem> linkSharingMenuItems(
  BuildContext context,
  String url, {
  bool canSendViaChat = true,
  VoidCallback? onSendViaChat,
}) => [
  copyLinkMenuItem(context, url),
  shareLinkMenuItem(context, url),
  if (canSendViaChat)
    CraftskyContextMenuItem(
      text: AppLocalizations.of(context).chatSendViaChat,
      icon: CraftskyIcons.send,
      onPressed: onSendViaChat ?? () => showSendViaChat(context, url),
    ),
];

CraftskyContextMenuItem copyLinkMenuItem(BuildContext context, String url) {
  final l10n = AppLocalizations.of(context);
  return CraftskyContextMenuItem(
    text: l10n.recordCopyLinkAction,
    icon: CraftskyIcons.link,
    onPressed: () => copyRecordLink(context, url),
  );
}

Future<void> copyRecordLink(BuildContext context, String url) async {
  final l10n = AppLocalizations.of(context);
  try {
    await Clipboard.setData(ClipboardData(text: url));
  } on PlatformException {
    if (context.mounted) context.showError(l10n.recordCopyLinkFailed);
    return;
  }
  if (context.mounted) context.showInfo(l10n.recordLinkCopied);
}

CraftskyContextMenuItem shareLinkMenuItem(BuildContext context, String url) {
  return CraftskyContextMenuItem(
    text: AppLocalizations.of(context).recordShareLinkAction,
    icon: CraftskyIconsBold.shareForPlatform(Theme.of(context).platform),
    onPressed: () => shareRecordLink(context, url),
  );
}

/// Shares the same canonical URL as Copy link, including any reply focus.
/// Results describe the chooser interaction, not confirmed delivery, so neither
/// completion nor cancellation produces a success message.
Future<void> shareRecordLink(BuildContext context, String url) async {
  if (!context.mounted) return;
  final l10n = AppLocalizations.of(context);
  final renderObject = context.findRenderObject();
  final screen = Offset.zero & MediaQuery.sizeOf(context);
  Rect? origin;
  if (renderObject is RenderBox && renderObject.hasSize) {
    final visible =
        (renderObject.localToGlobal(Offset.zero) & renderObject.size).intersect(
          screen,
        );
    if (!visible.isEmpty) origin = visible;
  }
  // Rows may have scrolled offscreen while their menu was open. Keep the iPad
  // popover's source rectangle inside the visible view in that case.
  origin ??= Rect.fromCenter(center: screen.center, width: 1, height: 1);
  try {
    await SharePlus.instance.share(
      ShareParams(uri: Uri.parse(url), sharePositionOrigin: origin),
    );
  } on PlatformException {
    if (context.mounted) context.showError(l10n.recordShareLinkFailed);
  }
}
