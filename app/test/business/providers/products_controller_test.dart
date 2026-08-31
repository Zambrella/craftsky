import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-007 saves one complete declaration with current CID and order',
    () async {
      final repository = _Repository();
      final container = _container(repository, _profile('bafy-current'));
      addTearDown(container.dispose);
      final controller = container.read(productsControllerProvider.notifier);
      final initial = await container.read(productsControllerProvider.future);
      final reordered = initial.products.reversed.toList();

      controller.replaceProducts(reordered);
      expect(await controller.save(), isTrue);

      expect(repository.expectedCids.single.toString(), 'bafy-current');
      expect(repository.bodies.single, {
        'businessTypes': ['dyer'],
        'offerings': ['yarn'],
        'tagline': 'Details survive',
        'products': [
          isA<Map<String, dynamic>>().having(
            (value) => value['title'],
            'title',
            'Two',
          ),
          isA<Map<String, dynamic>>().having(
            (value) => value['title'],
            'title',
            'One',
          ),
        ],
      });
      expect(
        container
            .read(productsControllerProvider)
            .requireValue
            .products
            .map((p) => p.title),
        ['Two', 'One'],
      );
    },
  );

  test('IT-006 conflict requires complete reload before retry', () async {
    final repository = _Repository()
      ..error = const ApiBadRequest('pds_record_conflict');
    final current = _profile('bafy-current');
    final reloaded = _profile('bafy-new', tagline: 'Changed elsewhere');
    var reloads = 0;
    final container = _container(
      repository,
      current,
      loader: () async {
        reloads++;
        return reloaded;
      },
    );
    addTearDown(container.dispose);
    final controller = container.read(productsControllerProvider.notifier);
    await container.read(productsControllerProvider.future);

    expect(await controller.save(), isFalse);
    expect(
      container.read(productsControllerProvider).requireValue.status,
      ProductsStatus.conflict,
    );
    expect(await controller.save(), isFalse);
    expect(repository.bodies, hasLength(1));

    await controller.reloadAfterConflict();
    final state = container.read(productsControllerProvider).requireValue;
    expect(reloads, 1);
    expect(state.declaration.expectedCid.toString(), 'bafy-new');
    expect(state.declaration.tagline, 'Changed elsewhere');
    expect(state.products.map((product) => product.title), ['One', 'Two']);

    repository.error = null;
    expect(await controller.save(), isTrue);
    expect(repository.expectedCids.last.toString(), 'bafy-new');
  });

  test(
    'IT-007 upload failure or cancellation leaves saved image untouched',
    () async {
      final container = _container(_Repository(), _profile('bafy-current'));
      addTearDown(container.dispose);
      final initial = await container.read(productsControllerProvider.future);
      final controller = container.read(productsControllerProvider.notifier);
      final savedImage = initial.products.first.image;

      controller
        ..imageUploadFailed(initial.products.first.id)
        ..imageUploadCancelled(initial.products.first.id);

      expect(
        container
            .read(productsControllerProvider)
            .requireValue
            .products
            .first
            .image,
        same(savedImage),
      );
    },
  );

  test(
    'IT-007 accepted replacement retains its local preview without adding '
    'display fields to the mutation',
    () async {
      final repository = _Repository();
      final container = _container(repository, _profile('bafy-current'));
      addTearDown(container.dispose);
      final initial = await container.read(productsControllerProvider.future);
      final product = initial.products.first;
      final previewBytes = Uint8List.fromList([1, 2, 3, 4]);
      container.read(productsControllerProvider.notifier).replaceProducts([
        ProductDraft(
          id: product.id,
          title: product.title,
          destination: product.destination,
          image: UploadedBusinessImageDraft(
            cid: 'bafy-replacement',
            mime: 'image/png',
            size: 4,
            alt: 'Replacement image',
            localPreviewBytes: previewBytes,
          ),
          amount: product.amount,
          currency: product.currency,
        ),
      ]);

      expect(
        await container.read(productsControllerProvider.notifier).save(),
        isTrue,
      );

      final accepted =
          (container
                  .read(businessProjectionOverlayProvider)
                  .values
                  .single
                  .acceptedView
              as BusinessProfile?)!;
      expect(accepted.products.single.image?.previewBytes, same(previewBytes));
      expect(repository.bodies.single['products'], [
        isA<Map<String, dynamic>>().having(
          (product) => product['image'],
          'image',
          {
            'image': {
              r'$type': 'blob',
              'ref': {r'$link': 'bafy-replacement'},
              'mimeType': 'image/png',
              'size': 4,
            },
            'alt': 'Replacement image',
          },
        ),
      ]);
    },
  );

  test(
    'IT-013 Products save retains accepted products through a lagging '
    'profile read',
    () async {
      final oldProfile = _profile('bafy-current');
      final laggingRead = Completer<Profile>();
      final currentRead = Completer<Profile>();
      final profileRepository = _ProfileRepository(
        oldProfile,
        laggingRead,
        currentRead,
      );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'token-owner',
                did: 'did:plc:owner',
                handle: 'owner.test',
              ),
            ),
          ),
          profileRepositoryProvider.overrideWithValue(profileRepository),
          businessRepositoryProvider.overrideWithValue(_Repository()),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      final subscription = container.listen(
        userProfileProvider('owner.test'),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(userProfileProvider('owner.test').future);
      final initial = await container.read(productsControllerProvider.future);
      final first = initial.products.first;
      final previewBytes = Uint8List.fromList([5, 6, 7]);
      container.read(productsControllerProvider.notifier).replaceProducts([
        ProductDraft(
          id: first.id,
          title: 'Accepted product',
          destination: first.destination,
          image: UploadedBusinessImageDraft(
            cid: 'bafy-replacement',
            mime: 'image/png',
            size: 3,
            alt: 'Accepted replacement',
            localPreviewBytes: previewBytes,
          ),
          amount: first.amount,
          currency: first.currency,
        ),
      ]);
      container.invalidate(userProfileProvider('owner.test'));
      final staleProfileRead = container.read(
        userProfileProvider('owner.test').future,
      );
      await Future<void>.delayed(Duration.zero);
      expect(profileRepository.fetches, 2);

      expect(
        await container.read(productsControllerProvider.notifier).save(),
        isTrue,
      );
      await Future<void>.delayed(Duration.zero);
      expect(profileRepository.fetches, 3);

      currentRead.complete(oldProfile);
      final reconciled = await container.read(
        userProfileProvider('owner.test').future,
      );
      expect(
        reconciled.business?.products.map((product) => product.title),
        ['Accepted product'],
      );
      expect(
        reconciled.business?.products.single.image?.previewBytes,
        same(previewBytes),
      );

      laggingRead.complete(oldProfile.copyWith(business: null));
      expect(
        (await staleProfileRead).business?.products.single.title,
        'Accepted product',
      );
      expect(
        container
            .read(userProfileProvider('owner.test'))
            .requireValue
            .business
            ?.products
            .single
            .title,
        'Accepted product',
      );
    },
  );
}

