import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/messaging_consent_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_sync_timestamp.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Private content is reconstructed only for the current authorized account.
class ChatContentGate extends ConsumerWidget {
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
  Widget build(BuildContext context, WidgetRef ref) {
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
              const Scaffold(body: Center(child: CircularProgressIndicator())),
          error: (_, _) => retry(l10n.chatUnavailable),
          data: (status) {
            if (status.state == ChatConnectionState.disabled ||
                status.state == ChatConnectionState.authorizationNeeded) {
              return MessagingConsentPage(
                key: ValueKey(account),
                initialPushEnabled: status.pushEnabled,
                onClose: onClose,
                onAuthorize: (push) {
                  if (ref
                          .read(sessionRegistryProvider)
                          .value
                          ?.activeLease
                          ?.session
                          .account !=
                      account) {
                    return;
                  }
                  ActiveAccountOperation.fromWidget(
                    ref,
                    lease,
                  ).publish(() => onAuthorize(push));
                },
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
            return Column(
              children: [
                if (degraded)
                  Material(
                    child: SafeArea(
                      bottom: false,
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Row(
                          children: [
                            Expanded(
                              child: Semantics(
                                liveRegion: true,
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(l10n.chatSyncStale),
                                    ChatSyncTimestamp(at: status.lastSuccessAt),
                                  ],
                                ),
                              ),
                            ),
                            TextButton(
                              onPressed: () =>
                                  ref.invalidate(chatStatusProvider(account)),
                              child: Text(l10n.chatRefresh),
                            ),
                          ],
                        ),
                      ),
                    ),
                  )
                else
                  const SizedBox.shrink(),
                Expanded(child: builder(context, ref, status)),
              ],
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
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: 12),
          TextButton(
            onPressed: onRetry,
            child: Text(AppLocalizations.of(context).chatRefresh),
          ),
        ],
      ),
    ),
  );
}
