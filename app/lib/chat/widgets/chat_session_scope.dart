import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/providers/chat_session_runtime_provider.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Keeps account messaging synchronized across authenticated foreground routes.
class ChatSessionScope extends ConsumerStatefulWidget {
  const new({required this.child, super.key});
  final Widget child;

  @override
  ConsumerState<ChatSessionScope> createState() => _ChatSessionScopeState();
}

class _ChatSessionScopeState extends ConsumerState<ChatSessionScope> {
  late final AppLifecycleListener _lifecycle;
  late final ProviderSubscription<ChatSyncCoordinator?> _runtime;
  ChatSyncCoordinator? _coordinator;
  late bool _resumed;

  @override
  void initState() {
    super.initState();
    final state = WidgetsBinding.instance.lifecycleState;
    _resumed = state == null || state == AppLifecycleState.resumed;
    _runtime = ref.listenManual(
      chatSessionRuntimeProvider,
      (previous, next) {
        if (previous != next) previous?.setSessionForeground(active: false);
        _coordinator = next;
        next?.setSessionForeground(active: _resumed);
      },
      fireImmediately: true,
    );
    _lifecycle = AppLifecycleListener(
      onResume: _refresh,
      onStateChange: (state) {
        _resumed = state == AppLifecycleState.resumed;
        _coordinator?.setSessionForeground(active: _resumed);
      },
    );
  }

  void _refresh() {
    if (!mounted) return;
    final account = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    if (account == null) return;
    ref.invalidate(chatStatusProvider(account));
    if (ref.exists(chatUnreadCountsProvider(account))) {
      ref.invalidate(chatUnreadCountsProvider(account));
    }
  }

  @override
  void dispose() {
    _lifecycle.dispose();
    _coordinator?.setSessionForeground(active: false);
    _runtime.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
