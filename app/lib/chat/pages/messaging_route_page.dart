import 'dart:async';

import 'package:craftsky_app/auth/data/oauth_handoff_mode.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/chat_conversation_page.dart';
import 'package:craftsky_app/chat/pages/chat_settings_page.dart';
import 'package:craftsky_app/chat/pages/messaging_inbox_page.dart';
import 'package:craftsky_app/chat/pages/report_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/chat/widgets/chat_recipient_picker.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

enum MessagingDestination {
  inbox,
  requests,
  newMessage,
  conversation,
  settings,
}

/// Cold routes reconstruct private state from the current activation and ID.
class MessagingRoutePage extends ConsumerStatefulWidget {
  const new({
    required this.destination,
    this.conversationId,
    this.recipientDid,
    super.key,
  });
  final MessagingDestination destination;
  final String? conversationId;
  final String? recipientDid;
  @override
  ConsumerState<MessagingRoutePage> createState() => _MessagingRoutePageState();
}

class _MessagingRoutePageState extends ConsumerState<MessagingRoutePage> {
  bool _authorizing = false;
  Future<void> _authorize(AccountKey account, bool push) async {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (_authorizing || lease?.session.account != account) return;
    final operation = ActiveAccountOperation.fromWidget(ref, lease);
    setState(() {
      _authorizing = true;
    });
    try {
      final repository = await operation.step(
        () => ref.read(accountChatRepositoryProvider(account).future),
      );
      final uri = await operation.step(
        () => repository.authorize(
          handoffMode: oauthHandoffModeForCurrentBuild(),
          pushEnabled: push,
        ),
      );
      final launched = await operation.step(
        () => ref.read(authUrlLauncherProvider)(uri),
      );
      if (!launched) {
        throw const FormatException('Messaging browser did not open');
      }
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (operation.isCurrent && mounted) {
        unawaited(
          GuardedErrorReporter(ref.read(errorReporterProvider))
              .captureException(
                error,
                stackTrace: stack,
                context: ReportContext(
                  feature: 'Chat',
                  operation: 'authorize',
                  classification: 'chat.failure',
                  workflow: PrivateOperationalFailureContext(
                    accountDid: account.did.value,
                  ),
                ),
              ),
        );
        context.showError(AppLocalizations.of(context).chatUnavailable);
      }
    } finally {
      if (mounted) setState(() => _authorizing = false);
    }
  }

  void _open(ChatConversationId id) =>
      ChatConversationRoute(conversationId: id.value).go(context);
  void _close() {
    if (context.canPop()) {
      context.pop();
    } else {
      switch (widget.destination) {
        case MessagingDestination.inbox:
          const FeedRoute().go(context);
        case MessagingDestination.settings:
          const SettingsRoute().go(context);
        case _:
          const ChatInboxRoute().go(context);
      }
    }
  }

