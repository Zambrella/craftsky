import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<void> showAccountSwitcherSheet({
  required BuildContext context,
  required AccountSwitcherState fallbackState,
  required Future<AccountActivationResult> Function(AccountSessionLease)
  onSelect,
  required VoidCallback onAddAccount,
}) => showModalBottomSheet<void>(
  context: context,
  showDragHandle: true,
  builder: (sheetContext) => LiveAccountSwitcherContent(
    fallbackState: fallbackState,
    onSelect: onSelect,
    onAddAccount: onAddAccount,
  ),
);

class LiveAccountSwitcherContent extends ConsumerStatefulWidget {
  const LiveAccountSwitcherContent({
    required this.fallbackState,
    required this.onSelect,
    required this.onAddAccount,
    super.key,
  });

  final AccountSwitcherState fallbackState;
  final Future<AccountActivationResult> Function(AccountSessionLease) onSelect;
  final VoidCallback onAddAccount;

  @override
  ConsumerState<LiveAccountSwitcherContent> createState() =>
      _LiveAccountSwitcherContentState();
}

class _LiveAccountSwitcherContentState
    extends ConsumerState<LiveAccountSwitcherContent> {
  AccountSessionLease? _activating;

  @override
  Widget build(BuildContext context) {
    final registry = ref.watch(sessionRegistryProvider).value;
    final state = registry == null
        ? widget.fallbackState
        : AccountSwitcherState.fromRegistry(registry);
    return AccountSwitcherContent(
      state: state,
      activating: _activating,
      onSelect: (lease) => unawaited(_activate(lease)),
      onAddAccount: widget.onAddAccount,
    );
  }

  Future<void> _activate(AccountSessionLease lease) async {
    if (_activating != null) return;
    setState(() => _activating = lease);
    try {
      final result = await widget.onSelect(lease);
      if (!mounted) return;
      if (result == AccountActivationResult.activated ||
          result == AccountActivationResult.alreadyActive) {
        await Navigator.maybePop(context);
      }
    } finally {
      if (mounted) setState(() => _activating = null);
    }
  }
}
