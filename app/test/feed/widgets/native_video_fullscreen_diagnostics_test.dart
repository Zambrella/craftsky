import 'dart:convert';

import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:media_kit/media_kit.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../../test_support/serialized_sentry_transport.dart';

void main() {
  for (final enter in [true, false]) {
    testWidgets(
      'fullscreen ${enter ? 'entry' : 'exit'} reports protected cause',
      (
        tester,
      ) async {
        debugDefaultTargetPlatformOverride = TargetPlatform.android;
        addTearDown(() => debugDefaultTargetPlatformOverride = null);
        final messenger =
            TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
              ..setMockMethodCallHandler(SystemChannels.platform, (_) async {
                throw PlatformException(
                  code: 'fullscreen_failed',
                  message: 'PRIVATE_VIDEO_ERROR',
                  details: {
                    'uri': 'https://private.invalid/?token=PRIVATE_VIDEO_TOKEN',
                  },
                );
              });
        addTearDown(
          () =>
              messenger.setMockMethodCallHandler(SystemChannels.platform, null),
        );
        final local = <String>[];
        final previousLevel = Logger.root.level;
        Logger.root.level = Level.ALL;
        final logSubscription = configureRootLogForwarding(
          platformSink: local.add,
        );
        addTearDown(logSubscription.cancel);
        addTearDown(() => Logger.root.level = previousLevel);
        final reporter = _Reporter();
        final adapter = MediaKitNativeVideoAdapter(
          player: _UnusedPlayer(),
          reporter: reporter,
        );
        late BuildContext videoContext;
        await tester.pumpWidget(
          MediaQuery(
            data: const MediaQueryData(size: Size(400, 800)),
            child: Builder(
              builder: (context) {
                videoContext = context;
                return const SizedBox.shrink();
              },
            ),
          ),
        );
        if (enter) {
          await adapter.enterFullscreen(videoContext, isPortraitVideo: true);
        } else {
          await adapter.exitFullscreen();
        }
        debugDefaultTargetPlatformOverride = null;
        expect(reporter.errors, hasLength(1));
        expect(reporter.errors.single, isA<PlatformException>());
        expect(reporter.stacks.single, isNotNull);
        final context = reporter.contexts.single;
        expect(context.operation, 'video.fullscreen');
        expect(
          context.safeDiagnostics['failureStage'],
          enter ? 'fullscreen_enter' : 'fullscreen_exit',
        );

        final transport = SerializedTransport();
        await tester.runAsync(() async {
          await Sentry.init((options) {
            options
              ..dsn = 'https://public@example.invalid/1'
              ..transport = transport;
            configureDiagnosticOptions(options, enableLogs: false);
          });
          try {
            await const SentryErrorReporter().captureException(
              reporter.errors.single,
              stackTrace: reporter.stacks.single,
              context: context,
            );
          } finally {
            await Sentry.close();
          }
        });
        final events = transport.payloads
            .map(jsonDecode)
            .where((event) => (event as Map).containsKey('exception'))
            .toList();
        expect(events, hasLength(1));
        for (final output in [local.join(), jsonEncode(events)]) {
          expect(output, contains('video.fullscreen'));
          expect(output, contains('PlatformException'));
          expect(
            output,
            contains(enter ? 'fullscreen_enter' : 'fullscreen_exit'),
          );
          expect(output, isNot(contains('PRIVATE_VIDEO')));
          expect(output, isNot(contains('private.invalid')));
        }
      },
    );
  }
}

final class _UnusedPlayer extends Fake implements Player;

final class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];

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
    return null;
  }
}
