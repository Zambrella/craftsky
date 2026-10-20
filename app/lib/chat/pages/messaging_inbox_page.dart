import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class MessagingInboxPage extends ConsumerWidget {
  const new({
    required this.account,
    required this.onOpen,
    required this.onNewMessage,
    required this.onSettings,
    required this.onAuthorize,
    this.initialBucket = ChatBucket.accepted,
    this.onClose,
    super.key,
  });

  final AccountKey account;
  final ChatBucket initialBucket;
  final ValueChanged<ChatConversationId> onOpen;
  final VoidCallback onNewMessage;
  final VoidCallback onSettings;
  final ValueChanged<bool> onAuthorize;
  final VoidCallback? onClose;

  void _owned(WidgetRef ref, VoidCallback action) {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (lease?.session.account != account) return;
    ActiveAccountOperation.fromWidget(ref, lease).publish(action);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return ChatContentGate(
      account: account,
      onAuthorize: onAuthorize,
      onClose: onClose,
      builder: (context, ref, status) {
        final counts = ref.watch(chatUnreadCountsProvider(account)).value;
        Widget tab(String title, int? count, String? label) => Tab(
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Flexible(child: Text(title, overflow: TextOverflow.ellipsis)),
              if (count != null && count > 0) ...[
                const SizedBox(width: 8),
                Badge(label: Text(label!)),
              ],
            ],
          ),
        );
        return DefaultTabController(
          key: ValueKey((account, status.generation)),
          length: 2,
          initialIndex: initialBucket == ChatBucket.request ? 1 : 0,
          child: Scaffold(
            appBar: AppBar(
              leading: onClose == null ? null : BackButton(onPressed: onClose),
              title: Text(l10n.chatInboxTitle),
              actions: [
                IconButton(
                  tooltip: l10n.chatNewMessage,
                  icon: const Icon(Icons.edit_outlined),
                  onPressed:
                      status.canSend &&
                          status.state == ChatConnectionState.ready
                      ? () => _owned(ref, onNewMessage)
                      : null,
                ),
                IconButton(
                  tooltip: l10n.chatSettingsTitle,
                  icon: const Icon(Icons.settings_outlined),
                  onPressed: () => _owned(ref, onSettings),
                ),
              ],
              bottom: TabBar(
                tabs: [
                  tab(
                    l10n.chatInboxTab,
                    counts?.accepted,
                    counts?.acceptedLabel,
                  ),
                  tab(
                    l10n.chatRequestsTab,
                    counts?.requests,
                    counts?.requestsLabel,
                  ),
                ],
              ),
            ),
            body: TabBarView(
              children: [
                for (final bucket in ChatBucket.values)
                  _Bucket(
                    account: account,
                    bucket: bucket,
                    onOpen: (id) => _owned(ref, () => onOpen(id)),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _Bucket extends ConsumerWidget {
  const new({
    required this.account,
    required this.bucket,
    required this.onOpen,
  });
  final AccountKey account;
  final ChatBucket bucket;
  final ValueChanged<ChatConversationId> onOpen;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final family = chatInboxControllerProvider(account, bucket);
    return ref
        .watch(family)
        .when(
          skipLoadingOnRefresh: false,
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (_, _) => ChatFailurePanel(
            message: l10n.chatUnavailable,
            onRetry: () => ref.invalidate(family),
          ),
          data: (controller) => ListenableBuilder(
            listenable: controller,
            builder: (context, _) {
              return RefreshIndicator(
                onRefresh: controller.refresh,
                child: ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  children: [
                    if (controller.loading) const LinearProgressIndicator(),
                    if (controller.failed)
                      ChatFailurePanel(
                        message: l10n.chatUnavailable,
                        onRetry: () => unawaited(controller.refresh()),
                      ),
                    if (controller.items.isEmpty &&
                        !controller.loading &&
                        !controller.failed)
                      Padding(
                        padding: const EdgeInsets.all(32),
                        child: Center(
                          child: Text(
                            bucket == ChatBucket.accepted
                                ? l10n.chatEmptyInbox
                                : l10n.chatEmptyRequests,
                          ),
                        ),
                      ),
                    for (final conversation in controller.items)
                      ListTile(
                        key: ValueKey(conversation.id),
                        leading: const CircleAvatar(
                          child: Icon(Icons.person_outline),
                        ),
                        title: Text(
                          conversation.peer.displayName ??
                              conversation.peer.handle ??
                              l10n.chatSomeone,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: conversation.lastMessage == null
                            ? null
                            : Text(
                                switch (conversation.lastMessage!.kind) {
                                  ChatMessageKind.text =>
                                    conversation.lastMessage!.text!,
                                  ChatMessageKind.deleted =>
                                    l10n.chatMessageDeleted,
                                  ChatMessageKind.unsupported =>
                                    l10n.chatMessageUnsupported,
                                },
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            if (conversation.muted)
                              Icon(
                                Icons.notifications_off_outlined,
                                semanticLabel: l10n.chatMuted,
                              ),
                            if (conversation.unreadCount > 0)
                              Padding(
                                padding: const EdgeInsets.only(left: 8),
                                child: Badge(
                                  label: Text(
                                    conversation.unreadCount >= 100
                                        ? '99+'
                                        : '${conversation.unreadCount}',
                                  ),
                                ),
                              ),
                          ],
                        ),
                        onTap: () => onOpen(conversation.id),
                      ),
                    if (controller.loadingMore)
                      const Padding(
                        padding: EdgeInsets.all(16),
                        child: Center(child: CircularProgressIndicator()),
                      ),
                    if (controller.hasMore && !controller.loadingMore)
                      TextButton(
                        onPressed: () => unawaited(controller.loadMore()),
                        child: Text(l10n.chatLoadMore),
                      ),
                  ],
                ),
              );
            },
          ),
        );
  }
}
