import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final unsavedWorkGuardProvider = Provider<UnsavedWorkGuard>(
  (ref) => UnsavedWorkGuard(),
);

final class UnsavedWorkRegistration {
  const UnsavedWorkRegistration._(this.id);

  final int id;

  @override
  String toString() => 'UnsavedWorkRegistration(<redacted>)';
}

final class UnsavedWorkGuard {
  final _entries = <int, _UnsavedWorkEntry>{};
  int _nextId = 0;
  Future<bool>? _pendingConfirmation;

  UnsavedWorkRegistration register({
    required AccountSessionLease owner,
    required bool Function() isDirty,
    required Future<bool> Function() confirmAndClose,
  }) {
    final registration = UnsavedWorkRegistration._(++_nextId);
    _entries[registration.id] = _UnsavedWorkEntry(
      owner: owner,
      isDirty: isDirty,
      confirmAndClose: confirmAndClose,
    );
    return registration;
  }

  UnsavedWorkRegistration replace(
    UnsavedWorkRegistration? previous, {
    required AccountSessionLease owner,
    required bool Function() isDirty,
    required Future<bool> Function() confirmAndClose,
  }) {
    unregister(previous);
    return register(
      owner: owner,
      isDirty: isDirty,
      confirmAndClose: confirmAndClose,
    );
  }

  void unregister(UnsavedWorkRegistration? registration) {
    if (registration == null) return;
    _entries.remove(registration.id);
  }

  Future<bool> confirmLeave(AccountSessionLease owner) {
    final pending = _pendingConfirmation;
    if (pending != null) return pending;
    _UnsavedWorkEntry? dirty;
    for (final entry in _entries.values.toList().reversed) {
      if (entry.owner == owner && entry.isDirty()) {
        dirty = entry;
        break;
      }
    }
    if (dirty == null) return Future.value(true);

    final operation = dirty.confirmAndClose();
    _pendingConfirmation = operation;
    unawaited(
      operation.whenComplete(() {
        if (identical(_pendingConfirmation, operation)) {
          _pendingConfirmation = null;
        }
      }),
    );
    return operation;
  }
}

final class _UnsavedWorkEntry {
  const _UnsavedWorkEntry({
    required this.owner,
    required this.isDirty,
    required this.confirmAndClose,
  });

  final AccountSessionLease owner;
  final bool Function() isDirty;
  final Future<bool> Function() confirmAndClose;
}
