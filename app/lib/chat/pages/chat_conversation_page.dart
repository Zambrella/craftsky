import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/chat/adapters/flutter_chat_adapter.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_conversation_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/chat/widgets/chat_message_preview.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class MessagingConversationPage extends ConsumerWidget {
  const new({
    required this.account,
    required this.conversationId,
    required this.onClose,
    required this.onReport,
    required this.onAuthorize,
    super.key,
  });
  final AccountKey account;
  final ChatConversationId conversationId;
  final VoidCallback onClose;
  final ValueChanged<ChatMessageId> onReport;
  final ValueChanged<bool> onAuthorize;

  Future<void> _close(
    WidgetRef ref,
    ChatConversationController controller,
  ) async {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (lease?.session.account != account) return;
    final operation = ActiveAccountOperation.fromWidget(ref, lease);
    try {
      await operation.step(controller.flush);
      if (!controller.saveFailed) operation.publish(onClose);
    } on AccountOperationDiscarded {
      return;
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) => ChatContentGate(
    account: account,
    onAuthorize: onAuthorize,
    onClose: onClose,
    builder: (context, ref, status) {
      final family = chatConversationControllerProvider(
        account,
        conversationId,
      );
      return ref
          .watch(family)
          .when(
            skipLoadingOnRefresh: false,
            loading: () => const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            ),
            error: (_, _) => Scaffold(
              appBar: AppBar(
                title: Text(AppLocalizations.of(context).chatInboxTitle),
              ),
              body: ChatFailurePanel(
                message: AppLocalizations.of(context).chatUnavailable,
                onRetry: () => ref.invalidate(family),
              ),
            ),
            data: (controller) => _ConversationView(
              key: ValueKey((account, conversationId, status.generation)),
              controller: controller,
              onClose: () => unawaited(_close(ref, controller)),
              onReload: () => ref.invalidate(family),
              canOperate: status.state == ChatConnectionState.ready,
              onReport:
                  status.canReport && status.state == ChatConnectionState.ready
                  ? onReport
                  : null,
            ),
          );
    },
  );
}

class _ConversationView extends ConsumerStatefulWidget {
  const new({
    required this.controller,
    required this.onClose,
    required this.onReload,
    required this.onReport,
    required this.canOperate,
    super.key,
  });
  final ChatConversationController controller;
  final VoidCallback onClose;
  final VoidCallback onReload;
  final ValueChanged<ChatMessageId>? onReport;
  final bool canOperate;
  @override
  ConsumerState<_ConversationView> createState() => _ConversationViewState();
}

