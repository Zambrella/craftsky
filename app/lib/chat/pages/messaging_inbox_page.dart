import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_inbox_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_leading.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/chat/widgets/chat_peer_profile.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class MessagingInboxPage extends ConsumerWidget {
  const new({
    required this.account,
    required this.onOpen,
    required this.onNewMessage,
    required this.onRequests,
    required this.onSettings,
    required this.onAuthorize,
    this.bucket = ChatBucket.accepted,
    this.onClose,
    super.key,
  });
  final AccountKey account;
  final ChatBucket bucket;
  final ValueChanged<ChatConversationId> onOpen;
  final VoidCallback onNewMessage;
  final VoidCallback onRequests;
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
      loadingBody: const ChatListSkeleton(),
      onAuthorize: onAuthorize,
      onClose: onClose,
      builder: (context, ref, status) {
        final counts = ref.watch(displayedChatUnreadCountsProvider(account));
        final family = chatInboxControllerProvider(account, bucket);
        Future<void> refresh(ChatInboxController controller) async {
          await controller.refresh();
          if (context.mounted) {
            _owned(
              ref,
              () => ref.invalidate(chatUnreadCountsProvider(account)),
            );
          }
        }

        Widget page({ChatInboxController? controller, Widget? error}) =>
            Scaffold(
              key: ValueKey((account, bucket, status.generation)),
              appBar: AppBar(
                leading: chatAppBarLeading(context, onClose),
                title: Text(
                  bucket == ChatBucket.request
                      ? l10n.chatRequestsTab
                      : l10n.chatInboxTitle,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                actions: [
                  if (bucket == ChatBucket.accepted)
                    IconButton(
                      tooltip: l10n.chatRequestsTab,
                      onPressed: () => _owned(ref, onRequests),
                      icon: Badge(
                        isLabelVisible: (counts?.requests ?? 0) > 0,
                        label: Text(counts?.requestsLabel ?? '0'),
                        child: const Icon(CraftskyIcons.email),
                      ),
                    ),
                  if (bucket == ChatBucket.accepted)
                    IconButton(
                      tooltip: l10n.chatSettingsTitle,
                      icon: const Icon(CraftskyIcons.settings),
                      onPressed: () => _owned(ref, onSettings),
                    ),
                ],
              ),
              floatingActionButton: bucket == ChatBucket.accepted
                  ? CraftskyFloatingActionButton.extended(
                      tooltip: l10n.chatNewMessage,
                      onPressed:
                          status.canSend &&
                              status.state == ChatConnectionState.ready
                          ? () => _owned(ref, onNewMessage)
                          : null,
                      icon: const Icon(CraftskyIcons.edit),
                      label: Text(l10n.chatNewMessage),
                    )
                  : null,
              body:
                  error ??
                  (controller == null
                      ? const ChatListSkeleton()
                      : _Bucket(
                          controller: controller,
                          onRefresh: () => refresh(controller),
                          onOpen: (id) => _owned(ref, () => onOpen(id)),
                        )),
            );
        return ref
            .watch(family)
            .when(
              skipLoadingOnRefresh: false,
              loading: page,
              error: (_, _) => page(
                error: ChatFailurePanel(
                  message: l10n.chatUnavailable,
                  onRetry: () => ref.invalidate(family),
                ),
              ),
              data: (controller) => ListenableBuilder(
                listenable: controller,
                builder: (_, _) => page(controller: controller),
              ),
            );
      },
    );
  }
}

class _Bucket extends StatelessWidget {
  const new({
    required this.controller,
    required this.onOpen,
    required this.onRefresh,
  });
  final ChatInboxController controller;
  final ValueChanged<ChatConversationId> onOpen;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final bucket = controller.bucket;
    return YarnRefreshScrollView.list(
      onRefresh: onRefresh,
      // The controller retains errors rather than throwing to the refresh UI.
      showSuccess: false,
      padding: EdgeInsets.only(bottom: bucket == ChatBucket.accepted ? 96 : 24),
      children: [
        if (controller.failed)
          ChatFailurePanel(
            message: l10n.chatUnavailable,
            onRetry: () => unawaited(onRefresh()),
          ),
        if (controller.items.isEmpty && !controller.failed)
          CraftskyEmptyState(
            icon: bucket == ChatBucket.request
                ? CraftskyIcons.email
                : CraftskyIcons.comment,
            title: bucket == ChatBucket.accepted
                ? l10n.chatInboxTab
                : l10n.chatRequestsTab,
            subtitle: bucket == ChatBucket.accepted
                ? l10n.chatEmptyInbox
                : l10n.chatEmptyRequests,
          ),
        for (final conversation in controller.items)
          ChatPeerProfile(
            key: ValueKey(conversation.id),
            peer: conversation.peer,
            builder: (context, avatar, identity, _) => ListTile(
              horizontalTitleGap: 12,
              minLeadingWidth: 0,
              leading: avatar,
              title: LayoutBuilder(
                builder: (context, constraints) {
                  final message = conversation.lastMessage;
                  if (message == null) return identity;
                  final timestamp = RelativeTimeText(
                    timestamp: message.sentAt,
                    textAlign: TextAlign.end,
                  );
                  // Keep identity and full date readable at large text sizes.
                  if (constraints.maxWidth < 320 &&
                      MediaQuery.textScalerOf(context).scale(12) > 18) {
                    return Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        identity,
                        Align(
                          alignment: Alignment.centerRight,
                          child: FittedBox(
                            fit: BoxFit.scaleDown,
                            child: timestamp,
                          ),
                        ),
                      ],
                    );
                  }
                  return Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(child: identity),
                      const SizedBox(width: 8),
                      ConstrainedBox(
                        constraints: const BoxConstraints(minWidth: 48),
                        child: Align(
                          widthFactor: 1,
                          alignment: Alignment.centerRight,
                          child: timestamp,
                        ),
                      ),
                    ],
                  );
                },
              ),
              subtitle: Row(
                children: [
                  Expanded(
                    child: conversation.lastMessage == null
                        ? const SizedBox.shrink()
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
                  ),
                  if (conversation.muted)
                    Icon(CraftskyIcons.muted, semanticLabel: l10n.chatMuted),
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
          ),
        if (controller.hasMore && !controller.loadingMore)
          TextButton(
            onPressed: () => unawaited(controller.loadMore()),
            child: Text(l10n.chatLoadMore),
          ),
      ],
    );
  }
}
