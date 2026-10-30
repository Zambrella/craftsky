import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/providers/chat_session_runtime_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Gives lightweight messaging state an owner across authenticated routes.
class ChatSessionScope extends ConsumerStatefulWidget {
  const new({required this.child, super.key});
  final Widget child;

  @override
  ConsumerState<ChatSessionScope> createState() => _ChatSessionScopeState();
}

class _ChatSessionScopeState extends ConsumerState<ChatSessionScope> {
  late final AppLifecycleListener _lifecycle;

  @override
  void initState() {
    super.initState();
    _lifecycle = AppLifecycleListener(onResume: _refresh);
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
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(chatSessionRuntimeProvider);
    return widget.child;
  }
}
