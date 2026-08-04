import 'dart:async';

import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('releases screen-awake ownership after success and failure', () async {
    final awake = _RecordingScreenAwake();
    final lifecycle = ComposerSubmissionLifecycle(awake);

    await lifecycle.run(() async {});
    await expectLater(
      lifecycle.run(() async => throw StateError('safe fake failure')),
      throwsStateError,
    );

    expect(awake.events, ['enable', 'disable', 'enable', 'disable']);
  });

  test('releases once when the owning composer is disposed mid-run', () async {
    final awake = _RecordingScreenAwake();
    final lifecycle = ComposerSubmissionLifecycle(awake);
    final operationGate = Completer<void>();

    final run = lifecycle.run(() => operationGate.future);
    await Future<void>.delayed(Duration.zero);

    await lifecycle.dispose();
    expect(awake.events, ['enable', 'disable']);

    operationGate.complete();
    await run;
    expect(awake.events, ['enable', 'disable']);
  });

  test(
    'disable failure preserves the terminal result and allows retry',
    () async {
      final awake = _FailFirstDisableScreenAwake();
      final lifecycle = ComposerSubmissionLifecycle(awake);

      expect(await lifecycle.run(() async => 'published'), 'published');
      expect(lifecycle.isRunning, isFalse);

      expect(await lifecycle.run(() async => 'retried'), 'retried');
      expect(lifecycle.isRunning, isFalse);
      expect(awake.events, ['enable', 'disable', 'enable', 'disable']);
    },
  );

  test(
    'disable failure preserves the operation error and disposal retries',
    () async {
      final awake = _FailFirstDisableScreenAwake();
      final lifecycle = ComposerSubmissionLifecycle(awake);

      await expectLater(
        lifecycle.run<void>(
          () async => throw StateError('safe fake operation failure'),
        ),
        throwsA(
          isA<StateError>().having(
            (error) => error.message,
            'message',
            'safe fake operation failure',
          ),
        ),
      );
      expect(lifecycle.isRunning, isFalse);
      expect(awake.events, ['enable', 'disable']);

      await lifecycle.dispose();
      expect(awake.events, ['enable', 'disable', 'disable']);
    },
  );
}

final class _RecordingScreenAwake implements SubmissionScreenAwake {
  final events = <String>[];

  @override
  Future<void> enable() async => events.add('enable');

  @override
  Future<void> disable() async => events.add('disable');
}

final class _FailFirstDisableScreenAwake implements SubmissionScreenAwake {
  final events = <String>[];
  var _disableCalls = 0;

  @override
  Future<void> enable() async => events.add('enable');

  @override
  Future<void> disable() async {
    events.add('disable');
    _disableCalls += 1;
    if (_disableCalls == 1) throw StateError('safe fake disable failure');
  }
}
