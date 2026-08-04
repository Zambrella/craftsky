import 'dart:io';

import 'package:wakelock_plus/wakelock_plus.dart';

abstract interface class SubmissionScreenAwake {
  Future<void> enable();

  Future<void> disable();
}

final class WakelockSubmissionScreenAwake implements SubmissionScreenAwake {
  const WakelockSubmissionScreenAwake();

  @override
  Future<void> enable() => Platform.isAndroid || Platform.isIOS
      ? WakelockPlus.enable()
      : Future<void>.value();

  @override
  Future<void> disable() => Platform.isAndroid || Platform.isIOS
      ? WakelockPlus.disable()
      : Future<void>.value();
}

final class ComposerSubmissionLifecycle {
  ComposerSubmissionLifecycle(this._screenAwake);

  final SubmissionScreenAwake _screenAwake;
  bool _running = false;
  bool _enabled = false;
  bool _disposed = false;

  bool get isRunning => _running;

  Future<T> run<T>(Future<T> Function() operation) async {
    if (_disposed) throw StateError('submission lifecycle is disposed');
    if (_running) throw StateError('submission already running');
    _running = true;
    try {
      await _screenAwake.enable();
      _enabled = true;
      if (_disposed) {
        await _release();
        throw StateError('submission lifecycle is disposed');
      }
      return await operation();
    } finally {
      try {
        await _release();
      } finally {
        _running = false;
      }
    }
  }

  Future<void> dispose() async {
    _disposed = true;
    await _release();
  }

  Future<void> _release() async {
    if (!_enabled) return;
    try {
      await _screenAwake.disable();
      _enabled = false;
    } on Object {
      // Best effort: retain ownership so a later run/dispose can retry release.
    }
  }
}
