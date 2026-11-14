import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/drafts/composer/save_composer_snapshot.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/session_registry_storage.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  for (final published in [true, false]) {
    test('draft snapshot failure published=$published has one owner', () async {
      final account = AccountKey('did:plc:alice');
      final original = StateError('PRIVATE_DRAFT_CONTENT');
      final stack = StackTrace.fromString(
        '#0 persistDraft (package:craftsky_app/drafts/persist.dart:12:3)',
      );
      final reporter = _Reporter();
      final container = ProviderContainer.test(
        observers: [ProviderLogger(reporter: reporter)],
        retry: appProviderRetry,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            InMemorySessionRegistryStorage(SessionRegistry.empty()),
          ),
          accountLocalPostDraftRepositoryProvider(account).overrideWith(
            (ref) async => _FailingRepository(original, stack),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: const _Awake(),
        reporter: reporter,
      );
      var failed = false;
      await coordinator.run(
        accountDid: account.did.value,
        presentOverlay: () async {},
        ownershipIsCurrent: () => true,
        saveOriginSnapshot: () async {
          await saveComposerSnapshot(
            container: container,
            owner: account,
            request: DraftWriteRequest(
              id: '90000000-0000-4000-8000-000000000001',
              owner: published ? account : AccountKey('did:plc:other'),
              kind: LocalPostDraftKind.standard,
              content: const StandardDraftContent(
                text: 'PRIVATE_DRAFT_CONTENT',
                languages: ['en'],
              ),
              schedule: const DraftScheduleIntent.now(),
              orderedMedia: const [],
            ),
          );
        },
        operation: () async =>
            fail('write must not start after failed snapshot'),
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async {},
        onRunningChanged: ({required running}) {},
        onFailure: (_) => failed = true,
      );
      expect(failed, isTrue);
      if (published) {
        expect(reporter.errors, [same(original)]);
        expect(reporter.stacks, [same(stack)]);
        expect(reporter.contexts.single.feature, 'draftSaveControllerProvider');
      } else {
        expect(reporter.errors.single, isA<DraftRepositoryException>());
        expect(reporter.contexts.single.feature, 'ComposerSubmission');
      }
    });
  }
}

final class _FailingRepository extends Fake
    implements LocalPostDraftRepository {
  new(this.error, this.stack);
  final Object error;
  final StackTrace stack;
  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) async =>
      Error.throwWithStackTrace(error, stack);
}

final class _Awake implements SubmissionScreenAwake {
  const new();
  @override
  Future<void> enable() async {}
  @override
  Future<void> disable() async {}
}

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
