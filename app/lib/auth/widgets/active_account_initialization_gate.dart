import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart'
    as registry_model;
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_logging.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/auth/widgets/active_account_initialization_error_screen.dart';
import 'package:craftsky_app/initialization_loading_screen.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/providers/account_language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Mounts signed-in UI only after account-critical state is ready for the
/// registry's exact active lease.
class ActiveAccountInitializationGate extends ConsumerStatefulWidget {
  const ActiveAccountInitializationGate({
    required this.child,
    super.key,
  });

  final Widget child;

  @override
  ConsumerState<ActiveAccountInitializationGate> createState() =>
      _ActiveAccountInitializationGateState();
}

class _ActiveAccountInitializationGateState
    extends ConsumerState<ActiveAccountInitializationGate> {
  bool _recovering = false;

  @override
  Widget build(BuildContext context) {
    ref.listen(activeAccountInitializationProvider, (previous, next) {
      if (next case AsyncError()) {
        logActiveAccountInitializationFailure();
      }
    });

    final registry = ref.watch(sessionRegistryProvider);
    final initialization = ref.watch(activeAccountInitializationProvider);
    if (registry case AsyncData(:final value)) {
      final activeLease = value.activeLease;
      if (initialization case AsyncData(value: final initialized)) {
        final isCurrent =
            (activeLease == null && initialized == null) ||
            (activeLease != null && initialized?.lease == activeLease);
        if (isCurrent) return widget.child;
      }
      if (initialization case AsyncError()) {
        return _errorScreen(registry: value);
      }
    } else if (registry case AsyncError()) {
      return _errorScreen(registry: null);
    }

    return const InitializationLoadingScreen();
  }

  Widget _errorScreen({
    required registry_model.SessionRegistry? registry,
  }) {
    final hasAlternative =
        registry != null && registry.orderedSessions.length > 1;
    return ActiveAccountInitializationErrorScreen(
      busy: _recovering || ref.watch(authControllerProvider).isLoading,
      onRetry: _retry,
      onSwitchAccount: hasAlternative
          ? () => unawaited(_showAccountSwitcher(registry))
          : null,
      onSignOut: () => unawaited(_signOut()),
    );
  }

  void _retry() {
    final registry = ref.read(sessionRegistryProvider);
    if (registry case AsyncData(:final value)) {
      final lease = value.activeLease;
      if (lease != null) {
        ref
          ..invalidate(
            languagePreferencesRepositoryProvider(lease.session.account),
          )
          ..invalidate(accountLanguagePreferencesProvider(lease));
      }
    } else {
      ref.invalidate(sessionRegistryProvider);
    }
    ref.invalidate(activeAccountInitializationProvider);
  }

  Future<void> _showAccountSwitcher(
    registry_model.SessionRegistry registry,
  ) async {
    final state = AccountSwitcherState.fromRegistry(registry);
    AccountSessionLease? activating;
    final activation = AccountActivationCoordinator(
      readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
      commitActivation: ref.read(sessionRegistryProvider.notifier).activate,
      invalidateAccountState: ref.read(accountStateInvalidatorProvider),
      resetToHome: ref.read(accountHomeResetProvider),
      confirmLeave: ref.read(unsavedWorkGuardProvider).confirmLeave,
    );

    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (sheetContext) => StatefulBuilder(
        builder: (context, setSheetState) => AccountSwitcherContent(
          state: state,
          activating: activating,
          showAddAccount: false,
          onAddAccount: () {},
          onSelect: (lease) {
            if (activating != null) return;
            setSheetState(() => activating = lease);
            unawaited(
              activation
                  .activate(lease)
                  .then((result) {
                    if (!sheetContext.mounted) return;
                    if (result == AccountActivationResult.activated ||
                        result == AccountActivationResult.alreadyActive) {
                      Navigator.pop(sheetContext);
                      return;
                    }
                    setSheetState(() => activating = null);
                  })
                  .catchError((Object _, StackTrace _) {
                    if (!sheetContext.mounted) return;
                    setSheetState(() => activating = null);
                    _showRecoveryFailure();
                  }),
            );
          },
        ),
      ),
    );
  }

  Future<void> _signOut() async {
    if (_recovering) return;
    setState(() => _recovering = true);
    try {
      final result = await ref.read(authControllerProvider.notifier).signOut();
      if (result == null && mounted) _showRecoveryFailure();
    } on Object {
      if (mounted) _showRecoveryFailure();
    } finally {
      if (mounted) setState(() => _recovering = false);
    }
  }

  void _showRecoveryFailure() {
    ScaffoldMessenger.maybeOf(context)?.showSnackBar(
      SnackBar(
        content: Text(
          AppLocalizations.of(context).activeAccountRecoveryFailed,
        ),
      ),
    );
  }
}
