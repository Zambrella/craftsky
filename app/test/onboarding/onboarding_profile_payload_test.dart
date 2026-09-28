import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/onboarding/data/onboarding_profile_payload.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
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

final class _Update {
  const _Update({
    required this.displayName,
    required this.pronouns,
    required this.description,
    required this.crafts,
    required this.avatar,
    required this.clearAvatar,
    required this.banner,
    required this.clearBanner,
  });

  final String? displayName;
  final String? pronouns;
  final String? description;
  final List<String>? crafts;
  final UploadedBlob? avatar;
  final bool clearAvatar;
  final UploadedBlob? banner;
  final bool clearBanner;
}

void main() {
  test('AT-017 identity-only save sends the preserved full snapshot', () async {
    final profile = _fullProfile();
    _Update? sent;
    final harness = _flowHarness(
      profile,
      onUpdate:
          ({
            displayName,
            pronouns,
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async {
            sent = _Update(
              displayName: displayName,
              pronouns: pronouns,
              description: description,
              crafts: crafts,
              avatar: avatar,
              clearAvatar: clearAvatar,
              banner: banner,
              clearBanner: clearBanner,
            );
            return Profile(
              did: profile.did,
              handle: profile.handle,
              displayName: displayName,
              pronouns: pronouns,
              description: description,
              avatar: profile.avatar,
              banner: profile.banner,
              crafts: crafts ?? profile.crafts,
            );
          },
    );
    addTearDown(harness.container.dispose);
    final subscription = harness.container.listen(harness.provider, (_, _) {});
    addTearDown(subscription.close);
    await _waitForPrefill(harness);

    final notifier = harness.container.read(harness.provider.notifier)
      ..updateIdentity(
        displayName: 'Alicia',
        pronouns: ' she/they ',
        bio: 'New bio',
      );
    await notifier.saveAndNext();

    expect(sent?.displayName, 'Alicia');
    expect(sent?.pronouns, 'she/they');
    expect(sent?.description, 'New bio');
    expect(sent?.crafts, ['sewing', 'weaving', 'future-craft']);
    expect(sent?.avatar, isNull);
    expect(sent?.clearAvatar, isFalse);
    expect(sent?.banner, isNull);
    expect(sent?.clearBanner, isFalse);
  });

  test('AT-017 crafts-only save sends the preserved full snapshot', () async {
    final profile = _fullProfile();
    _Update? sent;
    final harness = _flowHarness(
      profile,
      onUpdate:
          ({
            displayName,
            pronouns,
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async {
            sent = _Update(
              displayName: displayName,
              pronouns: pronouns,
              description: description,
              crafts: crafts,
              avatar: avatar,
              clearAvatar: clearAvatar,
              banner: banner,
              clearBanner: clearBanner,
            );
            return Profile(
              did: profile.did,
              handle: profile.handle,
              displayName: displayName,
              pronouns: pronouns,
              description: description,
              avatar: profile.avatar,
              banner: profile.banner,
              crafts: crafts ?? profile.crafts,
            );
          },
    );
    addTearDown(harness.container.dispose);
    final subscription = harness.container.listen(harness.provider, (_, _) {});
    addTearDown(subscription.close);
    await _waitForPrefill(harness);

    final notifier = harness.container.read(harness.provider.notifier)
      ..next()
      ..toggleCraft('sewing')
      ..toggleCraft('quilting');
    await notifier.saveAndNext();

    expect(sent?.displayName, 'Alice');
    expect(sent?.pronouns, 'they/them');
    expect(sent?.description, 'Bio');
    expect(sent?.crafts, ['quilting', 'weaving', 'future-craft']);
    expect(sent?.avatar, isNull);
    expect(sent?.clearAvatar, isFalse);
    expect(sent?.banner, isNull);
    expect(sent?.clearBanner, isFalse);
  });

  test(
    'ambiguous save retries one key and installs the compound overlay',
    () async {
      final profile = _fullProfile();
      var calls = 0;
      late FakeProfileRepository repository;
      final harness = _flowHarness(
        profile,
        extraOverrides: [
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
        onRepository: (value) => repository = value,
        onUpdate:
            ({
              displayName,
              pronouns,
              description,
              crafts,
              avatar,
              clearAvatar = false,
              banner,
              clearBanner = false,
            }) async {
              calls++;
              if (calls == 1) {
                throw const PdsMutationAmbiguousException(
                  retryAfterSeconds: 1,
                );
              }
              return profile.copyWith(
                displayName: displayName,
                pronouns: pronouns,
                description: description,
                crafts: crafts,
              );
            },
      );
      addTearDown(harness.container.dispose);
      final subscription = harness.container.listen(
        harness.provider,
        (_, _) {},
      );
      addTearDown(subscription.close);
      await _waitForPrefill(harness);

      await harness.container.read(harness.provider.notifier).saveAndNext();

      expect(repository.updateOperationKeys, hasLength(2));
      expect(repository.updateOperationKeys.toSet(), hasLength(1));
      expect(
        isCanonicalPdsMutationOperationKey(
          repository.updateOperationKeys.first,
        ),
        isTrue,
      );
      expect(
        harness.container
            .read(pdsRecordOperationControllerProvider)
            .activeOverlays,
        hasLength(1),
      );
    },
  );

  test('step payloads preserve fields owned by the other step', () {
    final profile = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      displayName: 'Alice',
      pronouns: 'they/them',
      description: 'Bio',
      avatar: 'https://example/avatar',
      banner: 'https://example/banner',
      crafts: const ['sewing', 'future-craft'],
    );
    final identityState = OnboardingFlowState.fromProfile(profile).copyWith(
      identity: const OnboardingIdentityDraft(
        displayName: 'Alicia',
        pronouns: '  she/her  ',
        bio: 'New bio',
      ),
      selectedCraftIds: const {'quilting'},
    );

    final identityPayload = OnboardingProfilePayload.fromState(identityState);
    expect(identityPayload.displayName, 'Alicia');
    expect(identityPayload.pronouns, 'she/her');
    expect(identityPayload.description, 'New bio');
    expect(identityPayload.crafts, ['sewing', 'future-craft']);

    final craftsPayload = OnboardingProfilePayload.fromState(
      identityState.copyWith(step: OnboardingStep.crafts),
    );
    expect(craftsPayload.displayName, 'Alice');
    expect(craftsPayload.pronouns, 'they/them');
    expect(craftsPayload.description, 'Bio');
    expect(craftsPayload.crafts, ['quilting', 'future-craft']);
    expect(craftsPayload.clearAvatar, isFalse);
    expect(craftsPayload.clearBanner, isFalse);
    expect(craftsPayload.avatar, isNull);
  });

  test('an empty identity-step value clears existing pronouns', () {
    final state = OnboardingFlowState.fromProfile(_fullProfile()).copyWith(
      identity: const OnboardingIdentityDraft(
        displayName: 'Alice',
        pronouns: '   ',
        bio: 'Bio',
      ),
    );

    final payload = OnboardingProfilePayload.fromState(state);

    expect(payload.pronouns, isNull);
    expect(payload.displayName, 'Alice');
    expect(payload.description, 'Bio');
    expect(payload.crafts, ['sewing', 'weaving', 'future-craft']);
  });
}

({
  ProviderContainer container,
  OnboardingFlowProvider provider,
})
_flowHarness(
  Profile profile, {
  required Future<Profile> Function({
    String? displayName,
    String? pronouns,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar,
    UploadedBlob? banner,
    bool clearBanner,
  })
  onUpdate,
  List<dynamic> extraOverrides = const [],
  void Function(FakeProfileRepository repository)? onRepository,
}) {
  final registry = SessionRegistry.empty().upsertAndActivate(
    token: 'token',
    did: profile.did,
    handle: profile.handle,
  );
  final repository = FakeProfileRepository(
    onFetchMe: () async => profile,
    onUpdateMe: onUpdate,
  );
  onRepository?.call(repository);
  final container = ProviderContainer.test(
    overrides: List.from([
      secureSessionRegistryStorageProvider.overrideWithValue(
        _Storage(registry),
      ),
      accountProfileRepositoryProvider.overrideWith(
        (ref, lease) async => repository,
      ),
      ...extraOverrides,
    ]),
  );
  return (
    container: container,
    provider: onboardingFlowProvider(registry.activeLease!),
  );
}

Profile _fullProfile() => Profile(
  did: 'did:plc:alice',
  handle: 'alice.test',
  displayName: 'Alice',
  pronouns: 'they/them',
  description: 'Bio',
  avatar: 'https://example/avatar',
  banner: 'https://example/banner',
  crafts: const ['sewing', 'weaving', 'future-craft'],
);

Future<void> _waitForPrefill(
  ({ProviderContainer container, OnboardingFlowProvider provider}) harness,
) async {
  final prefilled = Completer<void>();
  final subscription = harness.container.listen(harness.provider, (_, next) {
    if ((next.value?.baseline.crafts.isNotEmpty ?? false) &&
        !prefilled.isCompleted) {
      prefilled.complete();
    }
  }, fireImmediately: true);
  try {
    await harness.container.read(harness.provider.future);
    await prefilled.future.timeout(const Duration(seconds: 1));
  } finally {
    subscription.close();
  }
}
