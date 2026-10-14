import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_identity_cache_invalidator.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/save_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image_picker/image_picker.dart';

import '../fakes/fake_profile_repository.dart';

final class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  for (final onboarding in [true, false]) {
    test(
      '${onboarding ? 'onboarding' : 'edit'} overlays stale reads and '
      'refreshes images after closing',
      () async {
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
        final lease = registry.activeLease!;
        final storage = _Storage(registry);
        final scheduled = <Duration, void Function()>{};
        final controller = PdsRecordOperationController(
          schedule: (delay, callback) => scheduled[delay] = callback,
        );
        var authoritative = Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          displayName: 'Before',
          avatar: _url('old'),
          banner: _url('old-banner'),
          crafts: const ['sewing'],
          followerCount: 12,
          customisation: const ProfileCustomisation(
            colour: 'coral',
          ),
        );
        var reads = 0;
        var invalidations = 0;
        Completer<Profile>? blockedRead;
        final repository = FakeProfileRepository(
          onFetchMe: () async {
            reads++;
            return blockedRead == null
                ? authoritative
                : await blockedRead.future;
          },
          onUpdateMe:
              ({
                displayName,
                pronouns,
                description,
                crafts,
                avatar,
                clearAvatar = false,
                banner,
                clearBanner = false,
              }) async => Profile(
                did: 'did:plc:alice',
                handle: 'alice.test',
                displayName: displayName,
                pronouns: pronouns,
                description: description,
                crafts: crafts ?? [],
              ),
        );
        final container = ProviderContainer.test(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(storage),
            profileIdentityCacheInvalidatorProvider.overrideWithValue(
              () => invalidations++,
            ),
            profileRepositoryProvider.overrideWithValue(repository),
            accountProfileImagePickerProvider.overrideWith(
              (ref, lease) async => _Picker(),
            ),
            accountProfileRepositoryProvider.overrideWith(
              (ref, lease) async => repository,
            ),
            pdsRecordOperationControllerProvider.overrideWithValue(controller),
          ],
        );
        addTearDown(container.dispose);
        await container.read(sessionRegistryProvider.future);
        final profileProvider = userProfileProvider(lease.session.account.did);
        final profileSubscription = container.listen(
          profileProvider,
          (_, _) {},
        );
        addTearDown(profileSubscription.close);
        await container.read(profileProvider.future);
        final flowProvider = onboardingFlowProvider(lease);
        if (onboarding) {
          final flowSubscription = container.listen(flowProvider, (_, _) {});
          await container.read(flowProvider.future);
          await Future<void>.delayed(Duration.zero);
          final notifier = container.read(flowProvider.notifier)
            ..updateIdentity(
              displayName: 'After',
              pronouns: 'they/them',
              bio: 'New bio',
            );
          await notifier.pickAvatar(ImageSource.gallery);
          await notifier.saveAndNext();
          expect(container.read(flowProvider).requireValue.step.index, 1);
          expect(
            container.read(flowProvider).requireValue.baseline.avatar,
            authoritative.avatar,
          );
          expect(
            container.read(flowProvider).requireValue.baseline.banner,
            authoritative.banner,
          );
          // Persisting identity must preserve the current onboarding step.
          final identitySubscription = container.listen(
            activeAccountIdentityProvider,
            (_, _) {},
          );
          await container.read(activeAccountIdentityProvider.future);
          expect(container.read(flowProvider).requireValue.step.index, 1);
          identitySubscription.close();
          notifier.toggleCraft('weaving');
          await notifier.saveAndNext();
          expect(container.read(flowProvider).requireValue.step.index, 2);
          expect(
            (controller.activeOverlays.single.optimisticValue!
                    as AcceptedPersonalProfile)
                .avatarCid,
            'new',
          );
          flowSubscription.close();
          container.invalidate(flowProvider);
        } else {
          final saveSubscription = container.listen(
            saveProfileProvider,
            (_, _) {},
          );
          await container.read(saveProfileProvider.future);
          blockedRead = Completer<Profile>();
          await container
              .read(saveProfileProvider.notifier)
              .save(
                currentProfile: authoritative,
                displayName: 'After',
                pronouns: 'they/them',
                description: 'New bio',
                crafts: const ['weaving'],
                avatar: _blob('new'),
                banner: _blob('new-banner'),
              );
          // Accepted writes invalidate ordinary reads; the overlay composes
          // saved fields with their response instead of mutating the cache.
          expect(container.read(profileProvider).isLoading, isTrue);
          expect(container.read(profileProvider).value?.displayName, 'Before');
          expect(container.read(profileProvider).value?.followerCount, 12);
          expect(
            container.read(profileProvider).value?.customisation,
            authoritative.customisation,
          );
          blockedRead.complete(authoritative);
          blockedRead = null;
          saveSubscription.close();
          container.invalidate(saveProfileProvider);
        }
        await container.read(profileProvider.future);
        final accepted = container.read(profileProvider).requireValue;
        expect(accepted.displayName, 'After');
        expect(accepted.pronouns, 'they/them');
        expect(accepted.description, 'New bio');
        expect(accepted.followerCount, 12);
        expect(accepted.customisation, authoritative.customisation);
        expect(accepted.avatar, authoritative.avatar);
        expect(accepted.banner, authoritative.banner);
        final previousReads = reads;
        final previousInvalidations = invalidations;
        authoritative = authoritative.copyWith(
          displayName: 'After',
          pronouns: 'they/them',
          description: 'New bio',
          crafts: accepted.crafts,
          avatar: _url('new'),
          banner: onboarding ? authoritative.banner : _url('new-banner'),
        );
        scheduled[const Duration(seconds: 5)]!();
        await container.read(profileProvider.future);
        expect(reads, greaterThan(previousReads));
        expect(invalidations, previousInvalidations + 1);
        expect(
          container.read(profileProvider).requireValue.avatar,
          authoritative.avatar,
        );
        expect(
          container.read(profileProvider).requireValue.banner,
          authoritative.banner,
        );
        expect(controller.activeOverlays, isEmpty);
        final identitySubscription = container.listen(
          activeAccountIdentityProvider,
          (_, _) {},
        );
        addTearDown(identitySubscription.close);
        await container.read(activeAccountIdentityProvider.future);
        expect(storage.value.sessions.values.single.cachedDisplayName, 'After');
        expect(
          storage.value.sessions.values.single.cachedAvatarUrl,
          authoritative.avatar,
        );
        await container
            .read(sessionRegistryProvider.notifier)
            .upsertAndActivate(
              token: 'bob-token',
              did: 'did:plc:bob',
              handle: 'bob.test',
            );
        final afterSwitch = invalidations;
        scheduled[const Duration(seconds: 20)]!();
        expect(invalidations, afterSwitch);
      },
    );
  }

  test('omitted images preserve read-owned URLs unless explicitly cleared', () {
    final previous = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const ['sewing'],
      avatar: _url('existing'),
      banner: _url('existing-banner'),
    );
    final response = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const ['knitting'],
    );
    final accepted = AcceptedPersonalProfile(response);
    final visible = accepted.applyTo(previous);
    expect(visible.avatar, previous.avatar);
    expect(visible.banner, previous.banner);
    expect(accepted.agrees(personalProfileProjection(visible)), isTrue);
    final removed = AcceptedPersonalProfile(response, clearAvatar: true);
    expect(removed.applyTo(previous).avatar, isNull);
    expect(removed.applyTo(previous).banner, previous.banner);
    expect(removed.agrees(personalProfileProjection(visible)), isFalse);
    expect(
      removed.agrees(personalProfileProjection(visible.copyWith(avatar: null))),
      isTrue,
    );
    // Later server visibility changes still take effect for untouched images.
    expect(accepted.applyTo(previous.copyWith(avatar: null)).avatar, isNull);
  });

  test('image confirmation waits for CID and preserves other fields', () {
    final profile = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      displayName: 'Saved',
      crafts: const ['sewing'],
      avatar: _url('old'),
      banner: _url('old-banner'),
    );
    final accepted = AcceptedPersonalProfile(
      profile,
      avatarCid: 'new',
      bannerCid: 'new-banner',
    );
    expect(accepted.agrees(personalProfileProjection(profile)), isFalse);
    final partial = profile.copyWith(avatar: _url('new'), followerCount: 9);
    expect(accepted.agrees(personalProfileProjection(partial)), isFalse);
    expect(accepted.applyTo(partial).avatar, _url('new'));
    expect(accepted.applyTo(partial).followerCount, 9);
    expect(
      accepted.agrees(
        personalProfileProjection(partial.copyWith(banner: _url('new-banner'))),
      ),
      isTrue,
    );
    expect(
      AcceptedPersonalProfile(
        profile.copyWith(avatar: null),
        clearAvatar: true,
      ).applyTo(profile).avatar,
      isNull,
    );
  });
}

String _url(String cid) =>
    'https://cdn.bsky.app/img/avatar/plain/did:plc:alice/$cid@jpeg';
UploadedBlob _blob(String cid) => UploadedBlob(
  type: 'blob',
  ref: UploadedBlobRef(link: cid),
  mimeType: 'image/jpeg',
  size: 10,
);

class _Picker extends Fake implements ProfileImagePicker {
  @override
  Future<ProfileImagePickResult?> pickAndUpload({
    required ImageSource source,
    required void Function(Uint8List bytes) onPreviewReady,
  }) async => ProfileImagePickResult(
    previewBytes: Uint8List.fromList([1]),
    uploaded: UploadedImageBlob(
      blob: _blob('new'),
      cid: 'new',
      mime: 'image/jpeg',
      size: 10,
    ),
  );
}
