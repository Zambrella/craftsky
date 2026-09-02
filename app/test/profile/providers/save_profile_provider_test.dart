import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/save_profile_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-005 ordinary-only save sends no business request', () async {
    final profileRepository = _ProfileRepository();
    final businessRepository = _BusinessRepository();
    final container = ProviderContainer.test(
      overrides: [
        profileRepositoryProvider.overrideWithValue(profileRepository),
        businessRepositoryProvider.overrideWithValue(businessRepository),
      ],
    );
    final subscription = container.listen(saveProfileProvider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(saveProfileProvider.future);

    final result = await container
        .read(saveProfileProvider.notifier)
        .save(
          currentProfile: _profile,
          displayName: 'Renamed',
          description: 'Bio',
          crafts: const ['sewing'],
        );

    expect(profileRepository.updateCalls, 1);
    expect(businessRepository.putCalls, 0);
    expect(result?.isFullSuccess, isTrue);
  });

  test('IT-005 business-only save sends complete declaration only', () async {
    final profileRepository = _ProfileRepository();
    final businessRepository = _BusinessRepository();
    final container = _container(profileRepository, businessRepository);
    final subscription = container.listen(saveProfileProvider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(saveProfileProvider.future);
    final draft = BusinessDeclarationDraft.fromProfile(
      _businessProfile,
    ).copyWith(tagline: 'Updated');

    final result = await container
        .read(saveProfileProvider.notifier)
        .save(
          currentProfile: _profile.copyWith(
            accountType: AccountType.business,
            business: _businessProfile,
          ),
          ordinaryChanged: false,
          businessChanged: true,
          businessDraft: draft,
        );

    expect(profileRepository.updateCalls, 0);
    expect(businessRepository.putCalls, 1);
    expect(businessRepository.expectedCids, [_businessProfile.cid]);
    expect(businessRepository.bodies.single['products'], hasLength(1));
    expect(result?.isFullSuccess, isTrue);
  });

  test('IT-005 partial save retries only the failed ordinary record', () async {
    final profileRepository = _ProfileRepository()
      ..updateError = Exception('ordinary failed');
    final businessRepository = _BusinessRepository();
    final container = _container(profileRepository, businessRepository);
    final subscription = container.listen(saveProfileProvider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(saveProfileProvider.future);
    final current = _profile.copyWith(
      accountType: AccountType.business,
      business: _businessProfile,
    );
    final draft = BusinessDeclarationDraft.fromProfile(
      _businessProfile,
    ).copyWith(tagline: 'Updated');

    final partial = await container
        .read(saveProfileProvider.notifier)
        .save(
          currentProfile: current,
          businessChanged: true,
          businessDraft: draft,
          displayName: 'Renamed',
          description: 'Bio',
          crafts: const ['sewing'],
        );

    expect(partial?.retryOrdinary, isTrue);
    expect(partial?.retryBusiness, isFalse);
    expect(profileRepository.updateCalls, 1);
    expect(businessRepository.putCalls, 1);

    profileRepository.updateError = null;
    final retry = await container
        .read(saveProfileProvider.notifier)
        .save(
          currentProfile: current.copyWith(
            business: partial?.business.value as BusinessProfile?,
          ),
          ordinaryChanged: partial!.retryOrdinary,
          businessChanged: partial.retryBusiness,
          businessDraft: draft,
          displayName: 'Renamed',
          description: 'Bio',
          crafts: const ['sewing'],
        );

    expect(retry?.isFullSuccess, isTrue);
    expect(profileRepository.updateCalls, 2);
    expect(businessRepository.putCalls, 1);
  });

  test('IT-005 inverse partial save retries only business', () async {
    final profileRepository = _ProfileRepository();
    final businessRepository = _BusinessRepository()
      ..putError = Exception('business failed');
    final container = _container(profileRepository, businessRepository);
    final subscription = container.listen(saveProfileProvider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(saveProfileProvider.future);
    final draft = BusinessDeclarationDraft.fromProfile(
      _businessProfile,
    ).copyWith(tagline: 'Updated');

    final partial = await container
        .read(saveProfileProvider.notifier)
        .save(
          currentProfile: _profile,
          businessChanged: true,
          businessDraft: draft,
          displayName: 'Renamed',
          crafts: const ['sewing'],
        );

    expect(partial?.retryOrdinary, isFalse);
    expect(partial?.retryBusiness, isTrue);
    businessRepository.putError = null;
    await container
        .read(saveProfileProvider.notifier)
        .save(
          currentProfile: partial?.ordinary.value as Profile?,
          ordinaryChanged: partial!.retryOrdinary,
          businessChanged: partial.retryBusiness,
          businessDraft: draft,
        );

    expect(profileRepository.updateCalls, 1);
    expect(businessRepository.putCalls, 2);
  });
}

ProviderContainer _container(
  _ProfileRepository profileRepository,
  _BusinessRepository businessRepository,
) => ProviderContainer.test(
  overrides: [
    profileRepositoryProvider.overrideWithValue(profileRepository),
    businessRepositoryProvider.overrideWithValue(businessRepository),
  ],
);

final _profile = Profile(
  did: 'did:plc:alice',
  handle: 'alice.test',
  displayName: 'Alice',
  crafts: const ['sewing'],
);

final _businessProfile = BusinessProfile(
  cid: 'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  businessTypes: const [BusinessOpenValue(value: 'teacher', known: true)],
  products: const [
    BusinessProductView(title: 'Pattern', uri: 'https://example.com/pattern'),
  ],
);

final class _ProfileRepository extends Fake implements ProfileRepository {
  int updateCalls = 0;
  Exception? updateError;

  @override
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) async {
    updateCalls++;
    if (updateError case final error?) throw error;
    return _profile.copyWith(
      displayName: displayName,
      description: description,
      crafts: crafts,
    );
  }
}

final class _BusinessRepository extends Fake implements BusinessRepository {
  int putCalls = 0;
  Exception? putError;
  final bodies = <Map<String, dynamic>>[];
  final expectedCids = <Cid?>[];

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) async {
    putCalls++;
    bodies.add(body);
    expectedCids.add(expectedCid);
    if (putError case final error?) throw error;
    return RecordMutationResult(
      cid: 'bafyreifbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    );
  }
}
