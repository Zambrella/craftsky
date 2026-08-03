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
}

final class _RecordingScreenAwake implements SubmissionScreenAwake {
  final events = <String>[];

  @override
  Future<void> enable() async => events.add('enable');

  @override
  Future<void> disable() async => events.add('disable');
}