class _ConversationViewState extends ConsumerState<_ConversationView>
    with WidgetsBindingObserver {
  late final TextEditingController _text;
  late final UnsavedWorkGuard _workGuard;
  UnsavedWorkRegistration? _workRegistration;
  bool _resumed = true;
  bool _routeCurrent = true;

  @override
  void initState() {
    super.initState();
    _text = TextEditingController(text: widget.controller.draft.text);
    _workGuard = ref.read(unsavedWorkGuardProvider);
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (lease?.session.account == widget.controller.account) {
      _workRegistration = _workGuard.register(
        owner: lease!.session,
        isDirty: () =>
            widget.controller.draft.text.isNotEmpty ||
            widget.controller.draft.pending != null ||
            widget.controller.saveFailed,
        confirmAndClose: () async {
          await widget.controller.flush();
          return !widget.controller.saveFailed;
        },
      );
    }
    _resumed =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    widget.controller.addListener(_updated);
    WidgetsBinding.instance.addObserver(this);
  }

  void _updated() {
    if (!mounted) return;
    final text = widget.controller.draft.text;
    if (_text.text != text) {
      _text.value = TextEditingValue(
        text: text,
        selection: TextSelection.collapsed(offset: text.length),
      );
    }
    setState(() {});
  }

  Future<void> _sendAgain() async {
    final controller = widget.controller;
    final uncertain =
        controller.draft.pending?.state == ChatDraftSendState.uncertain;
    if (uncertain) {
      final l10n = AppLocalizations.of(context);
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(l10n.chatSendAgain),
          content: Text(l10n.chatSendAgainWarning),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(l10n.actionCancel),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(l10n.chatSendAgain),
            ),
          ],
        ),
      );
      if (confirmed != true || !mounted) return;
    }
    if (!widget.canOperate) return;
    await controller.sendAgain(duplicateRiskAcknowledged: uncertain);
  }

  void _openLink(Uri uri) {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (lease?.session.account != widget.controller.account) return;
    final operation = ActiveAccountOperation.fromWidget(ref, lease);
    final target = ChatLinkTarget.fromUri(uri);
    if (target != null) {
      operation.publish(() {
        switch (target.kind) {
          case ChatLinkKind.post:
            unawaited(
              PostThreadRoute(
                did: target.did.value,
                rkey: target.rkey!.value,
                focus: target.focus?.value,
              ).push<void>(context),
            );
          case ChatLinkKind.profile:
            unawaited(UserProfileRoute(did: target.did).push<void>(context));
          case ChatLinkKind.event:
            unawaited(
              BusinessEventRoute(
                did: target.did.value,
                rkey: target.rkey!.value,
              ).push<void>(context),
            );
        }
      });
      return;
    }
    unawaited(
      confirmAndLaunchExternalLink(
        context,
        uri: uri,
        launchUrl: (uri) async {
          operation.requireCurrent();
          return await operation.step(() => launchExternalLink(uri));
        },
      ),
    );
  }

  Future<void> _leave() async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.chatLeaveTitle),
        content: Text(l10n.chatLeaveExplanation),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l10n.actionCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l10n.chatLeave),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted || !widget.canOperate) return;
    if (await widget.controller.leave() && mounted) widget.onClose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _resumed = state == AppLifecycleState.resumed;
    widget.controller.setForeground(
      value: _resumed && _routeCurrent && widget.canOperate,
    );
    if (!_resumed) unawaited(widget.controller.flush());
  }

  @override
  void dispose() {
    _workGuard.unregister(_workRegistration);
    widget.controller.setForeground(value: false);
    widget.controller.removeListener(_updated);
    WidgetsBinding.instance.removeObserver(this);
    _text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final l10n = AppLocalizations.of(context);
    _routeCurrent = ModalRoute.of(context)?.isCurrent ?? true;
    controller.setForeground(
      value: _resumed && _routeCurrent && widget.canOperate,
    );
    final conversation = controller.conversation;
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) widget.onClose();
      },
      child: Scaffold(
        appBar: AppBar(
          leading: BackButton(onPressed: widget.onClose),
          actions: [
            if (conversation != null)
              PopupMenuButton<String>(
                tooltip: l10n.chatConversationActions,
                enabled: !controller.busy && widget.canOperate,
                onSelected: (value) {
                  if (value == 'mute') {
                    unawaited(controller.setMuted(muted: !conversation.muted));
                  }
                  if (value == 'leave') unawaited(_leave());
                },
                itemBuilder: (context) => [
                  PopupMenuItem(
                    value: 'mute',
                    child: Text(
                      conversation.muted ? l10n.chatUnmute : l10n.chatMute,
                    ),
                  ),
                  PopupMenuItem(
                    value: 'leave',
                    child: Text(l10n.chatLeaveTitle),
                  ),
                ],
              ),
          ],
          title: Text(
            conversation?.peer.displayName ??
                conversation?.peer.handle ??
                l10n.chatInboxTitle,
          ),
        ),
        body: conversation == null
            ? ChatFailurePanel(
                message: l10n.chatUnavailable,
                onRetry: widget.onReload,
              )
            : LayoutBuilder(
                builder: (context, constraints) => FlutterChatAdapter(
                  history: controller.history,
                  owner: controller.account.did,
                  onLoadOlder: controller.loadOlder,
                  onReport: widget.onReport,
                  onLink: _openLink,
                  messageFooterBuilder:
                      conversation.bucket == ChatBucket.accepted
                      ? (message) => ChatMessagePreview(
                          account: controller.account,
                          conversationId: controller.id,
                          message: message,
                        )
                      : null,
                  onPresented: (ids) {
                    if (_resumed && _routeCurrent && widget.canOperate) {
                      unawaited(controller.presented(ids));
                    }
                  },
                  composer: ConstrainedBox(
                    constraints: BoxConstraints(
                      maxHeight: constraints.maxHeight * 0.6,
                    ),
                    child: SingleChildScrollView(
                      child: SafeArea(
                        top: false,
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              if (controller.failed) Text(l10n.chatUnavailable),
                              if (controller.saveFailed) ...[
                                Text(l10n.chatDraftSaveFailed),
                                TextButton(
                                  onPressed: () =>
                                      unawaited(controller.flush()),
                                  child: Text(l10n.chatSaveDraft),
                                ),
                              ],
                              if (conversation.bucket ==
                                  ChatBucket.request) ...[
                                Text(l10n.chatRequestExplanation),
                                const SizedBox(height: 8),
                                FilledButton(
                                  onPressed:
                                      controller.busy || !widget.canOperate
                                      ? null
                                      : () => unawaited(controller.accept()),
                                  child: Text(l10n.chatAcceptRequest),
                                ),
                              ] else ...[
                                Row(
                                  crossAxisAlignment: CrossAxisAlignment.end,
                                  children: [
                                    Expanded(
                                      child: TextField(
                                        controller: _text,
                                        onChanged: controller.edit,
                                        minLines: 1,
                                        maxLines: 5,
                                        keyboardType: TextInputType.multiline,
                                        decoration: InputDecoration(
                                          labelText: l10n.chatComposeLabel,
                                          errorText:
                                              _text.text.isNotEmpty &&
                                                  validateMessageText(
                                                        _text.text,
                                                      ) !=
                                                      null
                                              ? l10n.chatMessageInvalid
                                              : null,
                                        ),
                                      ),
                                    ),
                                    IconButton(
                                      tooltip: l10n.chatSend,
                                      onPressed:
                                          controller.canSend &&
                                              widget.canOperate
                                          ? () => unawaited(controller.send())
                                          : null,
                                      icon: const Icon(Icons.send_outlined),
                                    ),
                                  ],
                                ),
                                if (!conversation.canSend)
                                  Text(l10n.chatCannotSend),
                                if (controller.draft.pending
                                    case final attempt?) ...[
                                  const SizedBox(height: 8),
                                  Text(switch (attempt.state) {
                                    ChatDraftSendState.pending =>
                                      l10n.chatSending,
                                    ChatDraftSendState.failed =>
                                      l10n.chatSendFailed,
                                    ChatDraftSendState.uncertain =>
                                      l10n.chatSendUncertain,
                                  }),
                                  if (attempt.state !=
                                      ChatDraftSendState.pending)
                                    Wrap(
                                      children: [
                                        TextButton(
                                          onPressed:
                                              controller.canSendAgain &&
                                                  widget.canOperate
                                              ? () => unawaited(_sendAgain())
                                              : null,
                                          child: Text(l10n.chatSendAgain),
                                        ),
                                        TextButton(
                                          onPressed:
                                              controller.busy ||
                                                  !widget.canOperate
                                              ? null
                                              : () => unawaited(
                                                  controller.checkReceipt(),
                                                ),
                                          child: Text(l10n.chatCheckStatus),
                                        ),
                                        TextButton(
                                          onPressed: controller.busy
                                              ? null
                                              : () => unawaited(
                                                  controller.discardDraft(),
                                                ),
                                          child: Text(l10n.chatDiscardDraft),
                                        ),
                                      ],
                                    ),
                                ],
                              ],
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
      ),
    );
  }
}
