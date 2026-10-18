import 'dart:convert';

import 'package:craftsky_app/feed/media/bounded_image_file_reader.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/services/profile_image_error_reporting.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/serialized_sentry_transport.dart';

void main() {
  final stack = StackTrace.fromString(
    '#0 pickPhoto (package:craftsky_app/profile/photo.dart:12:3)',
  );

  test(
    'unknown photo failure preserves its original cause and caught stack',
    () async {
      final reporter = _Reporter();
      final error = PlatformException(
        code: 'codec_failed',
        message: 'PRIVATE_IMAGE',
      );
      await reportProfileImageFailure(
        reporter: reporter,
        error: error,
        stackTrace: stack,
        operation: ProfileImageOperation.onboarding,
        accountDid: 'did:plc:alice',
      );
      expect(reporter.errors, [same(error)]);
      expect(reporter.stacks, [same(stack)]);
      expect(reporter.contexts.single.feature, 'Onboarding');
      expect(
        reporter.contexts.single.workflow,
        isA<PrivateOperationalFailureContext>(),
      );
    },
  );

  test('expected size, format, permission, cancellation and offline failures '
      'do not create issues', () async {
    final reporter = _Reporter();
    for (final error in [
      const ProfileImagePickException(expectedRejection: true),
      const ProfileImagePickException(
        reason: ProfileImageFailure.tooManyPixels,
      ),
      const ProfileImagePickException(reason: ProfileImageFailure.fileTooLarge),
      const ImageSourceDimensionsTooLargeException(
        width: 8000,
        height: 6001,
        maxSide: 16384,
        maxPixels: 48000000,
      ),
      const ImageSourceTooLargeException(),
      PlatformException(code: 'photo_access_denied'),
      const ApiCanceled(),
      const ApiNetworkError('offline'),
    ]) {
      await reportProfileImageFailure(
        reporter: reporter,
        error: error,
        stackTrace: stack,
        operation: ProfileImageOperation.editor,
        accountDid: 'did:plc:alice',
      );
    }
    expect(reporter.errors, isEmpty);
  });

  test('an unclassified profile image error is reportable', () async {
    final reporter = _Reporter();
    const error = ProfileImagePickException();
    await reportProfileImageFailure(
      reporter: reporter,
      error: error,
      stackTrace: stack,
      operation: ProfileImageOperation.editor,
      accountDid: 'did:plc:alice',
    );
    expect(reporter.errors, [same(error)]);
  });

  test('unexpected PDS transfer rejections create issues', () async {
    final reporter = _Reporter();
    const error = ApiBadRequest('image_upload_failed');
    await reportProfileImageFailure(
      reporter: reporter,
      error: error,
      stackTrace: stack,
      operation: ProfileImageOperation.editor,
      accountDid: 'did:plc:alice',
    );
    expect(reporter.errors, [same(error)]);
    expect(reporter.contexts.single.outcome, DiagnosticOutcome.terminal);
  });

  test('reporter failure cannot change the photo-error flow', () async {
    final reporter = _Reporter()..fail = true;
    await reportProfileImageFailure(
      reporter: reporter,
      error: StateError('PRIVATE_IMAGE'),
      stackTrace: stack,
      operation: ProfileImageOperation.editor,
      accountDid: 'did:plc:alice',
    );
    expect(reporter.errors, hasLength(1));
  });

  test('serialized local and Sentry sinks keep context '
      'and exclude protected image values', () async {
    final transport = SerializedTransport();
    await Sentry.init((options) {
      options
        ..dsn = 'https://public@example.invalid/1'
        ..transport = transport;
      configureDiagnosticOptions(options, enableLogs: false);
    });
    addTearDown(Sentry.close);
    final local = <String>[];
    final previousLevel = Logger.root.level;
    Logger.root.level = Level.ALL;
    final sub = configureRootLogForwarding(platformSink: local.add);
    addTearDown(sub.cancel);
    addTearDown(() => Logger.root.level = previousLevel);
    final error = PlatformException(
      code: 'codec_failed',
      message: 'PRIVATE_IMAGE_PROSE',
      details: {
        'filename': 'PRIVATE_IMAGE_FILENAME.heic',
        'imageBytes': 'PRIVATE_IMAGE_BYTES',
        'authorization': 'Bearer PRIVATE_IMAGE_TOKEN',
        'uploadEndpoint':
            'https://private.invalid/photo?token=PRIVATE_IMAGE_CAPABILITY',
      },
    );
    await reportProfileImageFailure(
      reporter: const SentryErrorReporter(),
      error: error,
      stackTrace: stack,
      operation: ProfileImageOperation.editor,
      accountDid: 'did:plc:alice',
    );
    await Sentry.close();
    final events = transport.payloads
        .map(jsonDecode)
        .where((e) => (e as Map).containsKey('exception'))
        .toList();
    expect(events, hasLength(1));
    for (final output in [local.join(), jsonEncode(events)]) {
      expect(output, contains('profile.photo.pick_upload'));
      expect(output, contains('did:plc:alice'));
      expect(output, contains('PlatformException'));
      expect(output, contains('pickPhoto'));
      expect(output, isNot(contains('PRIVATE_IMAGE')));
      expect(output, isNot(contains('private.invalid')));
    }
  });
}

class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];
  bool fail = false;
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
    if (fail) throw StateError('Reporter failed');
    return null;
  }
}
