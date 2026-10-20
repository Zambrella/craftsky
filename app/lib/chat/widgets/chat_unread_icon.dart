import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Native accepted-conversation aggregate; requests have a separate inbox tab.
class ChatUnreadIcon extends ConsumerWidget {
  const new({required this.icon, super.key});
  final IconData icon;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final account = ref.watch(
      sessionRegistryProvider.select(
        (value) => value.value?.activeLease?.session.account,
      ),
    );
    final counts = account == null
        ? null
        : ref.watch(chatUnreadCountsProvider(account)).value;
    if (counts == null || counts.accepted == 0) return Icon(icon);
    return Semantics(
      label: AppLocalizations.of(context)
          .chatUnreadConversations(counts.acceptedLabel),
      excludeSemantics: true,
      child: Badge(label: Text(counts.acceptedLabel), child: Icon(icon)),
    );
  }
}
