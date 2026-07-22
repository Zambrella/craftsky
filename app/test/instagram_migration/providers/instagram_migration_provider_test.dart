import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_account_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_imports_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_suggestions_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_verification_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-011 late account A load cannot publish into account B state',
    () async {
      final initial = registry.SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final aliceLease = initial.activeLease!;
      final aliceLoad = Completer<InstagramAccountStatus>();
      final repositories = <AccountKey, InstagramMigrationRepository>{
        aliceLease.session.account: _Repository(onGetAccount: aliceLoad.future),
        AccountKey('did:plc:bob'): _Repository(
          onGetAccount: Future.value(
            const InstagramAccountStatus(
              integrationAvailable: true,
              account: null,
            ),
          ),
        ),
      };
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, lease) async => repositories[lease.session.account]!,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);

      final aliceFuture = container.read(
        instagramAccountProvider(aliceLease).future,
      );
      await Future<void>.delayed(Duration.zero);
      final bobSession = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(bobSession);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!;

      final bobStatus = await container.read(
        instagramAccountProvider(bobLease).future,
      );
      aliceLoad.complete(
        const InstagramAccountStatus(
          integrationAvailable: false,
          account: null,
        ),
      );
      await expectLater(
        aliceFuture,
        throwsA(isA<InstagramOperationDiscarded>()),
      );

      expect(bobStatus.integrationAvailable, isTrue);
      expect(
        container
            .read(instagramAccountProvider(bobLease))
            .requireValue
            .integrationAvailable,
        isTrue,
      );
    },
  );

  test(
    'UT-011 late account A verification create has no account B effect',
    () async {
      final initial = registry.SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final aliceLease = initial.activeLease!;
      final aliceCreate = Completer<InstagramVerificationAttempt>();
      final verificationStorage = _VerificationStorage();
      final repositories = <AccountKey, InstagramMigrationRepository>{
        aliceLease.session.account: _Repository(
          onGetAccount: Future.value(
            const InstagramAccountStatus(
              integrationAvailable: true,
              account: null,
            ),
          ),
          onCreateVerification: () => aliceCreate.future,
        ),
        AccountKey('did:plc:bob'): _Repository(
          onGetAccount: Future.value(
            const InstagramAccountStatus(
              integrationAvailable: true,
              account: null,
            ),
          ),
        ),
      };
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, lease) async => repositories[lease.session.account]!,
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            verificationStorage,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final subscription = container.listen(
        instagramVerificationProvider(aliceLease),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);

      final create = container
          .read(instagramVerificationProvider(aliceLease).notifier)
          .create();
      await Future<void>.delayed(Duration.zero);
      final bobSession = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(bobSession);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!;
      aliceCreate.complete(
        InstagramVerificationAttempt(
          verificationId: 'verification-a',
          state: InstagramVerificationState.pendingDm,
          expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
          challenge: 'PRIVATE-A',
          dmUrl: Uri.parse('https://instagram.example/dm'),
        ),
      );

      expect(await create, isFalse);
      expect(
        container.read(instagramVerificationProvider(bobLease)).attempt,
        isNull,
      );
      expect(verificationStorage.values, isEmpty);
    },
  );

  test(
    'IT-015 membership-inactive import reactivation does not renew consent',
    () async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = initial.activeLease!;
      final expiry = DateTime.utc(2026, 12);
      final inactive = InstagramImportSummary(
        importId: 'import-a',
        state: InstagramImportState.membershipInactive,
        sourceType: InstagramImportSourceType.instagramJson,
        retainUnmatched: true,
        retentionExpiresAt: expiry,
        followingCount: 3,
        createdAt: DateTime.utc(2026, 7),
      );
      InstagramImportPatch? sentPatch;
      final repository = _Repository(
        onGetAccount: Future.value(
          const InstagramAccountStatus(
            integrationAvailable: true,
            account: null,
          ),
        ),
        onListImports: () async => InstagramImportPage(
          items: [inactive],
          cursor: null,
        ),
        onUpdateImport: (id, patch) async {
          sentPatch = patch;
          return InstagramImportSummary(
            importId: id,
            state: InstagramImportState.active,
            sourceType: inactive.sourceType,
            retainUnmatched: inactive.retainUnmatched,
            retentionExpiresAt: expiry,
            followingCount: inactive.followingCount,
            createdAt: inactive.createdAt,
          );
        },
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await container.read(instagramImportsProvider(lease).future);

      expect(
        await container
            .read(instagramImportsProvider(lease).notifier)
            .reactivate('import-a'),
        isTrue,
      );

      expect(sentPatch?.reactivate, isTrue);
      expect(sentPatch?.retainUnmatched, isNull);
      expect(
        container
            .read(instagramImportsProvider(lease))
            .requireValue
            .items
            .single
            .retentionExpiresAt,
        expiry,
      );
    },
  );

  test(
    'FR-026 select all includes only reviewed pending suggestions',
    () async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = initial.activeLease!;
      InstagramSuggestion suggestion(
        String id,
        InstagramSuggestionState state,
      ) => InstagramSuggestion(
        suggestionId: id,
        profile: InstagramSuggestionProfile(
          did: 'did:plc:$id',
          handle: '$id.test',
        ),
        reason: InstagramSuggestionReason.verifiedInstagramFollow,
        state: state,
      );
      final repository = _Repository(
        onGetAccount: Future.value(
          const InstagramAccountStatus(
            integrationAvailable: true,
            account: null,
          ),
        ),
        onListSuggestions: () async => InstagramSuggestionPage(
          items: [
            suggestion('pending', InstagramSuggestionState.pending),
            suggestion('accepted', InstagramSuggestionState.alreadyFollowing),
            suggestion('invalid', InstagramSuggestionState.invalidated),
          ],
          cursor: null,
        ),
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await container.read(instagramSuggestionsProvider(lease).future);

      container
          .read(instagramSuggestionsProvider(lease).notifier)
          .selectAllReviewed();

      expect(
        container
            .read(instagramSuggestionsProvider(lease))
            .requireValue
            .selectedIds,
        {'pending'},
      );
    },
  );

  test(
    'UT-011 late account A import cannot update account B imports',
    () async {
      final initial = registry.SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final aliceLease = initial.activeLease!;
      final lateImport = Completer<InstagramImportCreateResult>();
      Future<InstagramImportPage> emptyImports() async => InstagramImportPage(
        items: const [],
        cursor: null,
      );
      final repositories = <AccountKey, InstagramMigrationRepository>{
        aliceLease.session.account: _Repository(
          onGetAccount: Future.value(
            const InstagramAccountStatus(
              integrationAvailable: true,
              account: null,
            ),
          ),
          onListImports: emptyImports,
          onCreateImport: (_) => lateImport.future,
        ),
        AccountKey('did:plc:bob'): _Repository(
          onGetAccount: Future.value(
            const InstagramAccountStatus(
              integrationAvailable: true,
              account: null,
            ),
          ),
          onListImports: emptyImports,
        ),
      };
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, lease) async => repositories[lease.session.account]!,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await container.read(instagramImportsProvider(aliceLease).future);
      final create = container
          .read(instagramImportsProvider(aliceLease).notifier)
          .create(
            InstagramImportRequest(
              sourceType: InstagramImportSourceType.manual,
              retainUnmatched: false,
              entries: const [
                InstagramImportEntry(username: 'private_a'),
              ],
            ),
          );
      await Future<void>.delayed(Duration.zero);
      final bobSession = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(bobSession);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!;
      await container.read(instagramImportsProvider(bobLease).future);
      lateImport.complete(
        InstagramImportCreateResult(
          import: InstagramImportSummary(
            importId: 'import-a',
            state: InstagramImportState.active,
            sourceType: InstagramImportSourceType.manual,
            retainUnmatched: false,
            retentionExpiresAt: null,
            followingCount: 1,
            createdAt: DateTime.utc(2026, 7, 19),
          ),
          followingCount: 1,
          initialSuggestionCount: 0,
        ),
      );

      expect(await create, isNull);
      expect(
        container.read(instagramImportsProvider(bobLease)).requireValue.items,
        isEmpty,
      );
    },
  );

  test(
    'IT-015 verification polling timer stops when provider disposes',
    () async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = initial.activeLease!;
      var pollCalls = 0;
      final repository = _Repository(
        onGetAccount: Future.value(
          const InstagramAccountStatus(
            integrationAvailable: true,
            account: null,
          ),
        ),
        onCreateVerification: () async => InstagramVerificationAttempt(
          verificationId: 'verification-a',
          state: InstagramVerificationState.pendingDm,
          expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
          challenge: 'PRIVATE-A',
          dmUrl: Uri.parse('https://instagram.example/dm'),
        ),
        onGetVerification: (_) async {
          pollCalls++;
          return InstagramVerificationAttempt(
            verificationId: 'verification-a',
            state: InstagramVerificationState.pendingDm,
            expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
          );
        },
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramVerificationPollIntervalProvider.overrideWithValue(
            const Duration(milliseconds: 1),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final subscription = container.listen(
        instagramVerificationProvider(lease),
        (_, _) {},
        fireImmediately: true,
      );
      expect(
        await container
            .read(instagramVerificationProvider(lease).notifier)
            .create(),
        isTrue,
      );
      await _waitUntil(() => pollCalls > 0);
      expect(pollCalls, greaterThan(0));

      subscription.close();
      await container.pump();
      await Future<void>.delayed(const Duration(milliseconds: 20));
      final callsAtDispose = pollCalls;
      await Future<void>.delayed(const Duration(milliseconds: 50));

      expect(pollCalls, callsAtDispose);
    },
  );

  test(
    'IT-022 verification resumes after page disposal without creating again',
    () async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = initial.activeLease!;
      final storage = _VerificationStorage();
      final now = DateTime.utc(2026, 7, 22, 15);
      var createCalls = 0;
      InstagramVerificationAttempt? current;
      final repository = _Repository(
        onGetAccount: Future.value(
          const InstagramAccountStatus(
            integrationAvailable: true,
            account: null,
          ),
        ),
        onGetCurrentVerification: () async => current,
        onCreateVerification: () async {
          createCalls++;
          return current = InstagramVerificationAttempt(
            verificationId: 'verification-a',
            state: InstagramVerificationState.pendingDm,
            expiresAt: now.add(const Duration(minutes: 10)),
            challenge: 'CSKY-PRIVATE-A',
            dmUrl: Uri.parse('https://instagram.example/dm'),
          );
        },
        onGetVerification: (_) async => current!,
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramVerificationStorageProvider.overrideWithValue(storage),
          instagramVerificationNowProvider.overrideWithValue(() => now),
          instagramVerificationPollIntervalProvider.overrideWithValue(
            const Duration(days: 1),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      var subscription = container.listen(
        instagramVerificationProvider(lease),
        (_, _) {},
        fireImmediately: true,
      );
      await _waitUntil(
        () => !container.read(instagramVerificationProvider(lease)).isBusy,
      );

      expect(
        await container
            .read(instagramVerificationProvider(lease).notifier)
            .create(),
        isTrue,
      );
      expect(createCalls, 1);
      expect(
        storage.values[lease.session.account]?.challenge,
        'CSKY-PRIVATE-A',
      );

      subscription.close();
      await container.pump();
      current = InstagramVerificationAttempt(
        verificationId: 'verification-a',
        state: InstagramVerificationState.pendingConfirmation,
        expiresAt: now.add(const Duration(minutes: 10)),
        candidateUsername: 'synthetic.candidate',
      );

      subscription = container.listen(
        instagramVerificationProvider(lease),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await _waitUntil(() {
        final resumed = container.read(instagramVerificationProvider(lease));
        return !resumed.isBusy &&
            resumed.attempt?.state ==
                InstagramVerificationState.pendingConfirmation;
      });

      final resumed = container
          .read(instagramVerificationProvider(lease))
          .attempt!;
      expect(createCalls, 1);
      expect(resumed.verificationId, 'verification-a');
      expect(resumed.challenge, 'CSKY-PRIVATE-A');
      expect(resumed.dmUrl, Uri.parse('https://instagram.example/dm'));
      expect(resumed.candidateUsername, 'synthetic.candidate');
    },
  );

  test('IT-022 server attempt wins over a mismatched local snapshot', () async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = initial.activeLease!;
    final storage = _VerificationStorage();
    final now = DateTime.utc(2026, 7, 22, 15);
    storage.values[lease.session.account] = InstagramVerificationSnapshot(
      verificationId: 'stale-verification',
      challenge: 'CSKY-STALE-PRIVATE',
      dmUrl: Uri.parse('https://instagram.example/stale'),
      expiresAt: now.add(const Duration(minutes: 10)),
    );
    final repository = _Repository(
      onGetAccount: Future.value(
        const InstagramAccountStatus(
          integrationAvailable: true,
          account: null,
        ),
      ),
      onGetCurrentVerification: () async => InstagramVerificationAttempt(
        verificationId: 'server-verification',
        state: InstagramVerificationState.processing,
        expiresAt: now.add(const Duration(minutes: 10)),
      ),
      onGetVerification: (_) async => InstagramVerificationAttempt(
        verificationId: 'server-verification',
        state: InstagramVerificationState.processing,
        expiresAt: now.add(const Duration(minutes: 10)),
      ),
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(initial),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, _) async => repository,
        ),
        instagramVerificationStorageProvider.overrideWithValue(storage),
        instagramVerificationNowProvider.overrideWithValue(() => now),
        instagramVerificationPollIntervalProvider.overrideWithValue(
          const Duration(days: 1),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final subscription = container.listen(
      instagramVerificationProvider(lease),
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(subscription.close);
    await _waitUntil(
      () => !container.read(instagramVerificationProvider(lease)).isBusy,
    );

    final attempt = container
        .read(instagramVerificationProvider(lease))
        .attempt!;
    expect(attempt.verificationId, 'server-verification');
    expect(attempt.challenge, isNull);
    expect(attempt.dmUrl, isNull);
    expect(storage.values[lease.session.account], isNull);
  });

  test('IT-022 verification expiry clears its secure snapshot', () async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = initial.activeLease!;
    final storage = _VerificationStorage();
    final now = DateTime.utc(2026, 7, 22, 15);
    final repository = _Repository(
      onGetAccount: Future.value(
        const InstagramAccountStatus(
          integrationAvailable: true,
          account: null,
        ),
      ),
      onCreateVerification: () async => InstagramVerificationAttempt(
        verificationId: 'verification-expiring',
        state: InstagramVerificationState.pendingDm,
        expiresAt: now.add(const Duration(milliseconds: 20)),
        challenge: 'CSKY-EXPIRING-PRIVATE',
        dmUrl: Uri.parse('https://instagram.example/dm'),
      ),
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(initial),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, _) async => repository,
        ),
        instagramVerificationStorageProvider.overrideWithValue(storage),
        instagramVerificationNowProvider.overrideWithValue(() => now),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final subscription = container.listen(
      instagramVerificationProvider(lease),
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(subscription.close);
    await _waitUntil(
      () => !container.read(instagramVerificationProvider(lease)).isBusy,
    );
    expect(
      await container
          .read(instagramVerificationProvider(lease).notifier)
          .create(),
      isTrue,
    );
    expect(storage.values[lease.session.account], isNotNull);

    await Future<void>.delayed(const Duration(milliseconds: 40));
    await container.pump();

    expect(
      container.read(instagramVerificationProvider(lease)).attempt?.state,
      InstagramVerificationState.expired,
    );
    expect(storage.values[lease.session.account], isNull);
  });

  test('UT-011 late account A acceptance has no account B effect', () async {
    final initial = registry.SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final aliceLease = initial.activeLease!;
    final lateAcceptance = Completer<InstagramSuggestionActionResult>();
    const suggestion = InstagramSuggestion(
      suggestionId: 'suggestion-a',
      profile: InstagramSuggestionProfile(
        did: 'did:plc:target',
        handle: 'target.test',
      ),
      reason: InstagramSuggestionReason.verifiedInstagramFollow,
      state: InstagramSuggestionState.pending,
    );
    final repositories = <AccountKey, InstagramMigrationRepository>{
      aliceLease.session.account: _Repository(
        onGetAccount: Future.value(
          const InstagramAccountStatus(
            integrationAvailable: true,
            account: null,
          ),
        ),
        onListSuggestions: () async => InstagramSuggestionPage(
          items: [suggestion],
          cursor: null,
        ),
        onAcceptSuggestion: (_) => lateAcceptance.future,
      ),
      AccountKey('did:plc:bob'): _Repository(
        onGetAccount: Future.value(
          const InstagramAccountStatus(
            integrationAvailable: true,
            account: null,
          ),
        ),
        onListSuggestions: () async => InstagramSuggestionPage(
          items: const [],
          cursor: null,
        ),
      ),
    };
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(initial),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, lease) async => repositories[lease.session.account]!,
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    await container.read(instagramSuggestionsProvider(aliceLease).future);
    final accept = container
        .read(instagramSuggestionsProvider(aliceLease).notifier)
        .accept('suggestion-a');
    await Future<void>.delayed(Duration.zero);
    final bobSession = container
        .read(sessionRegistryProvider)
        .requireValue
        .leaseFor(AccountKey('did:plc:bob'))!;
    await container.read(sessionRegistryProvider.notifier).activate(bobSession);
    final bobLease = container
        .read(sessionRegistryProvider)
        .requireValue
        .activeLease!;
    await container.read(instagramSuggestionsProvider(bobLease).future);
    lateAcceptance.complete(
      const InstagramSuggestionActionResult(
        suggestionId: 'suggestion-a',
        state: InstagramSuggestionState.accepted,
      ),
    );

    expect(await accept, isFalse);
    expect(
      container.read(instagramSuggestionsProvider(bobLease)).requireValue.items,
      isEmpty,
    );
  });
}