  Future<void> _report(
    AccountKey account,
    ChatConversationId id,
    ChatMessageId? message,
  ) async {
    if (!mounted ||
        ref.read(sessionRegistryProvider).value?.activeLease?.session.account !=
            account) {
      return;
    }
    await responsiveModalNavigator(context).push<void>(
      FullscreenModalRoute<void>(
        builder: (_) => _MessagingReportPage(
          account: account,
          id: id,
          message: message,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final lease = ref.watch(
      sessionRegistryProvider.select((value) => value.value?.activeLease),
    );
    if (lease == null) return const SizedBox.shrink();
    final account = lease.session.account;
    void authorize({required bool push}) =>
        unawaited(_authorize(account, push));
    Widget failure(VoidCallback retry) => Scaffold(
      body: ChatFailurePanel(
        message: AppLocalizations.of(context).chatUnavailable,
        onRetry: retry,
      ),
    );
    final Widget page;
    switch (widget.destination) {
      case MessagingDestination.inbox:
      case MessagingDestination.requests:
        page = MessagingInboxPage(
          key: ValueKey(lease),
          account: account,
          onClose: widget.destination == MessagingDestination.inbox
              ? null
              : _close,
          bucket: widget.destination == MessagingDestination.requests
              ? ChatBucket.request
              : ChatBucket.accepted,
          onOpen: _open,
          onNewMessage: () => const NewMessageRoute().push<void>(context),
          onRequests: () => const ChatRequestsRoute().push<void>(context),
          onSettings: () => const ChatSettingsRoute().push<void>(context),
          onAuthorize: (push) => authorize(push: push),
        );
      case MessagingDestination.conversation:
        ChatConversationId id;
        try {
          id = ChatConversationId.parse(widget.conversationId ?? '');
        } on FormatException {
          return failure(() => const ChatInboxRoute().go(context));
        }
        page = MessagingConversationPage(
          key: ValueKey(lease),
          account: account,
          conversationId: id,
          onClose: _close,
          onReport: (message) => unawaited(_report(account, id, message)),
          onReportConversation: () => unawaited(_report(account, id, null)),
          onAuthorize: (push) => authorize(push: push),
        );
      case MessagingDestination.newMessage:
        Did? initialRecipient;
        try {
          if (widget.recipientDid != null) {
            initialRecipient = Did.parse(widget.recipientDid!);
          }
        } on Object {
          return failure(() => const NewMessageRoute().go(context));
        }
        page = ChatContentGate(
          account: account,
          loadingBody: const ChatListSkeleton(),
          onAuthorize: (push) => authorize(push: push),
          onClose: _close,
          builder: (context, ref, status) => ChatRecipientPicker(
            key: ValueKey(lease),
            account: account,
            initialRecipient: initialRecipient,
            onClose: _close,
            onSelected: _open,
          ),
        );
      case MessagingDestination.settings:
        page = MessagingSettingsPage(
          key: ValueKey(lease),
          account: account,
          onAuthorize: (push) => authorize(push: push),
          onClose: _close,
        );
    }
    final view = switch (widget.destination) {
      MessagingDestination.inbox ||
      MessagingDestination.requests => ChatView.inbox,
      MessagingDestination.conversation => ChatView.conversation,
      _ => ChatView.none,
    };
    final activityId = view == ChatView.conversation
        ? ChatConversationId.parse(widget.conversationId!)
        : null;
    return ChatActivityScope(
      account: account,
      view: view,
      conversationId: activityId,
      child: Stack(
        children: [
          page,
          if (_authorizing)
            const Positioned.fill(
              child: IgnorePointer(
                child: Center(child: StitchProgressIndicator()),
              ),
            ),
        ],
      ),
    );
  }
}

class _MessagingReportPage extends ConsumerWidget {
  const new({required this.account, required this.id, required this.message});
  final AccountKey account;
  final ChatConversationId id;
  final ChatMessageId? message;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (ref.watch(
          sessionRegistryProvider.select(
            (value) => value.value?.activeLease?.session.account,
          ),
        ) !=
        account) {
      return const SizedBox.shrink();
    }
    final family = chatReportControllerProvider(account, id, message);
    final title = message == null
        ? AppLocalizations.of(context).chatReportConversationTitle
        : AppLocalizations.of(context).chatReportTitle;
    return ref
        .watch(family)
        .when(
          skipLoadingOnRefresh: false,
          loading: () => ChatLoadingPage(
            title: title,
            onClose: () => Navigator.of(context).pop(),
          ),
          error: (_, _) => Scaffold(
            appBar: AppBar(
              leading: CloseButton(
                onPressed: () => Navigator.of(context).pop(),
              ),
              title: Text(title),
            ),
            body: ChatFailurePanel(
              message: AppLocalizations.of(context).chatUnavailable,
              onRetry: () => ref.invalidate(family),
            ),
          ),
          data: (controller) => ReportPage(
            isConversation: message == null,
            onClose: () => Navigator.of(context).pop(),
            onCheckStatus: () async {
              await controller.checkReceipt();
              return switch (controller.state) {
                ChatOperationState.confirmed => ReportSubmissionState.confirmed,
                ChatOperationState.failed => ReportSubmissionState.failed,
                _ => ReportSubmissionState.uncertain,
              };
            },
            onSubmit: (type, reason) async {
              await controller.submit(type, reason);
              return switch (controller.state) {
                ChatOperationState.confirmed => ReportSubmissionState.confirmed,
                ChatOperationState.failed => ReportSubmissionState.failed,
                _ => ReportSubmissionState.uncertain,
              };
            },
          ),
        );
  }
}
