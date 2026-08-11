import 'dart:async';

import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart'
    show deletionStatusRegistryProvider;
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Keeps retained deletion rows current even when their status page is not
/// open. Polling is status-capability-only, backs off to a bounded interval,
/// and stops when every observed job is terminal or needs attention.
class AccountDeletionStatusRefreshHost extends ConsumerStatefulWidget {
  const AccountDeletionStatusRefreshHost({
    required this.child,
    this.refreshImmediately = true,
    this.pollBackoff = const [
      Duration(seconds: 2),
      Duration(seconds: 5),
      Duration(seconds: 15),
      Duration(seconds: 30),
    ],
    super.key,
  }) : assert(
         pollBackoff.length > 0,
         'pollBackoff must contain at least one duration.',
       );

  final Widget child;
  final bool refreshImmediately;
  final List<Duration> pollBackoff;

  @override
  ConsumerState<AccountDeletionStatusRefreshHost> createState() =>
      _AccountDeletionStatusRefreshHostState();
}

class _AccountDeletionStatusRefreshHostState
    extends ConsumerState<AccountDeletionStatusRefreshHost>
    with WidgetsBindingObserver {
  Timer? _timer;
  Set<String> _observedJobIds = const {};
  int _backoffIndex = 0;
  bool _refreshing = false;
  bool _refreshAgain = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed || _observedJobIds.isEmpty) return;
    _timer?.cancel();
    _backoffIndex = 0;
    unawaited(_refreshObserved());
  }

  @override
  Widget build(BuildContext context) {
    final registry = ref.watch(deletionStatusRegistryProvider).value;
    final observed = registry == null
        ? const <String>{}
        : {
            for (final entry in registry.entries)
              if (_pollable(entry)) entry.jobId,
          };
    if (!setEquals(observed, _observedJobIds)) {
      _observedJobIds = Set.unmodifiable(observed);
      _backoffIndex = 0;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted || !setEquals(observed, _observedJobIds)) return;
        _timer?.cancel();
        if (_observedJobIds.isEmpty) return;
        if (widget.refreshImmediately) {
          unawaited(_refreshObserved());
        } else {
          _scheduleNext();
        }
      });
    }
    return widget.child;
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _timer?.cancel();
    super.dispose();
  }

  bool _pollable(DeletionStatusEntry entry) =>
      entry.status == AccountDeletionStatus.active ||
      entry.status == AccountDeletionStatus.retrying;

  Future<void> _refreshObserved() async {
    if (!mounted || _observedJobIds.isEmpty) return;
    if (_refreshing) {
      _refreshAgain = true;
      return;
    }
    _refreshing = true;
    _timer?.cancel();
    final jobs = List<String>.of(_observedJobIds);
    try {
      await Future.wait([
        for (final jobId in jobs)
          ref
              .read(accountDeletionControllerProvider.notifier)
              .refresh(jobId)
              .catchError((Object _) => null),
      ]);
    } finally {
      _refreshing = false;
      if (mounted) {
        _synchronizeObservedAfterRefresh();
        if (_refreshAgain) {
          _refreshAgain = false;
          unawaited(_refreshObserved());
        } else {
          _scheduleNext();
        }
      }
    }
  }

  void _synchronizeObservedAfterRefresh() {
    final registry = ref.read(deletionStatusRegistryProvider).value;
    if (registry == null) return;
    final current = {
      for (final entry in registry.entries)
        if (_pollable(entry)) entry.jobId,
    };
    if (setEquals(current, _observedJobIds)) return;
    _timer?.cancel();
    _observedJobIds = Set.unmodifiable(current);
    _backoffIndex = 0;
  }

  void _scheduleNext() {
    _timer?.cancel();
    if (_observedJobIds.isEmpty) return;
    final index = _backoffIndex.clamp(0, widget.pollBackoff.length - 1);
    final delay = widget.pollBackoff[index];
    if (_backoffIndex < widget.pollBackoff.length - 1) _backoffIndex++;
    _timer = Timer(delay, () => unawaited(_refreshObserved()));
  }
}
