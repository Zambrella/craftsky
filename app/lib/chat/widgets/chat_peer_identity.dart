import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:material_ui/material_ui.dart';

/// Matches the compact profile title's name and secondary handle treatment.
class ChatPeerIdentity extends StatelessWidget {
  const new({required this.peer, this.nameStyle, super.key});

  final ChatActor peer;
  final TextStyle? nameStyle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final handle = ProfileHandle(peer.handle);
    final hasName = peer.displayName?.trim().isNotEmpty ?? false;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          handle.displayLabel(
            displayName: peer.displayName,
            unavailableLabel: l10n.chatSomeone,
          ),
          style: nameStyle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        if (hasName)
          Text(
            handle.currentLabel(unavailableLabel: l10n.handleUnavailable),
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
      ],
    );
  }
}
