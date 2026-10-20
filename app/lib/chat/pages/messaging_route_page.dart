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
import 'package:craftsky_app/chat/pages/new_message_page.dart';
import 'package:craftsky_app/chat/pages/report_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
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
  bool _authorizationFailed = false;
  Future<void> _authorize(AccountKey account, bool push) async {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (_authorizing || lease?.session.account != account) return;
    final operation = ActiveAccountOperation.fromWidget(ref, lease);
    setState(() {
      _authorizing = true;
      _authorizationFailed = false;
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
        setState(() => _authorizationFailed = true);
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
        case MessagingDestination.inbox || MessagingDestination.requests:
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
    ChatMessageId message,
  ) async {
    await showDialog<void>(
      context: context,
      builder: (context) => Dialog.fullscreen(
        child: _MessagingReportPage(account: account, id: id, message: message),
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
          onClose: _close,
          initialBucket: widget.destination == MessagingDestination.requests
              ? ChatBucket.request
              : ChatBucket.accepted,
          onOpen: _open,
          onNewMessage: () => const NewMessageRoute().go(context),
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
          onAuthorize: (push) => authorize(push: push),
          onClose: _close,
          builder: (context, ref, status) {
            final family = chatRecipientControllerProvider(account);
            return ref
                .watch(family)
                .when(
                  skipLoadingOnRefresh: false,
                  loading: () => const Scaffold(
                    body: Center(child: CircularProgressIndicator()),
                  ),
                  error: (_, _) => failure(() => ref.invalidate(family)),
                  data: (controller) => NewMessageView(
                    key: ValueKey(lease),
                    controller: controller,
                    initialRecipient: initialRecipient,
                    onClose: _close,
                    onOpen: _open,
                  ),
                );
          },
        );
      case MessagingDestination.settings:
        final family = chatSettingsControllerProvider(account);
        page = ref
            .watch(family)
            .when(
              skipLoadingOnRefresh: false,
              loading: () => const Scaffold(
                body: Center(child: CircularProgressIndicator()),
              ),
              error: (_, _) => failure(() => ref.invalidate(family)),
              data: (controller) => MessagingSettingsView(
                key: ValueKey(lease),
                controller: controller,
                onAuthorize: (push) => authorize(push: push),
                onClose: _close,
              ),
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
            const Positioned(
              top: 0,
              left: 0,
              right: 0,
              child: LinearProgressIndicator(),
            ),
          if (_authorizationFailed)
            Positioned(
              bottom: 0,
              left: 0,
              right: 0,
              child: Material(
                child: SafeArea(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Text(AppLocalizations.of(context).chatUnavailable),
                  ),
                ),
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
  final ChatMessageId message;
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
    return ref
        .watch(family)
        .when(
          skipLoadingOnRefresh: false,
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (_, _) => ChatFailurePanel(
            message: AppLocalizations.of(context).chatUnavailable,
            onRetry: () => ref.invalidate(family),
          ),
          data: (controller) => ReportPage(
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
