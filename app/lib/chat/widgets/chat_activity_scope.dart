import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Registers the visible route with one account's disposable heartbeat loop.
class ChatActivityScope extends ConsumerWidget {
  const new({
    required this.account,
    required this.view,
    required this.child,
    this.conversationId,
    super.key,
  });
  final AccountKey account;
  final ChatView view;
  final ChatConversationId? conversationId;
  final Widget child;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (view == ChatView.none) return child;
    final state = ref.watch(chatSyncCoordinatorProvider(account));
    final coordinator = state.isLoading || state.hasError ? null : state.value;
    return _AttachedChatActivity(
      key: ValueKey(account),
      coordinator: coordinator,
      view: view,
      conversationId: conversationId,
      child: child,
    );
  }
}

class _AttachedChatActivity extends StatefulWidget {
  const new({
    required this.coordinator,
    required this.view,
    required this.conversationId,
    required this.child,
    super.key,
  });
  final ChatSyncCoordinator? coordinator;
  final ChatView view;
  final ChatConversationId? conversationId;
  final Widget child;
  @override
  State<_AttachedChatActivity> createState() => _AttachedChatActivityState();
}

class _AttachedChatActivityState extends State<_AttachedChatActivity> {
  late final AppLifecycleListener _lifecycle;
  late bool _resumed;
  bool _routeCurrent = true;
  final _registration = Object();
  @override
  void initState() {
    super.initState();
    widget.coordinator?.addListener(_changed);
    final state = WidgetsBinding.instance.lifecycleState;
    _resumed = state == null || state == AppLifecycleState.resumed;
    _lifecycle = AppLifecycleListener(
      onStateChange: (state) {
        _resumed = state == AppLifecycleState.resumed;
        _configure();
      },
    );
  }

  void _changed() {
    if (mounted) setState(() {});
  }

  void _configure() {
    widget.coordinator?.registerView(
      _registration,
      widget.view,
      conversationId: widget.conversationId,
      active: _resumed && _routeCurrent,
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _routeCurrent = ModalRoute.of(context)?.isCurrent ?? true;
    _configure();
  }

  @override
  void didUpdateWidget(covariant _AttachedChatActivity oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.coordinator != widget.coordinator) {
      oldWidget.coordinator?.unregisterView(_registration);
      oldWidget.coordinator?.removeListener(_changed);
      widget.coordinator?.addListener(_changed);
    }
    _configure();
  }

  @override
  void dispose() {
    _lifecycle.dispose();
    widget.coordinator?.removeListener(_changed);
    widget.coordinator?.unregisterView(_registration);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final coordinator = widget.coordinator;
    final stale = coordinator?.showSyncWarning == true;
    final l10n = AppLocalizations.of(context);
    return ChatMessageEffect(
      severity: MessageSeverity.warning,
      message: stale ? l10n.chatSyncStale : null,
      action: MessageAction(
        label: l10n.chatRefresh,
        onPressed: () => coordinator?.retry(),
      ),
      child: widget.child,
    );
  }
}
