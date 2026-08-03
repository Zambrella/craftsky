import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'orders presentation, snapshot, operation, cleanup, and release',
    () async {
      final events = <String>[];
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake(events),
      );

      await coordinator.run(
        presentOverlay: () async => events.add('present'),
        saveOriginSnapshot: () async => events.add('snapshot'),
        operation: () async => events.add('operation'),
        didSucceed: () => true,
        deleteOriginAfterSuccess: () async => events.add('delete'),
        onRunningChanged: ({required running}) =>
            events.add('running:$running'),
        onFailure: (_) => events.add('failure'),
      );

      expect(events, [
        'running:true',
        'present',
        'enable',
        'snapshot',
        'operation',
        'delete',
        'disable',
        'running:false',
      ]);
    },
  );

  test(
    'failure skips cleanup, releases ownership, and stays retryable',
    () async {
      final events = <String>[];
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake(events),
      );

      await coordinator.run(
        presentOverlay: () async => events.add('present'),
        saveOriginSnapshot: () async => events.add('snapshot'),
        operation: () async => throw StateError('safe fake failure'),
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async => events.add('delete'),
        onRunningChanged: ({required running}) =>
            events.add('running:$running'),
        onFailure: (_) => events.add('failure'),
      );

      expect(events, [
        'running:true',
        'present',
        'enable',
        'snapshot',
        'disable',
        'failure',
        'running:false',
      ]);
      expect(coordinator.isRunning, isFalse);
    },
  );
}

final class _RecordingScreenAwake implements SubmissionScreenAwake {
  const _RecordingScreenAwake(this.events);

  final List<String> events;

  @override
  Future<void> enable() async => events.add('enable');

  @override
  Future<void> disable() async => events.add('disable');
}
