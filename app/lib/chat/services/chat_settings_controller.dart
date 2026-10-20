import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';

/// Explicit local preferences and shared declaration actions for one owner.
final class ChatSettingsController extends ChangeNotifier {
  new({
    required this.account,
    required this._repository,
    required this._operation,
    required this.canSetPolicy,
    this._onChanged,
    this._onConnectionChanged,
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _reporter = GuardedErrorReporter(reporter);
  final AccountKey account;
  final ChatRepository _repository;
  final ActiveAccountOperation _operation;
  final bool canSetPolicy;
  final Future<void> Function()? _onChanged;
  final void Function(ChatStatus)? _onConnectionChanged;
  final ErrorReporter _reporter;
  bool? pushEnabled;
  ChatPolicySetting? policy;
  bool loading = true;
  bool busy = false;
  bool failed = false;
  bool disconnected = false;
  bool _disposed = false;
  bool get _current => !_disposed && _operation.isCurrent;
  void _publish(void Function() action) {
    if (!_current) return;
    _operation.publish(() {
      action();
      notifyListeners();
    });
  }

  void _report(Object error, StackTrace stack) {
    unawaited(
      _reporter.captureException(
        error,
        stackTrace: stack,
        context: ReportContext(
          feature: 'Chat',
          operation: 'settings',
          classification: 'chat.failure',
          workflow: PrivateOperationalFailureContext(
            accountDid: account.did.value,
          ),
        ),
      ),
    );
  }

  Future<void> load() async {
    if (!_current || busy) return;
    _publish(() {
      loading = true;
      busy = true;
    });
    try {
      final push = await _operation.step(_repository.pushPreference);
      final incoming = canSetPolicy
          ? await _operation.step(_repository.incomingPolicy)
          : null;
      _publish(() {
        pushEnabled = push;
        policy = incoming;
        failed = false;
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) {
        _report(error, stack);
        _publish(() => failed = true);
      }
    } finally {
      _publish(() {
        loading = false;
        busy = false;
      });
    }
  }

  Future<void> _effect(Future<void> Function() effect) async {
    if (!_current || busy || loading || disconnected) return;
    _publish(() => busy = true);
    try {
      await _operation.step(effect);
      _publish(() => failed = false);
      if (_onChanged case final callback?) await _operation.step(callback);
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) {
        _report(error, stack);
        _publish(() => failed = true);
      }
    } finally {
      _publish(() => busy = false);
    }
  }

  Future<void> setPush({required bool enabled}) => _effect(() async {
    final saved = await _operation.step(
      () => _repository.setPushPreference(enabled: enabled),
    );
    _publish(() => pushEnabled = saved);
  });
  Future<void> setPolicy(ChatIncomingPolicy value) async {
    if (!canSetPolicy) return;
    await _effect(() async {
      final saved = await _operation.step(
        () => _repository.setIncomingPolicy(value),
      );
      _publish(() => policy = saved);
    });
  }

  Future<void> disconnect() => _effect(() async {
    final status = await _operation.step(
      () => _repository.setConnection(enabled: false),
    );
    if (status.enabled) {
      throw const FormatException('Messaging disconnect not confirmed');
    }
    _publish(() => disconnected = true);
    _operation.publish(() => _onConnectionChanged?.call(status));
  });
  @override
  void dispose() {
    _disposed = true;
    policy = null;
    pushEnabled = null;
    super.dispose();
  }
}
