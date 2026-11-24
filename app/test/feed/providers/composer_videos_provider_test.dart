import 'dart:typed_data';

import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/foundation.dart'
    show TargetPlatform, debugDefaultTargetPlatformOverride;
import 'package:flutter/services.dart' show PlatformException;
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../../test_support/serialized_sentry_transport.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('video selection reports a protected Android picker failure', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    final transport = SerializedTransport();
    await Sentry.init((options) {
      options
        ..dsn = 'https://public@example.invalid/1'
        ..transport = transport;
      configureDiagnosticOptions(options);
    });
    addTearDown(Sentry.close);
    final logs = <String>[];
    final emitter = DiagnosticEmitter(platformSink: logs.add);
    final subscription = Logger('ComposerVideoController').onRecord.listen(
      emitter.emitLocal,
    );
    addTearDown(subscription.cancel);
    final failure = PlatformException(
      code: 'no_valid_video_uri',
      message: 'PRIVATE_CLIP_NAME_CANARY',
      details: {'uri': 'content://PRIVATE_PICKER_URI_CANARY'},
    );
    final stack = StackTrace.fromString(
      '#0 readVideo (package:craftsky_app/feed/video_picker.dart:10:2)',
    );
    final controller = ComposerVideoController(
      picker: _FailedPicker(failure, stack),
      reporter: const SentryErrorReporter(),
    );

    await expectLater(controller.selectExisting(), throwsA(same(failure)));
    expect(controller.selection, isNull);
    await transport.issueReceived.future;
    final events = transport.payloads.where(
      (payload) => payload.contains('"exception"'),
    );
    expect(events, hasLength(1));
    expect(events.single, contains('video.selection.failed'));
    expect(logs.single, contains('Video selection failed'));
    for (final payload in [logs.single, events.single]) {
      expect(payload, contains('selecting_local_video'));
      expect(payload, contains('no_valid_video_uri'));
      expect(payload, contains('video_picker.dart'));
      expect(payload, contains('android'));
      expect(payload, isNot(contains('PRIVATE_CLIP_NAME_CANARY')));
      expect(payload, isNot(contains('PRIVATE_PICKER_URI_CANARY')));
    }
  });

  test('AT-002 selection, replacement and removal stay local', () async {
    final picker = _FakePicker();
    final controller = ComposerVideoController(
      picker: picker,
    );

    await controller.selectExisting();
    expect(controller.selection?.displayName, 'local.mp4');
    await controller.replace();
    expect(picker.calls, 2);
    controller.remove();
    expect(controller.selection, isNull);
  });

  test(
    'AT-003 checks eligibility before picker and again at publish',
    () async {
      var eligibilityChecks = 0;
      final picker = _FakePicker();
      final controller = ComposerVideoController(
        picker: picker,
        checkEligibility: () async {
          eligibilityChecks++;
          return const VideoUploadLimits(
            canUpload: true,
            remainingDailyVideos: 1,
            remainingDailyBytes: 100,
          );
        },
      );

      await controller.selectExisting();
      controller.setAltText('A knitted blue scarf');
      final publishLimits = await controller.recheckEligibility();

      expect(eligibilityChecks, 2);
      expect(publishLimits?.canUpload, isTrue);
      expect(controller.selection?.altText, 'A knitted blue scarf');
    },
  );

  test('AT-003 disabled eligibility does not open picker', () async {
    final picker = _FakePicker();
    final controller = ComposerVideoController(
      picker: picker,
      checkEligibility: () async => const VideoUploadLimits(
        canUpload: false,
        remainingDailyVideos: 0,
        remainingDailyBytes: 0,
      ),
    );

    final limits = await controller.selectExisting();

    expect(limits?.canUpload, isFalse);
    expect(picker.calls, 0);
    expect(controller.selection, isNull);
  });
}

final class _FailedPicker implements ExistingVideoPicker {
  new(this.error, this.stack);

  final Object error;
  final StackTrace stack;

  @override
  Future<LocalVideoSelection?> pickExisting() => Future.error(error, stack);
}

final class _FakePicker implements ExistingVideoPicker {
  int calls = 0;

  @override
  Future<LocalVideoSelection?> pickExisting() async {
    calls++;
    return LocalVideoSelection(
      displayName: 'local.mp4',
      mimeType: 'video/mp4',
      byteLength: 8,
      duration: null,
      headerBytes: Uint8List.fromList(const [
        0,
        0,
        0,
        24,
        0x66,
        0x74,
        0x79,
        0x70,
        0x69,
        0x73,
        0x6f,
        0x6d,
      ]),
      openRead: () => Stream.value(const [
        0,
        0,
        0,
        24,
        0x66,
        0x74,
        0x79,
        0x70,
        0x69,
        0x73,
        0x6f,
        0x6d,
      ]),
    );
  }
}