ProviderContainer _container(
  _Repository repository,
  Profile profile, {
  Future<Profile> Function()? loader,
}) => ProviderContainer(
  overrides: [
    activeAccountIdentityProvider.overrideWith(
      (_) async => ActiveAccountIdentity(
        lease: AccountSessionLease(
          account: AccountKey('did:plc:owner'),
          sessionGeneration: 1,
        ),
        profile: profile,
      ),
    ),
    businessRepositoryProvider.overrideWithValue(repository),
    productsProfileLoaderProvider.overrideWithValue(
      loader ?? () async => profile,
    ),
  ],
);

Profile _profile(String cid, {String tagline = 'Details survive'}) => Profile(
  did: 'did:plc:owner',
  handle: 'owner.test',
  crafts: const [],
  accountType: AccountType.business,
  business: BusinessProfile(
    cid: cid,
    businessTypes: const [BusinessOpenValue(value: 'dyer', known: true)],
    offerings: const [BusinessOpenValue(value: 'yarn', known: true)],
    tagline: tagline,
    products: [_product('One', 'one'), _product('Two', 'two')],
  ),
);

BusinessProductView _product(String title, String path) => BusinessProductView(
  title: title,
  uri: 'https://shop.example/$path',
  image: BusinessImageView(
    cid: 'bafy-$path',
    mime: 'image/jpeg',
    size: 10,
    alt: title,
    thumb: 'https://cdn.example/$path/thumb',
    fullsize: 'https://cdn.example/$path/full',
  ),
);

final class _Repository extends Fake implements BusinessRepository {
  final bodies = <Map<String, dynamic>>[];
  final expectedCids = <Cid?>[];
  Exception? error;

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) async {
    bodies.add(body);
    expectedCids.add(expectedCid);
    if (error case final value?) throw value;
    return RecordMutationResult(cid: 'bafy-accepted');
  }
}

final class _ProfileRepository extends Fake implements ProfileRepository {
  _ProfileRepository(this.initial, this.laggingRead, [this.currentRead]);

  final Profile initial;
  final Completer<Profile> laggingRead;
  final Completer<Profile>? currentRead;
  int fetches = 0;

  @override
  Future<Profile> fetch(String handleOrDid) {
    fetches++;
    return switch (fetches) {
      1 => Future.value(initial),
      2 => laggingRead.future,
      _ => currentRead?.future ?? laggingRead.future,
    };
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}