Future<void> _waitUntil(bool Function() condition) async {
  final deadline = DateTime.now().add(const Duration(seconds: 1));
  while (!condition() && DateTime.now().isBefore(deadline)) {
    await Future<void>.delayed(const Duration(milliseconds: 5));
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  registry.SessionRegistry value;

  @override
  Future<registry.SessionRegistry> read() async => value;

  @override
  Future<void> write(registry.SessionRegistry registry) async =>
      value = registry;
}

final class _Repository implements InstagramMigrationRepository {
  const _Repository({
    required this.onGetAccount,
    this.onCreateVerification,
    this.onListImports,
    this.onUpdateImport,
    this.onListSuggestions,
    this.onCreateImport,
    this.onGetVerification,
    this.onGetCurrentVerification,
    this.onAcceptSuggestion,
  });

  final Future<InstagramAccountStatus> onGetAccount;
  final Future<InstagramVerificationAttempt> Function()? onCreateVerification;
  final Future<InstagramImportPage> Function()? onListImports;
  final Future<InstagramImportSummary> Function(
    String id,
    InstagramImportPatch patch,
  )?
  onUpdateImport;
  final Future<InstagramSuggestionPage> Function()? onListSuggestions;
  final Future<InstagramImportCreateResult> Function(
    InstagramImportRequest request,
  )?
  onCreateImport;
  final Future<InstagramVerificationAttempt> Function(String verificationId)?
  onGetVerification;
  final Future<InstagramVerificationAttempt?> Function()?
  onGetCurrentVerification;
  final Future<InstagramSuggestionActionResult> Function(String suggestionId)?
  onAcceptSuggestion;

  @override
  Future<InstagramVerificationAttempt> createVerification() =>
      onCreateVerification!.call();

  @override
  Future<InstagramVerificationAttempt> getVerification(
    String verificationId,
  ) => onGetVerification!.call(verificationId);

  @override
  Future<InstagramVerificationAttempt?> getCurrentVerification() =>
      onGetCurrentVerification?.call() ?? Future.value();

  @override
  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  ) => onCreateImport!.call(request);

  @override
  Future<InstagramSuggestionActionResult> acceptSuggestion(
    String suggestionId,
  ) => onAcceptSuggestion!.call(suggestionId);

  @override
  Future<InstagramImportPage> listImports({int? limit, String? cursor}) =>
      onListImports!.call();

  @override
  Future<InstagramImportSummary> updateImport(
    String importId,
    InstagramImportPatch patch,
  ) => onUpdateImport!.call(importId, patch);

  @override
  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) => onListSuggestions!.call();

  @override
  Future<InstagramAccountStatus> getAccount() => onGetAccount;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _VerificationStorage implements InstagramVerificationStorage {
  final values = <AccountKey, InstagramVerificationSnapshot>{};

  @override
  Future<void> delete(AccountKey account, {String? verificationId}) async {
    if (verificationId == null ||
        values[account]?.verificationId == verificationId) {
      values.remove(account);
    }
  }

  @override
  Future<InstagramVerificationSnapshot?> read(AccountKey account) async =>
      values[account];

  @override
  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  ) async => values[account] = snapshot;
}
