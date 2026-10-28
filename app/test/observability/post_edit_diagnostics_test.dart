import 'dart:convert';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/edit_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_edit_refresh.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../feed/fakes/fake_post_repository.dart';
import '../test_support/post_fixtures.dart';
import '../test_support/serialized_sentry_transport.dart';
import '../test_support/session_registry_storage.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  for (final refresh in [false]) {
    test('IT-017 edit failure has one protected diagnostic owner '
        '(refresh=$refresh)', () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final lines = <String>[];
      final emitter = DiagnosticEmitter(platformSink: lines.add);
      final previous = Logger.root.level;
      Logger.root.level = Level.ALL;
      final subscription = Logger.root.onRecord.listen(emitter.emitLocal);
      addTearDown(subscription.cancel);
      addTearDown(() => Logger.root.level = previous);
      final reporter = _EditReporter();
      final post = postFixture();
      final seed = PostEditSeed(
        post: post,
        baseCid: post.cid,
        recordRevision: '3aaaaaaaaaaa2',
        editable: {
          'text': post.text,
          'sponsored': false,
        },
      );
      final accepted = PostEditResult(
        acceptedCid: post.copyWith(cid: 'bafyaccepted').cid,
        changed: true,
        editedAt: DateTime.utc(2026),
      );
      final stack = StackTrace.fromString(
        '#0 editTransport (package:craftsky_app/feed/edit_transport.dart:42:3)',
      );
      final cause = PlatformException(
        code: 'storage_failed',
        message: 'PRIVATE_DEPENDENCY_CANARY',
        details: {
          'authorization': 'Bearer PRIVATE_TOKEN_CANARY',
          'url': 'https://private.invalid/?token=PRIVATE_CAPABILITY_CANARY',
        },
      );
      final error = ApiServerError(
        'PRIVATE_RESPONSE_CANARY',
        details: ApiFailureDetails(
          statusCode: 502,
          method: 'PATCH',
          requestId: '10000000-0000-4000-8000-000000000001',
          appViewError: 'upstream_unavailable',
          cause: cause,
          stackTrace: stack,
        ),
      );
      final container = ProviderContainer.test(
        observers: [ProviderLogger(reporter: reporter)],
        overrides: [
          postEditRefreshProvider.overrideWithValue((_, _) {}),
          secureSessionRegistryStorageProvider.overrideWithValue(
            InMemorySessionRegistryStorage(SessionRegistry.empty()),
          ),
          errorReporterProvider.overrideWithValue(reporter),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onEdit:
                  (
                    did,
                    key, {
                    required changes,
                    required expectedCid,
                    required operationKey,
                  }) async {
                    if (!refresh) Error.throwWithStackTrace(error, stack);
                    return accepted;
                  },
              onFetch: (did, key) async =>
                  Error.throwWithStackTrace(error, stack),
            ),
          ),
        ],
      );
      final listener = container.listen(editPostProvider, (_, _) {});
      addTearDown(listener.close);
      addTearDown(container.dispose);
      await container.read(editPostProvider.future);
      final notifier = container.read(editPostProvider.notifier);
      final result = await notifier.save(
        seed: seed,
        changes: PostEditChanges({'text': 'PRIVATE_EDIT_BODY_CANARY'}),
      );
      expect(result, isNull);
      await transport.issueReceived.future.timeout(const Duration(seconds: 5));
      await Future.wait(reporter.pending);
      expect(reporter.errors, [same(error)]);
      expect(reporter.stacks, [same(stack)]);
      final events = transport.payloads
          .map(jsonDecode)
          .whereType<Map<String, dynamic>>()
          .where((event) => event.containsKey('exception'))
          .toList();
      expect(events, hasLength(1));
      final serialized = [...lines, ...transport.payloads].join('\n');
      for (final positive in [
        'PlatformException',
        'ApiServerError',
        'upstream_unavailable',
        'edit_transport.dart',
        '10000000-0000-4000-8000-000000000001',
        post.author.did.value,
        post.uri.value,
        if (refresh) 'accepted_read' else 'pds_write',
      ]) {
        expect(serialized, contains(positive));
      }
      for (final canary in [
        'PRIVATE_DEPENDENCY_CANARY',
        'PRIVATE_TOKEN_CANARY',
        'PRIVATE_CAPABILITY_CANARY',
        'PRIVATE_RESPONSE_CANARY',
        'PRIVATE_EDIT_BODY_CANARY',
      ]) {
        expect(serialized, isNot(contains(canary)));
      }
    });
  }
}

final class _EditReporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final pending = <Future<String?>>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) {
    errors.add(error);
    stacks.add(stackTrace);
    final future = const SentryErrorReporter().captureException(
      error,
      context: context,
      stackTrace: stackTrace,
    );
    pending.add(future);
    return future;
  }
}
