import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/onboarding/data/onboarding_profile_payload.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
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
    required this.description,
    required this.crafts,
    required this.avatar,
    required this.clearAvatar,
    required this.banner,
    required this.clearBanner,
  });

  final String? displayName;
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
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async {
            sent = _Update(
              displayName: displayName,
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
      ..updateIdentity(displayName: 'Alicia', bio: 'New bio');
    await notifier.saveAndNext();

    expect(sent?.displayName, 'Alicia');
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
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async {
            sent = _Update(
              displayName: displayName,
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
    expect(sent?.description, 'Bio');
    expect(sent?.crafts, ['quilting', 'weaving', 'future-craft']);
    expect(sent?.avatar, isNull);
    expect(sent?.clearAvatar, isFalse);
    expect(sent?.banner, isNull);
    expect(sent?.clearBanner, isFalse);
  });

  test('step payloads preserve fields owned by the other step', () {
    final profile = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      displayName: 'Alice',
      description: 'Bio',
      avatar: 'https://example/avatar',
      banner: 'https://example/banner',
      crafts: const ['sewing', 'future-craft'],
    );
    final identityState = OnboardingFlowState.fromProfile(profile).copyWith(
      identity: const OnboardingIdentityDraft(
        displayName: 'Alicia',
        bio: 'New bio',
      ),
      selectedCraftIds: const {'quilting'},
    );

    final identityPayload = OnboardingProfilePayload.fromState(identityState);
    expect(identityPayload.displayName, 'Alicia');
    expect(identityPayload.description, 'New bio');
    expect(identityPayload.crafts, ['sewing', 'future-craft']);

    final craftsPayload = OnboardingProfilePayload.fromState(
      identityState.copyWith(step: OnboardingStep.crafts),
    );
    expect(craftsPayload.displayName, 'Alice');
    expect(craftsPayload.description, 'Bio');
    expect(craftsPayload.crafts, ['quilting', 'future-craft']);
    expect(craftsPayload.clearAvatar, isFalse);
    expect(craftsPayload.clearBanner, isFalse);
    expect(craftsPayload.avatar, isNull);
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
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar,
    UploadedBlob? banner,
    bool clearBanner,
  })
  onUpdate,
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
  final container = ProviderContainer.test(
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(
        _Storage(registry),
      ),
      accountProfileRepositoryProvider.overrideWith(
        (ref, lease) async => repository,
      ),
    ],
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
