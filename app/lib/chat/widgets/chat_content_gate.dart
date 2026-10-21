import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/chat_settings_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Private content is reconstructed only for the current authorized account.
class ChatContentGate extends ConsumerStatefulWidget {
  const new({
    required this.account,
    required this.onAuthorize,
    required this.builder,
    this.onClose,
    super.key,
  });
  final AccountKey account;
  final VoidCallback? onClose;
  final ValueChanged<bool> onAuthorize;
  final Widget Function(BuildContext, WidgetRef, ChatStatus) builder;

  @override
  ConsumerState<ChatContentGate> createState() => _ChatContentGateState();
}

class _ChatContentGateState extends ConsumerState<ChatContentGate> {
  bool _showSetup = false;

  @override
  void didUpdateWidget(covariant ChatContentGate oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.account != widget.account) _showSetup = false;
  }

  @override
  Widget build(BuildContext context) {
    final account = widget.account;
    final onClose = widget.onClose;
    final lease = ref.watch(
      sessionRegistryProvider.select((value) => value.value?.activeLease),
    );
    if (lease?.session.account != account) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    Widget retry(String message) => Scaffold(
      appBar: AppBar(
        leading: onClose == null ? null : BackButton(onPressed: onClose),
        title: Text(l10n.chatInboxTitle),
      ),
      body: ChatFailurePanel(
        message: message,
        onRetry: () => ref.invalidate(chatStatusProvider(account)),
      ),
    );
    return ref
        .watch(chatStatusProvider(account))
        .when(
          loading: () =>
              ChatLoadingPage(title: l10n.chatInboxTitle, onClose: onClose),
          error: (_, _) => retry(l10n.chatUnavailable),
          data: (status) {
            if (status.state == ChatConnectionState.disabled ||
                status.state == ChatConnectionState.authorizationNeeded) {
              _showSetup = true;
            }
            if (_showSetup) {
              return MessagingSettingsPage(
                key: ValueKey(lease),
                account: account,
                onClose: onClose,
                onAuthorize: widget.onAuthorize,
                onOpenMessages: () => setState(() => _showSetup = false),
              );
            }
            final degraded =
                status.state == ChatConnectionState.unavailable &&
                status.canRead;
            if (status.state != ChatConnectionState.ready && !degraded) {
              return retry(
                status.state == ChatConnectionState.restricted
                    ? l10n.chatRestricted
                    : l10n.chatUnavailable,
              );
            }
            return ChatMessageEffect(
              message: degraded ? l10n.chatSyncStale : null,
              severity: MessageSeverity.warning,
              action: MessageAction(
                label: l10n.chatRefresh,
                onPressed: () => ref.invalidate(chatStatusProvider(account)),
              ),
              child: widget.builder(context, ref, status),
            );
          },
        );
  }
}

class ChatFailurePanel extends StatelessWidget {
  const new({required this.message, required this.onRetry, super.key});
  final String message;
  final VoidCallback onRetry;
  @override
  Widget build(BuildContext context) => CraftskyEmptyState(
    icon: CraftskyIcons.error,
    title: AppLocalizations.of(context).chatInboxTitle,
    subtitle: message,
    actionLabel: AppLocalizations.of(context).chatRefresh,
    onAction: onRetry,
  );
}
