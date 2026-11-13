import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/video_draft_stream_store.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../../test_support/serialized_sentry_transport.dart';

void main() {
  test(
    'draft native cause chains retain useful evidence in protected sinks',
    () async {
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
      final subscription = configureRootLogForwarding(platformSink: local.add);
      addTearDown(subscription.cancel);
      addTearDown(() => Logger.root.level = previousLevel);
      final stack = StackTrace.fromString(
        '#0 readDraft (package:craftsky_app/drafts/storage.dart:42:7)',
      );
      final failures = [
        DraftRepositoryException(
          DraftRepositoryFailureReason.storageUnavailable,
          diagnosticStack: stack,
          diagnosticCause: DraftFileStoreException(
            DraftFileStoreFailureReason.ioFailure,
            diagnosticStack: stack,
            diagnosticCause: const FileSystemException(
              'PRIVATE_DRAFT_PROSE',
              '/PRIVATE_DRAFT_PATH/draft.json',
            ),
          ),
        ),
        DraftManifestException(
          DraftManifestFailureReason.corrupt,
          diagnosticStack: stack,
          diagnosticCause: const FormatException(
            'PRIVATE_DRAFT_PROSE',
            'PRIVATE_DRAFT_BODY',
          ),
        ),
        VideoDraftStreamException(
          diagnosticStack: stack,
          diagnosticCause: StateError('PRIVATE_VIDEO_PROSE'),
        ),
      ];
      const context = ReportContext(
        feature: 'Drafts',
        operation: 'draft.load',
        classification: 'draft.load.failed',
        workflow: PrivateOperationalFailureContext(accountDid: 'did:plc:alice'),
      );
      for (final failure in failures) {
        Logger('Drafts').severe(
          const DiagnosticMessage('Draft operation failed', context: context),
          failure,
          stack,
        );
        await const SentryErrorReporter().captureException(
          failure,
          stackTrace: stack,
          context: context,
        );
      }
      await Sentry.close();
      final issues = transport.payloads
          .map(jsonDecode)
          .where((value) => (value as Map).containsKey('exception'))
          .toList();
      expect(issues, hasLength(3));
      for (final output in [local.join(), jsonEncode(issues)]) {
        for (final positive in [
          'DraftRepositoryException',
          'DraftFileStoreException',
          'FileSystemException',
          'DraftManifestException',
          'FormatException',
          'VideoDraftStreamException',
          'StateError',
          'readDraft',
          'draft.load',
          'did:plc:alice',
        ]) {
          expect(output, contains(positive));
        }
        expect(output, isNot(contains('PRIVATE_')));
      }
    },
  );

  test('filesystem mapping keeps the actual native cause and stack', () async {
    final root = await Directory.systemTemp.createTemp('draft-errors-');
    addTearDown(() => root.delete(recursive: true));
    final missing = '${root.path}/PRIVATE_DRAFT/missing.json';
    final failure = await _caught(() => IoDraftFileStore().readBytes(missing));
    expect(failure, isA<DraftFileStoreException>());
    expect(
      failure,
      isA<DiagnosticFailureCause>()
          .having(
            (value) => value.diagnosticCause,
            'cause',
            isA<FileSystemException>(),
          )
          .having((value) => value.diagnosticStack, 'stack', isNotNull),
    );
    expect(failure.toString(), isNot(contains('PRIVATE_DRAFT')));
  });

  test('repository storage mapping retains its native error chain', () async {
    final root = await Directory.systemTemp.createTemp('draft-errors-');
    addTearDown(() => root.delete(recursive: true));
    final blockingFile = File('${root.path}/PRIVATE_DRAFT');
    await blockingFile.writeAsString('PRIVATE_CONTENT');
    final owner = AccountKey('did:plc:alice');
    final repository = FileLocalPostDraftRepository(
      documentsRoot: blockingFile.path,
      owner: owner,
    );
    final failure = await _caught(
      () => repository.save(
        DraftWriteRequest(
          id: '00000000-0000-4000-8000-000000000001',
          owner: owner,
          kind: LocalPostDraftKind.standard,
          content: const StandardDraftContent(
            text: 'PRIVATE_CONTENT',
            languages: ['en'],
          ),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: const [],
        ),
      ),
    );
    expect(failure, isA<DraftRepositoryException>());
    expect(
      failure,
      isA<DiagnosticFailureCause>().having(
        (value) => value.diagnosticCause,
        'filesystem wrapper',
        isA<DraftFileStoreException>().having(
          (value) => value as Object,
          'native cause carrier',
          isA<DiagnosticFailureCause>().having(
            (value) => value.diagnosticCause,
            'native cause',
            isA<FileSystemException>(),
          ),
        ),
      ),
    );
    expect(failure.toString(), isNot(contains('PRIVATE_')));
  });

  test(
    'manifest decoding keeps its parser cause without exposing private text',
    () {
      Object? failure;
      try {
        DraftManifestCodec.decode('PRIVATE_DRAFT_CONTENT');
      } on Object catch (error) {
        failure = error;
      }
      expect(failure, isA<DraftManifestException>());
      expect(
        failure,
        isA<DiagnosticFailureCause>()
            .having(
              (value) => value.diagnosticCause,
              'parser cause',
              isA<FormatException>(),
            )
            .having((value) => value.diagnosticStack, 'stack', isNotNull),
      );
      expect(failure.toString(), isNot(contains('PRIVATE_')));
    },
  );

  test(
    'video stream mapping keeps the original read failure and stack',
    () async {
      final root = await Directory.systemTemp.createTemp('draft-errors-');
      addTearDown(() => root.delete(recursive: true));
      final original = StateError('PRIVATE_VIDEO_SOURCE');
      final stack = StackTrace.fromString(
        '#0 readSource (package:craftsky_app/drafts/source.dart:42:7)',
      );
      final failure = await _caught(
        () => writeVideoDraftStream(
          source: Stream<List<int>>.error(original, stack),
          targetPath: '${root.path}/PRIVATE_VIDEO.mp4',
          maximumBytes: 100,
        ),
      );
      expect(failure, isA<VideoDraftStreamException>());
      expect(
        failure,
        isA<DiagnosticFailureCause>()
            .having((value) => value.diagnosticCause, 'cause', same(original))
            .having(
              (value) => value.diagnosticStack.toString(),
              'stack',
              stack.toString(),
            ),
      );
    },
  );
}

Future<Object> _caught(Future<Object?> Function() action) async {
  try {
    await action();
  } on Object catch (error) {
    return error;
  }
  fail('Expected a mapped persistence failure');
}
