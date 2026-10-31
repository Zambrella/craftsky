import 'package:craftsky_app/chat/widgets/chat_recipient_picker.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

List<CraftskyContextMenuItem> linkSharingMenuItems(
  BuildContext context,
  String url,
) => [
  copyLinkMenuItem(context, url),
  CraftskyContextMenuItem(
    text: AppLocalizations.of(context).chatSendViaChat,
    icon: CraftskyIcons.send,
    onPressed: () => showSendViaChat(context, url),
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
