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
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/chat/widgets/chat_message_preview.dart';
import 'package:craftsky_app/chat/widgets/chat_peer_profile.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/toggle_block_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class MessagingConversationPage extends ConsumerWidget {
  const new({
    required this.account,
    required this.conversationId,
    required this.onClose,
    required this.onReport,
    required this.onAuthorize,
    this.onReportConversation,
    super.key,
  });
  final AccountKey account;
  final ChatConversationId conversationId;
  final VoidCallback onClose;
  final ValueChanged<ChatMessageId> onReport;
  final VoidCallback? onReportConversation;
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
    loadingBody: const ChatConversationSkeleton(),
    loadingTitle: const ChatProfileHeaderSkeleton(),
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
            loading: () => ChatLoadingPage(
              title: AppLocalizations.of(context).chatInboxTitle,
              onClose: onClose,
              body: const ChatConversationSkeleton(),
              titleWidget: const ChatProfileHeaderSkeleton(),
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
              onReportConversation:
                  status.canReport && status.state == ChatConnectionState.ready
                  ? onReportConversation
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
    this.onReportConversation,
    super.key,
  });
  final ChatConversationController controller;
  final VoidCallback onClose;
  final VoidCallback onReload;
  final ValueChanged<ChatMessageId>? onReport;
  final VoidCallback? onReportConversation;
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
  bool _ignoring = false;
  bool _blocking = false;

  @override
  void initState() {
    super.initState();
    _text = TextEditingController(text: widget.controller.draft.composerText);
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
    final text = widget.controller.draft.composerText;
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
      final confirmed = await showCraftskyConfirmDialog(
        context,
        title: l10n.chatSendAgain,
        message: l10n.chatSendAgainWarning,
        cancelLabel: l10n.actionCancel,
        confirmLabel: l10n.chatSendAgain,
      );
      if (!confirmed || !mounted) return;
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

  void _openPeerProfile(VoidCallback openProfile) {
    final peer = widget.controller.conversation?.peer;
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (peer == null ||
        lease?.session.account != widget.controller.account ||
        !(ModalRoute.of(context)?.isCurrent ?? true)) {
      return;
    }
    ActiveAccountOperation.fromWidget(ref, lease).publish(
      openProfile,
    );
  }

  Future<void> _ignore() async {
    final controller = widget.controller;
    if (!widget.canOperate ||
        controller.busy ||
        controller.conversation?.bucket != ChatBucket.request) {
      return;
    }
    setState(() => _ignoring = true);
    try {
      final left = await controller.leave();
      if (left &&
          mounted &&
          widget.controller == controller &&
          widget.canOperate &&
          (ModalRoute.of(context)?.isCurrent ?? true)) {
        widget.onClose();
      }
    } finally {
      if (mounted) setState(() => _ignoring = false);
    }
  }

  Future<void> _leave() async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showCraftskyDestructiveConfirmDialog(
      context,
      title: l10n.chatLeaveTitle,
      message: l10n.chatLeaveExplanation,
      cancelLabel: l10n.actionCancel,
      confirmLabel: l10n.chatLeave,
    );
    if (!confirmed || !mounted || !widget.canOperate) return;
    if (await widget.controller.leave() && mounted) widget.onClose();
  }

  void _reportConversation() {
    if (!mounted) return;
    final callback = widget.onReportConversation;
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (callback == null ||
        lease?.session.account != widget.controller.account ||
        widget.controller.busy ||
        !widget.canOperate ||
        !(ModalRoute.of(context)?.isCurrent ?? true)) {
      return;
    }
    ActiveAccountOperation.fromWidget(ref, lease).publish(callback);
  }

  Future<void> _blockAccount() async {
    final controller = widget.controller;
    final peer = controller.conversation?.peer;
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (peer == null || lease?.session.account != controller.account) return;
    final operation = ActiveAccountOperation.fromWidget(ref, lease);
    bool canStart() =>
        mounted &&
        operation.isCurrent &&
        widget.controller == controller &&
        widget.canOperate &&
        !controller.busy &&
        !_blocking &&
        !ref.read(toggleBlockProfileProvider).isLoading &&
        (ModalRoute.of(context)?.isCurrent ?? true);
    if (!canStart()) return;
    final l10n = AppLocalizations.of(context);
    final confirmed = await showCraftskyDestructiveConfirmDialog(
      context,
      title: l10n.profileBlockConfirmTitle,
      message: l10n.profileBlockConfirmBody,
      cancelLabel: l10n.actionCancel,
      confirmLabel: l10n.profileBlockAction,
    );
    if (!confirmed || !canStart()) return;
    setState(() => _blocking = true);
    try {
      await operation.step(
        () => ref
            .read(toggleBlockProfileProvider.notifier)
            .toggle(targetDid: peer.did, isBlocking: false),
      );
      if (!mounted ||
          widget.controller != controller ||
          !widget.canOperate ||
          !(ModalRoute.of(context)?.isCurrent ?? true)) {
        return;
      }
      if (ref.read(toggleBlockProfileProvider).asData?.value != true) {
        context.showError(l10n.profileRelationshipError);
        return;
      }
      context.showInfo(l10n.profileBlockSuccess);
      await operation.step(controller.refresh);
    } on AccountOperationDiscarded {
      return;
    } finally {
      if (mounted) setState(() => _blocking = false);
    }
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
    final peer = conversation?.peer;
    final profile = peer == null
        ? null
        : ref.watch(userProfileProvider(peer.did)).value;
    final blockOverlay = peer == null
        ? null
        : ref
              .read(pdsRecordOperationControllerProvider)
              .overlayFor(blockProfileMutationScope(ref.read, peer.did))
              ?.optimisticValue;
    final blocking = blockOverlay is bool
        ? blockOverlay
        : profile?.blocking ?? false;
    final blocked = blocking || profile?.blockedBy == true;
    final blockMutation = ref.watch(toggleBlockProfileProvider);
    final blockBusy = _blocking || blockMutation.isLoading;
    final sendState = controller.draft.pending?.state;
    final sendStatus = switch (sendState) {
      ChatDraftSendState.pending => l10n.chatSending,
      ChatDraftSendState.failed => l10n.chatSendFailed,
      ChatDraftSendState.uncertain => l10n.chatSendUncertain,
      null => null,
    };
    return PopScope(
      canPop: !controller.saveFailed,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) {
          unawaited(controller.flush());
        } else {
          widget.onClose();
        }
      },
      child: Scaffold(
        appBar: AppBar(
          toolbarHeight: MediaQuery.textScalerOf(context)
              .scale(kToolbarHeight)
              .clamp(kToolbarHeight, double.infinity),
          leading: BackButton(onPressed: widget.onClose),
          actions: [
            if (conversation != null)
              CraftskyContextMenuButton(
                tooltip: l10n.chatConversationActions,
                enabled: !controller.busy && !blockBusy && widget.canOperate,
                groups: [
                  CraftskyContextMenuGroup(
                    items: [
                      CraftskyContextMenuItem(
                        text: conversation.muted
                            ? l10n.chatUnmute
                            : l10n.chatMute,
                        icon: conversation.muted
                            ? CraftskyIcons.unmuted
                            : CraftskyIcons.muted,
                        onPressed: () => controller.setMuted(
                          muted: !conversation.muted,
                        ),
                      ),
                    ],
                  ),
                  CraftskyContextMenuGroup(
                    items: [
                      if (!blocking)
                        CraftskyContextMenuItem(
                          text: l10n.profileBlockAction,
                          icon: CraftskyIconsBold.block,
                          style: CraftskyContextMenuItemStyle.destructive,
                          onPressed: _blockAccount,
                        ),
                      if (widget.onReportConversation != null)
                        CraftskyContextMenuItem(
                          text: l10n.chatReportConversationTitle,
                          icon: CraftskyIcons.report,
                          style: CraftskyContextMenuItemStyle.destructive,
                          onPressed: _reportConversation,
                        ),
                      CraftskyContextMenuItem(
                        text: l10n.chatLeaveTitle,
                        icon: CraftskyIcons.logout,
                        style: CraftskyContextMenuItemStyle.destructive,
                        onPressed: _leave,
                      ),
                    ],
                  ),
                ],
              ),
          ],
          title: conversation == null
              ? controller.loading
                    ? const ChatProfileHeaderSkeleton()
                    : Text(
                        l10n.chatInboxTitle,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      )
              : ChatPeerProfile(
                  peer: conversation.peer,
                  builder: (context, avatar, identity, openProfile) =>
                      Semantics(
                        button: true,
                        hint: l10n.profileVisitAction,
                        child: InkWell(
                          onTap: () => _openPeerProfile(openProfile),
                          child: ConstrainedBox(
                            constraints: const BoxConstraints(minHeight: 48),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                avatar,
                                const SizedBox(width: 12),
                                Flexible(child: identity),
                              ],
                            ),
                          ),
                        ),
                      ),
                ),
        ),
        body: controller.loading && conversation == null
            ? const ChatConversationSkeleton()
            : conversation == null
            ? ChatFailurePanel(
                message: l10n.chatUnavailable,
                onRetry: widget.onReload,
              )
            : LayoutBuilder(
                builder: (context, constraints) => FlutterChatAdapter(
                  key: ValueKey(controller),
                  history: controller.history,
                  outgoing: controller.outgoingMessages,
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
                              if (controller.failed)
                                ChatMessageEffect(
                                  severity: MessageSeverity.error,
                                  message: l10n.chatUnavailable,
                                ),
                              if (controller.saveFailed)
                                ChatMessageEffect(
                                  severity: MessageSeverity.error,
                                  message: l10n.chatDraftSaveFailed,
                                  action: MessageAction(
                                    label: l10n.chatSaveDraft,
                                    onPressed: () =>
                                        unawaited(controller.flush()),
                                  ),
                                ),
                              if (conversation.bucket ==
                                  ChatBucket.request) ...[
                                LayoutBuilder(
                                  builder: (context, constraints) {
                                    final enabled =
                                        !controller.busy &&
                                        !blockBusy &&
                                        widget.canOperate;
                                    final accept = ChunkyButton(
                                      onPressed: enabled && !blocked
                                          ? () => unawaited(controller.accept())
                                          : null,
                                      child: ActionProgress(
                                        isLoading:
                                            controller.busy && !_ignoring,
                                        child: Text(l10n.chatAcceptRequest),
                                      ),
                                    );
                                    final ignore = ChunkyButton(
                                      variant: ChunkyButtonVariant.secondary,
                                      onPressed: enabled
                                          ? () => unawaited(_ignore())
                                          : null,
                                      child: ActionProgress(
                                        isLoading: _ignoring,
                                        child: Text(l10n.chatIgnoreRequest),
                                      ),
                                    );
                                    if (constraints.maxWidth < 300 ||
                                        MediaQuery.textScalerOf(context)
                                                .scale(1) >
                                            1.3) {
                                      return Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.stretch,
                                        children: [
                                          accept,
                                          const SizedBox(height: 12),
                                          ignore,
                                        ],
                                      );
                                    }
                                    return Row(
                                      children: [
                                        Expanded(child: ignore),
                                        const SizedBox(width: 12),
                                        Expanded(child: accept),
                                      ],
                                    );
                                  },
                                ),
                              ] else ...[
                                Row(
                                  children: [
                                    Expanded(
                                      child: BrandTextField(
                                        label: l10n.chatComposeLabel,
                                        showLabel: false,
                                        hintText: l10n.chatComposeLabel,
                                        controller: _text,
                                        onChanged: controller.edit,
                                        minLines: 1,
                                        maxLines: 5,
                                        keyboardType: TextInputType.multiline,
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
                                    const SizedBox(width: 8),
                                    ChunkyIconButton(
                                      tooltip: l10n.chatSend,
                                      variant: ChunkyButtonVariant.primary,
                                      onPressed:
                                          controller.canSend &&
                                              !blockBusy &&
                                              !blocked &&
                                              widget.canOperate
                                          ? () => unawaited(controller.send())
                                          : null,
                                      icon: CraftskyIcons.sendFilled,
                                    ),
                                  ],
                                ),
                                if (!conversation.canSend || blocked)
                                  Text(l10n.chatCannotSend),
                                if (controller.draft.pending case final attempt?
                                    when attempt.state !=
                                        ChatDraftSendState.pending) ...[
                                  const SizedBox(height: 8),
                                  ChatMessageEffect(
                                    severity:
                                        attempt.state ==
                                            ChatDraftSendState.failed
                                        ? MessageSeverity.error
                                        : MessageSeverity.warning,
                                    message:
                                        attempt.state ==
                                            ChatDraftSendState.pending
                                        ? null
                                        : sendStatus,
                                    child: Text(sendStatus!),
                                  ),
                                  if (attempt.state !=
                                      ChatDraftSendState.pending)
                                    Wrap(
                                      children: [
                                        TextButton(
                                          onPressed:
                                              controller.canSendAgain &&
                                                  !blockBusy &&
                                                  !blocked &&
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
