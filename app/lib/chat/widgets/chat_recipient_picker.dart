import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/chat/widgets/chat_peer_profile.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

Future<void> showSendViaChat(BuildContext context, String url) async {
  final container = ProviderScope.containerOf(context, listen: false);
  final lease = container.read(sessionRegistryProvider).value?.activeLease;
  if (lease == null) {
    context.showInfo(AppLocalizations.of(context).chatUnavailable);
    return;
  }
  final id = await responsiveModalNavigator(context).push<ChatConversationId>(
    FullscreenModalRoute<ChatConversationId>(
      builder: (context) => ChatRecipientPicker(
        account: lease.session.account,
        prefillLink: url,
        onSelected: (id) => Navigator.of(context).pop(id),
      ),
    ),
  );
  if (context.mounted &&
      id != null &&
      (container.read(sessionRegistryProvider).value?.isCurrent(lease) ??
          false)) {
    ChatConversationRoute(conversationId: id.value).go(context);
  }
}

class ChatRecipientPicker extends ConsumerStatefulWidget {
  const new({
    required this.account,
    required this.onSelected,
    this.prefillLink,
    this.initialRecipient,
    this.onClose,
    super.key,
  });
  final AccountKey account;
  final String? prefillLink;
  final Did? initialRecipient;
  final VoidCallback? onClose;
  final ValueChanged<ChatConversationId> onSelected;
  @override
  ConsumerState<ChatRecipientPicker> createState() =>
      _ChatRecipientPickerState();
}

class _ChatRecipientPickerState extends ConsumerState<ChatRecipientPicker> {
  final _search = TextEditingController();
  Timer? _debounce;
  ActiveAccountLease? _lease;
  bool _busy = false;
  bool _attempted = false;
  String _query = '';
  int _selectionEpoch = 0;
  @override
  void initState() {
    super.initState();
    _lease = ref.read(sessionRegistryProvider).value?.activeLease;
    _openInitialRecipient();
  }

