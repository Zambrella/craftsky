import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_imports_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/onboarding/data/onboarding_repository.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_repository_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../profile/fakes/fake_profile_repository.dart';

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _InstagramRepository implements InstagramMigrationRepository {
  final createResult = Completer<InstagramImportCreateResult>();

  @override
  Future<InstagramImportPage> listImports({int? limit, String? cursor}) async =>
      InstagramImportPage(items: const [], cursor: null);

  @override
  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  ) => createResult.future;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _CompletionRepository implements OnboardingRepository {
  _CompletionRepository({required this.completion});

  final Completer<OnboardingCompletion>? completion;

  @override
  Future<OnboardingCompletion> readStatus() async =>
      const OnboardingCompletion(completed: false);

  @override
  Future<OnboardingCompletion> complete() =>
      completion?.future ??
      Future.value(const OnboardingCompletion(completed: true));
}

void main() {
  test('AT-015 a late account A profile read is rejected', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final aliceSession = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bobSession = registry.leaseFor(AccountKey('did:plc:bob'))!;
    registry = registry.activate(aliceSession);
    final aliceLease = registry.activeLease!;
    final read = Completer<Profile>();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith((ref, lease) async {
          return FakeProfileRepository(onFetchMe: () => read.future);
        }),
      ],
    );
    final provider = onboardingFlowProvider(aliceLease);
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(provider.future);
    await Future<void>.delayed(Duration.zero);

    await container.read(sessionRegistryProvider.notifier).activate(bobSession);
    read.complete(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        displayName: 'Late Alice',
        crafts: const [],
      ),
    );

    await expectLater(container.read(provider.future), throwsStateError);
  });

  test('AT-015 a late account A avatar failure is discarded', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final aliceSession = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bobSession = registry.leaseFor(AccountKey('did:plc:bob'))!;
    registry = registry.activate(aliceSession);
    final aliceLease = registry.activeLease!;
    final picker = Completer<ProfileImagePicker>();
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => FakeProfileRepository(
            onFetchMe: () async => Profile(
              did: 'did:plc:alice',
              handle: 'alice.test',
              displayName: 'Alice',
              crafts: const [],
            ),
          ),
        ),
        accountProfileImagePickerProvider.overrideWith(
          (ref, lease) => picker.future,
        ),
      ],
    );
    final provider = onboardingFlowProvider(aliceLease);
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(provider.future);
    final operation = container.read(provider.notifier).pickAvatar();
    await Future<void>.delayed(Duration.zero);

    await container.read(sessionRegistryProvider.notifier).activate(bobSession);
    picker.completeError(StateError('late picker failure'));
    await operation;

    final alice = container.read(provider).requireValue;
    expect(alice.avatarPreview, isNull);
    expect(alice.avatarUploadFailed, isFalse);
  });

  test('AT-015 a late account A Instagram action is discarded', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final aliceSession = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bobSession = registry.leaseFor(AccountKey('did:plc:bob'))!;
    registry = registry.activate(aliceSession);
    final aliceLease = registry.activeLease!;
    final repository = _InstagramRepository();
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, lease) async => repository,
        ),
      ],
    );
    final provider = instagramImportsProvider(aliceLease);
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(provider.future);
    final operation = container
        .read(provider.notifier)
        .create(
          InstagramImportRequest(
            sourceType: InstagramImportSourceType.manual,
            entries: [const InstagramImportEntry(username: 'maker')],
          ),
        );
    await Future<void>.delayed(Duration.zero);

    await container.read(sessionRegistryProvider.notifier).activate(bobSession);
    repository.createResult.complete(
      InstagramImportCreateResult(
        import: InstagramImportSummary(
          importId: 'late-import',
          state: InstagramImportState.active,
          sourceType: InstagramImportSourceType.manual,
          followingCount: 1,
          createdAt: DateTime.utc(2026, 8, 31),
        ),
        followingCount: 1,
      ),
    );

    expect(await operation, isNull);
    expect(container.read(provider).requireValue.items, isEmpty);
  });

  test('AT-015 account A completion cannot complete account B', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final aliceSession = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bobSession = registry.leaseFor(AccountKey('did:plc:bob'))!;
    registry = registry.activate(aliceSession);
    final completion = Completer<OnboardingCompletion>();
    final aliceRepository = _CompletionRepository(completion: completion);
    final bobRepository = _CompletionRepository(completion: null);
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        onboardingRepositoryProvider.overrideWith((ref, lease) async {
          return lease.account == aliceSession.account
              ? aliceRepository
              : bobRepository;
        }),
      ],
    );
    final aliceProvider = onboardingStatusProvider(aliceSession);
    final aliceSubscription = container.listen(aliceProvider, (_, _) {});
    addTearDown(aliceSubscription.close);
    await container.read(aliceProvider.future);
    final operation = container
        .read(aliceProvider.notifier)
        .completeOptimistically();
    await Future<void>.delayed(Duration.zero);

    await container.read(sessionRegistryProvider.notifier).activate(bobSession);
    final activeBobSession = container
        .read(sessionRegistryProvider)
        .requireValue
        .activeLease!
        .session;
    final bobProvider = onboardingStatusProvider(activeBobSession);
    final bobSubscription = container.listen(bobProvider, (_, _) {});
    addTearDown(bobSubscription.close);
    expect((await container.read(bobProvider.future)).completed, isFalse);

    completion.complete(const OnboardingCompletion(completed: true));
    await operation;
    expect(container.read(bobProvider).requireValue.completed, isFalse);
    expect(container.read(aliceProvider).requireValue.completed, isTrue);
  });

  test(
    'AT-015 a late account A profile save cannot update account B',
    () async {
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final aliceLease = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
      registry = registry.activate(aliceLease);
      final save = Completer<Profile>();
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountProfileRepositoryProvider.overrideWith((ref, lease) async {
            final isAlice = lease.session.account == aliceLease.account;
            return FakeProfileRepository(
              onFetchMe: () async => Profile(
                did: isAlice ? 'did:plc:alice' : 'did:plc:bob',
                handle: isAlice ? 'alice.test' : 'bob.test',
                displayName: isAlice ? 'Alice' : 'Bob',
                crafts: const [],
              ),
              onUpdateMe:
                  ({
                    displayName,
                    description,
                    crafts,
                    avatar,
                    clearAvatar = false,
                    banner,
                    clearBanner = false,
                  }) => save.future,
            );
          }),
        ],
      );
      final aliceProvider = onboardingFlowProvider(registry.activeLease!);
      final aliceSubscription = container.listen(aliceProvider, (_, _) {});
      addTearDown(aliceSubscription.close);
      await container.read(aliceProvider.future);
      final aliceNotifier = container.read(aliceProvider.notifier);
      final operation =
          (aliceNotifier..updateIdentity(displayName: 'Late Alice'))
              .saveAndNext();
      await Future<void>.delayed(Duration.zero);

      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      save.complete(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          displayName: 'Late Alice',
          crafts: const [],
        ),
      );
      await operation;

      final bobProvider = onboardingFlowProvider(
        container.read(sessionRegistryProvider).requireValue.activeLease!,
      );
      final bobPrefilled = Completer<void>();
      final bobSubscription = container.listen(bobProvider, (_, next) {
        if (next.value?.identity.displayName == 'Bob' &&
            !bobPrefilled.isCompleted) {
          bobPrefilled.complete();
        }
      });
      addTearDown(bobSubscription.close);
      await container.read(bobProvider.future);
      await bobPrefilled.future.timeout(const Duration(seconds: 1));
      final bob = container.read(bobProvider).requireValue;
      expect(bob.identity.displayName, 'Bob');
      expect(bob.step.index, 0);
    },
  );
}