  void _openInitialRecipient() {
    final did = widget.initialRecipient;
    if (did == null) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && widget.initialRecipient == did) unawaited(_choose(did));
    });
  }

  @override
  void didUpdateWidget(covariant ChatRecipientPicker oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.account != widget.account ||
        oldWidget.initialRecipient != widget.initialRecipient ||
        oldWidget.prefillLink != widget.prefillLink) {
      _selectionEpoch++;
      _debounce?.cancel();
      _search.clear();
      _query = '';
      _busy = false;
      _attempted = false;
      _openInitialRecipient();
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    super.dispose();
  }

  Future<void> _choose(Did did) async {
    final lease = _lease;
    if (_busy || lease?.session.account != widget.account) return;
    final epoch = _selectionEpoch;
    final authority = ActiveAccountOperation.fromWidget(ref, lease);
    final operation = ActiveAccountOperation.withGuard(
      ownership: lease,
      isCurrent: () =>
          authority.isCurrent &&
          epoch == _selectionEpoch &&
          widget.account == lease!.session.account &&
          (ModalRoute.of(context)?.isCurrent ?? true),
    );
    if (!operation.isCurrent) return;
    FocusScope.of(context).unfocus();
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    setState(() {
      _busy = true;
      _attempted = true;
    });
    final recipientFamily = chatRecipientControllerProvider(widget.account);
    final recipientSubscription = ref.listenManual(recipientFamily, (_, _) {});
    try {
      final recipient = await operation.step(
        () => ref.read(recipientFamily.future),
      );
      await operation.step(() => recipient.selectRecipient(did));
      if (recipient.failed) return;
      if (recipient.availability?.canChat != true) {
        if (mounted) {
          context.showInfo(
            AppLocalizations.of(context).chatRecipientUnavailable,
          );
        }
        return;
      }
      final id =
          recipient.availability?.conversationId ??
          await operation.step(recipient.start);
      if (id == null) return;
      final link = widget.prefillLink;
      if (link == null) {
        operation.publish(() => widget.onSelected(id));
        return;
      }
      final family = chatConversationControllerProvider(widget.account, id);
      final subscription = ref.listenManual(family, (_, _) {});
      try {
        final conversation = await operation.step(
          () => ref.read(family.future),
        );
        final prepared = await operation.step(
          () => conversation.prefillLink(link),
        );
        if (!prepared) {
          if (mounted) {
            context.showError(AppLocalizations.of(context).chatDraftSaveFailed);
          }
          return;
        }
        operation.publish(() => widget.onSelected(id));
      } finally {
        subscription.close();
      }
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (mounted && operation.isCurrent) {
        unawaited(
          reporter.captureException(
            error,
            stackTrace: stack,
            context: ReportContext(
              feature: 'Chat',
              operation: widget.prefillLink == null ? 'start' : 'shareLink',
              classification: 'chat.failure',
              workflow: PrivateOperationalFailureContext(
                accountDid: widget.account.did.value,
              ),
            ),
          ),
        );
        context.showError(AppLocalizations.of(context).chatUnavailable);
      }
    } finally {
      recipientSubscription.close();
      if (mounted && epoch == _selectionEpoch) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final registry = ref.watch(sessionRegistryProvider).value;
    final lease = _lease;
    if (lease == null ||
        lease.session.account != widget.account ||
        registry?.isCurrent(lease) != true) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context);
    final status = ref.watch(chatStatusProvider(widget.account));
    Widget body;
    var searchReady = false;
    if (status.isLoading) {
      body = const ChatListSkeleton();
    } else if (status.value?.state != ChatConnectionState.ready) {
      body = Center(
        child: TextButton(
          onPressed: () {
            Navigator.of(context).pop();
            const ChatInboxRoute().go(context);
          },
          child: Text(l10n.chatShareEnable),
        ),
      );
    } else {
      final recipients = ref.watch(
        chatRecipientControllerProvider(widget.account),
      );
      final inbox = ref.watch(
        chatInboxControllerProvider(widget.account, ChatBucket.accepted),
      );
      searchReady = recipients.hasValue && !recipients.isLoading;
      body = recipients.when(
        loading: () => const ChatListSkeleton(),
        error: (_, _) => ChatFailurePanel(
          message: l10n.chatUnavailable,
          onRetry: () =>
              ref.invalidate(chatRecipientControllerProvider(widget.account)),
        ),
        data: (controller) => ListenableBuilder(
          listenable: controller,
          builder: (context, _) => ChatMessageEffect(
            severity: MessageSeverity.error,
            message: controller.failed && (_attempted || _query.isNotEmpty)
                ? l10n.chatUnavailable
                : null,
            child: _query.isEmpty
                ? inbox.when(
                    loading: () => const ChatListSkeleton(),
                    error: (_, _) => ChatFailurePanel(
                      message: l10n.chatUnavailable,
                      onRetry: () => ref.invalidate(
                        chatInboxControllerProvider(
                          widget.account,
                          ChatBucket.accepted,
                        ),
                      ),
                    ),
                    data: (inbox) => ListenableBuilder(
                      listenable: inbox,
                      builder: (context, _) => ChatMessageEffect(
                        severity: MessageSeverity.error,
                        message: inbox.failed ? l10n.chatUnavailable : null,
                        child: ListView(
                          children: [
                            Padding(
                              padding: const EdgeInsets.all(16),
                              child: Text(l10n.chatShareRecent),
                            ),
                            for (final conversation in inbox.items)
                              ChatPeerProfile(
                                peer: conversation.peer,
                                builder: (context, avatar, identity, _) =>
                                    ListTile(
                                      leading: avatar,
                                      title: identity,
                                      onTap: _busy
                                          ? null
                                          : () => unawaited(
                                              _choose(conversation.peer.did),
                                            ),
                                    ),
                              ),
                            if (inbox.items.isEmpty)
                              Padding(
                                padding: const EdgeInsets.all(16),
                                child: Text(l10n.chatShareSearchHint),
                              ),
                          ],
                        ),
                      ),
                    ),
                  )
                : controller.searching
                ? const ChatListSkeleton()
                : ListView(
                    children: [
                      for (final profile in controller.items)
                        ListTile(
                          leading: ProfileAvatar(
                            seed: profile.displayName ?? profile.handle.value,
                            avatarUrl: profile.avatar,
                            size: ProfileAvatarSize.small,
                            customisation: profile.customisation,
                          ),
                          title: CompactProfileIdentity(
                            handle: profile.handle.value,
                            displayName: profile.displayName,
                            crafts: profile.crafts,
                          ),
                          onTap: _busy
                              ? null
                              : () => unawaited(_choose(profile.did)),
                        ),
                      if (controller.items.isEmpty)
                        Padding(
                          padding: const EdgeInsets.all(16),
                          child: Text(l10n.chatShareNoResults),
                        ),
                    ],
                  ),
          ),
        ),
      );
    }
    return Scaffold(
      appBar: AppBar(
        leading: CloseButton(onPressed: widget.onClose),
        title: Text(
          widget.prefillLink == null
              ? l10n.chatNewMessage
              : l10n.chatSendViaChat,
        ),
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: BrandTextField(
                label: l10n.chatSearchCraftsky,
                controller: _search,
                prefixIcon: const Icon(CraftskyIcons.search),
                autocorrect: false,
                enableSuggestions: false,
                enabled: !_busy && searchReady,
                onChanged: (query) {
                  setState(() => _query = query.trim());
                  _debounce?.cancel();
                  final controller = ref
                      .read(chatRecipientControllerProvider(widget.account))
                      .value;
                  controller?.beginSearch();
                  _debounce = Timer(const Duration(milliseconds: 300), () {
                    if (mounted) unawaited(controller?.search(query));
                  });
                },
              ),
            ),
            Expanded(child: body),
          ],
        ),
      ),
    );
  }
}
