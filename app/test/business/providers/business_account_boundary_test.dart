import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/account_type_controller.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_event_mutation_controller.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/providers/report_business_event_provider.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/save_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('AT-011 account boundary advances overlays before invalidation', () {
    final container = ProviderContainer.test();
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final lease = AccountSessionLease(
      account: AccountKey('did:plc:alice'),
      sessionGeneration: 1,
    );
    final key = BusinessProjectionKey.event(
      lease.account,
      Did.parse('did:plc:alice'),
      RecordKey.parse('alice-event'),
    );
    final pendingGeneration = controller.beginMutation(key, lease);
    final staleRead = controller.captureRead(lease);
    controller
      ..acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: pendingGeneration,
        preWriteCid: Cid.parse('bafy-alice-before'),
        acceptedCid: Cid.parse('bafy-alice-accepted'),
        acceptedView: 'Alice accepted event',
      )
      ..advanceAccountBoundary();

    expect(container.read(businessProjectionOverlayProvider), isEmpty);
    expect(controller.isReadCurrent(staleRead), isFalse);
    expect(
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: pendingGeneration,
        preWriteCid: Cid.parse('bafy-alice-before'),
        acceptedCid: Cid.parse('bafy-alice-late'),
        acceptedView: 'Alice late event',
      ),
      isFalse,
    );
  });

  test(
    'IT-010 late Alice profile list detail values and errors never publish '
    'as Bob',
    () async {
      final alicePublicMore = Completer<BusinessEventPage>();
      final aliceOwnerMore = Completer<BusinessEventPage>();
      final aliceDetail = Completer<BusinessEvent>();
      final aliceProfile = Completer<Profile>();
      final repository = _ReadBusinessRepository(
        alicePublicMore: alicePublicMore,
        aliceOwnerMore: aliceOwnerMore,
        aliceDetail: aliceDetail,
      );
      final profileRepository = _ReadProfileRepository(aliceProfile);
      final storage = _RegistryStorage(_registry());
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          businessRepositoryProvider.overrideWithValue(repository),
          profileRepositoryProvider.overrideWithValue(profileRepository),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final aliceTarget = ProfileBusinessEventsTarget(
        account: AccountKey('did:plc:alice'),
        owner: AtIdentifier.parse('did:plc:alice'),
      );
      final aliceDetailTarget = BusinessEventDetailTarget(
        account: AccountKey('did:plc:alice'),
        owner: Did.parse('did:plc:alice'),
        rkey: RecordKey.parse('alice-detail'),
      );
      final subscriptions = <ProviderSubscription<Object?>>[
        container.listen<AsyncValue<BusinessEventListState>>(
          profileBusinessEventsProvider(aliceTarget),
          (_, _) {},
          fireImmediately: true,
        ),
        container.listen<AsyncValue<BusinessEventListState>>(
          ownerBusinessEventsProvider(OwnerEventFilter.upcoming),
          (_, _) {},
          fireImmediately: true,
        ),
        container.listen<AsyncValue<BusinessEventDetailState>>(
          businessEventDetailProvider(aliceDetailTarget),
          (_, _) {},
          fireImmediately: true,
        ),
        container.listen<AsyncValue<Profile>>(
          userProfileProvider('alice.test'),
          (_, _) {},
          fireImmediately: true,
        ),
      ];
      addTearDown(() {
        for (final subscription in subscriptions) {
          subscription.close();
        }
      });

      final alicePublic = await container.read(
        profileBusinessEventsProvider(aliceTarget).future,
      );
      final aliceOwner = await container.read(
        ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
      );
      expect(alicePublic.cursor, 'alice-public-cursor');
      expect(aliceOwner.cursor, 'alice-owner-cursor');
      final latePublic = container
          .read(profileBusinessEventsProvider(aliceTarget).notifier)
          .loadMore();
      final lateOwner = container
          .read(
            ownerBusinessEventsProvider(OwnerEventFilter.upcoming).notifier,
          )
          .loadMore();
      await Future<void>.delayed(Duration.zero);

      await _switch(container, AccountKey('did:plc:bob'));
      final bobTarget = ProfileBusinessEventsTarget(
        account: AccountKey('did:plc:bob'),
        owner: AtIdentifier.parse('did:plc:bob'),
      );
      final bobDetailTarget = BusinessEventDetailTarget(
        account: AccountKey('did:plc:bob'),
        owner: Did.parse('did:plc:bob'),
        rkey: RecordKey.parse('bob-detail'),
      );
      subscriptions
        ..add(
          container.listen<AsyncValue<BusinessEventListState>>(
            profileBusinessEventsProvider(bobTarget),
            (_, _) {},
            fireImmediately: true,
          ),
        )
        ..add(
          container.listen<AsyncValue<BusinessEventDetailState>>(
            businessEventDetailProvider(bobDetailTarget),
            (_, _) {},
            fireImmediately: true,
          ),
        )
        ..add(
          container.listen<AsyncValue<Profile>>(
            userProfileProvider('bob.test'),
            (_, _) {},
            fireImmediately: true,
          ),
        );
      final bobPublic = await container.read(
        profileBusinessEventsProvider(bobTarget).future,
      );
      final bobOwner = await container.read(
        ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
      );
      final bobDetail = await container.read(
        businessEventDetailProvider(bobDetailTarget).future,
      );
      final bobProfile = await container.read(
        userProfileProvider('bob.test').future,
      );

      alicePublicMore.complete(
        BusinessEventPage(items: [_event('alice-late-public')]),
      );
      aliceOwnerMore.completeError(StateError('alice late owner error'));
      aliceDetail.complete(_event('alice-detail', owner: 'did:plc:alice'));
      aliceProfile.complete(_profile('alice', tagline: 'Alice late profile'));
      await Future.wait([latePublic, lateOwner]);
      await Future<void>.delayed(Duration.zero);

      expect(bobPublic.items.single.name, 'bob-public');
      expect(bobOwner.items.single.name, 'bob-owner');
      expect(
        (bobDetail as BusinessEventDetailAvailable).event.name,
        'bob-detail',
      );
      expect(bobProfile.business?.tagline, 'Bob authoritative profile');
      expect(
        container
            .read(profileBusinessEventsProvider(bobTarget))
            .requireValue
            .items
            .single
            .name,
        'bob-public',
      );

      await _switch(container, AccountKey('did:plc:alice'));
      final authoritativePublic = await container.read(
        profileBusinessEventsProvider(aliceTarget).future,
      );
      final authoritativeOwner = await container.read(
        ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
      );
      final authoritativeDetail = await container.read(
        businessEventDetailProvider(aliceDetailTarget).future,
      );
      final authoritativeProfile = await container.read(
        userProfileProvider('alice.test').future,
      );

      expect(
        authoritativePublic.items.single.name,
        'alice-authoritative-public',
      );
      expect(authoritativeOwner.items.single.name, 'alice-authoritative-owner');
      expect(
        (authoritativeDetail as BusinessEventDetailAvailable).event.name,
        'alice-authoritative-detail',
      );
      expect(
        authoritativeProfile.business?.tagline,
        'Alice authoritative profile',
      );
      expect(repository.publicCursors.last, isNull);
      expect(repository.ownerCursors.last, isNull);
    },
  );

  test(
    'IT-010 Alice CIDs mutations and reports cannot become Bob state',
    () async {
      final repository = _MutationBusinessRepository();
      final storage = _RegistryStorage(_registry());
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          businessRepositoryProvider.overrideWithValue(repository),
          businessTimeZoneServiceProvider.overrideWithValue(
            BusinessTimeZoneService.initialized(),
          ),
          activeAccountIdentityProvider.overrideWith((ref) async {
            final account = ref
                .watch(sessionRegistryProvider)
                .requireValue
                .activeDid!
                .value;
            final name = account.endsWith('alice') ? 'alice' : 'bob';
            return ActiveAccountIdentity(
              lease: ref
                  .read(sessionRegistryProvider)
                  .requireValue
                  .activeLease!
                  .session,
              profile: _profile(
                name,
                tagline: '$name authoritative profile',
                productTitle: '$name authoritative product',
              ),
            );
          }),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final reportProvider = reportBusinessEventProvider(
        AccountKey('did:plc:alice'),
      );
      final subscriptions = <ProviderSubscription<Object?>>[
        container.listen<AsyncValue<AccountType?>>(
          accountTypeControllerProvider,
          (_, _) {},
          fireImmediately: true,
        ),
        container.listen<AsyncValue<ProductsState>>(
          productsControllerProvider,
          (_, _) {},
          fireImmediately: true,
        ),
        container.listen<EventMutationState>(
          businessEventMutationControllerProvider,
          (_, _) {},
          fireImmediately: true,
        ),
        container.listen<AsyncValue<ReportResult?>>(reportProvider, (_, _) {}),
        container.listen<AsyncValue<dynamic>>(saveProfileProvider, (_, _) {}),
      ];
      addTearDown(() {
        for (final subscription in subscriptions) {
          subscription.close();
        }
      });
      final initialProducts = await container.read(
        productsControllerProvider.future,
      );
      final editedProduct = ProductDraft(
        id: initialProducts.products.single.id,
        title: 'Alice unsaved product',
        destination: 'https://alice.example/unsaved',
        image: initialProducts.products.single.image,
      );
      final aliceEvent = _event(
        'alice-mutation',
        owner: 'did:plc:alice',
      );
      final accountTypeOperation = container
          .read(accountTypeControllerProvider.notifier)
          .setAccountType(AccountType.regular);
      final productOperation = container
          .read(productsControllerProvider.notifier)
          .replaceProducts([editedProduct]);
      final eventOperation = container
          .read(businessEventMutationControllerProvider.notifier)
          .update(aliceEvent, _eventDraft('Alice unsaved event'));
      final reportOperation = container
          .read(reportProvider.notifier)
          .submit(
            owner: aliceEvent.did,
            rkey: aliceEvent.rkey,
            submission: const ReportSubmission(reasonType: 'spam'),
          );
      final combinedOperation = container
          .read(saveProfileProvider.notifier)
          .save(
            currentProfile: _profile(
              'alice',
              tagline: 'Alice declaration',
              productTitle: 'Alice saved product',
            ),
            ordinaryChanged: false,
            businessChanged: true,
            businessDraft: BusinessDeclarationDraft.fromProfile(
              _profile(
                'alice',
                tagline: 'Alice unsaved declaration',
                productTitle: 'Alice saved product',
              ).business,
            ),
          );
      await Future<void>.delayed(Duration.zero);

      var homeResets = 0;
      await _switch(
        container,
        AccountKey('did:plc:bob'),
        resetToHome: () async => homeResets++,
      );
      final bobProducts = await container.read(
        productsControllerProvider.future,
      );
      expect(bobProducts.products.single.title, 'bob authoritative product');
      expect(
        bobProducts.declaration.expectedCid.toString(),
        'bafy-bob-profile',
      );
      expect(
        container.read(businessEventMutationControllerProvider).status,
        EventMutationStatus.ready,
      );
      expect(container.read(businessProjectionOverlayProvider), isEmpty);
      expect(
        await container.read(accountTypeControllerProvider.future),
        AccountType.business,
      );
      expect(
        await container.read(
          reportBusinessEventProvider(AccountKey('did:plc:bob')).future,
        ),
        isNull,
      );
      expect(container.read(saveProfileProvider).value, isNull);

      repository
        ..accountType.completeError(StateError('Alice account type failed'))
        ..products.complete(
          RecordMutationResult(cid: 'bafy-alice-products-late'),
        )
        ..event.completeError(StateError('Alice event failed'))
        ..report.complete(
          const ReportResult(reportId: 'alice-report', status: 'accepted'),
        )
        ..combined.complete(
          RecordMutationResult(cid: 'bafy-alice-combined-late'),
        );
      expect(await accountTypeOperation, isFalse);
      expect(await productOperation, isFalse);
      expect(await eventOperation, isFalse);
      await reportOperation;
      expect(await combinedOperation, isNull);
      await Future<void>.delayed(Duration.zero);

      expect(
        container
            .read(productsControllerProvider)
            .requireValue
            .products
            .single
            .title,
        'bob authoritative product',
      );
      expect(
        container.read(businessEventMutationControllerProvider).status,
        EventMutationStatus.ready,
      );
      expect(container.read(businessProjectionOverlayProvider), isEmpty);
      expect(repository.calls.where((call) => call.contains('bob')), isEmpty);
      expect(homeResets, 1);

      await _switch(
        container,
        AccountKey('did:plc:alice'),
        resetToHome: () async => homeResets++,
      );
      final aliceAuthoritative = await container.read(
        productsControllerProvider.future,
      );
      expect(
        aliceAuthoritative.products.single.title,
        'alice authoritative product',
      );
      expect(
        aliceAuthoritative.declaration.expectedCid.toString(),
        'bafy-alice-profile',
      );
      expect(aliceAuthoritative.status, ProductsStatus.ready);
    },
  );

  testWidgets(
    'IT-010 late Alice upload cannot populate or submit the Bob editor',
    (tester) async {
      final upload = Completer<ProfileImagePickResult?>();
      final storage = _RegistryStorage(_registry());
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      ProductDraft? submitted;
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: ProductEditor(
                initial: _productDraft('Alice saved product'),
                pickImage: (_) => upload.future,
                onSave: (value) => submitted = value,
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Replace image'));
      await tester.pump();

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(
            container
                .read(sessionRegistryProvider)
                .requireValue
                .leaseFor(AccountKey('did:plc:bob'))!,
          );
      upload.complete(_uploadedPick('bafy-alice-upload'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Save product'));
      await tester.tap(find.text('Save product'));
      await tester.pump();

      expect(submitted, isNull);
      expect(find.byType(LinearProgressIndicator), findsNothing);
    },
  );
}

Future<void> _switch(
  ProviderContainer container,
  AccountKey account, {
  Future<bool> Function(AccountSessionLease owner)? confirmLeave,
  Future<void> Function()? resetToHome,
}) async {
  final target = container
      .read(sessionRegistryProvider)
      .requireValue
      .leaseFor(account)!;
  final coordinator = AccountActivationCoordinator(
    readRegistry: () => container.read(sessionRegistryProvider).requireValue,
    commitActivation: container.read(sessionRegistryProvider.notifier).activate,
    invalidateAccountState: container.read(accountStateInvalidatorProvider),
    resetToHome: resetToHome ?? () async {},
    confirmLeave: confirmLeave,
  );
  expect(
    await coordinator.activate(target),
    AccountActivationResult.activated,
  );
  await Future<void>.delayed(Duration.zero);
}

SessionRegistry _registry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'token-bob',
      did: 'did:plc:bob',
      handle: 'bob.test',
    )
    .upsertAndActivate(
      token: 'token-alice',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _ReadBusinessRepository extends Fake implements BusinessRepository {
  _ReadBusinessRepository({
    required this.alicePublicMore,
    required this.aliceOwnerMore,
    required this.aliceDetail,
  });

  final Completer<BusinessEventPage> alicePublicMore;
  final Completer<BusinessEventPage> aliceOwnerMore;
  final Completer<BusinessEvent> aliceDetail;
  final publicCursors = <String?>[];
  final ownerCursors = <String?>[];
  var _publicCalls = 0;
  var _ownerCalls = 0;
  var _detailCalls = 0;

  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) {
    publicCursors.add(cursor);
    _publicCalls++;
    if (_publicCalls == 1) {
      return Future.value(
        BusinessEventPage(
          items: [_event('alice-public')],
          cursor: 'alice-public-cursor',
        ),
      );
    }
    if (_publicCalls == 2) return alicePublicMore.future;
    if (owner.toString() == 'did:plc:bob') {
      return Future.value(BusinessEventPage(items: [_event('bob-public')]));
    }
    return Future.value(
      BusinessEventPage(items: [_event('alice-authoritative-public')]),
    );
  }

  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) {
    ownerCursors.add(cursor);
    _ownerCalls++;
    if (_ownerCalls == 1) {
      return Future.value(
        BusinessEventPage(
          items: [_event('alice-owner', owner: 'did:plc:alice')],
          cursor: 'alice-owner-cursor',
        ),
      );
    }
    if (_ownerCalls == 2) return aliceOwnerMore.future;
    if (_ownerCalls == 3) {
      return Future.value(
        BusinessEventPage(items: [_event('bob-owner', owner: 'did:plc:bob')]),
      );
    }
    return Future.value(
      BusinessEventPage(
        items: [_event('alice-authoritative-owner', owner: 'did:plc:alice')],
      ),
    );
  }

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) {
    _detailCalls++;
    if (_detailCalls == 1) return aliceDetail.future;
    if (owner.toString() == 'did:plc:bob') {
      return Future.value(_event('bob-detail', owner: 'did:plc:bob'));
    }
    return Future.value(
      _event('alice-authoritative-detail', owner: 'did:plc:alice'),
    );
  }
}

final class _ReadProfileRepository extends Fake implements ProfileRepository {
  _ReadProfileRepository(this.aliceProfile);

  final Completer<Profile> aliceProfile;
  int aliceCalls = 0;

  @override
  Future<Profile> fetch(String handleOrDid) {
    if (handleOrDid == 'bob.test') {
      return Future.value(
        _profile('bob', tagline: 'Bob authoritative profile'),
      );
    }
    aliceCalls++;
    if (aliceCalls == 1) return aliceProfile.future;
    return Future.value(
      _profile('alice', tagline: 'Alice authoritative profile'),
    );
  }

  @override
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) => throw UnimplementedError();
}

Profile _profile(
  String account, {
  required String tagline,
  String? productTitle,
}) => Profile(
  did: 'did:plc:$account',
  handle: '$account.test',
  crafts: const [],
  accountType: AccountType.business,
  business: BusinessProfile(
    cid: 'bafy-$account-profile',
    tagline: tagline,
    products: productTitle == null
        ? const []
        : [
            BusinessProductView(
              title: productTitle,
              uri: 'https://$account.example/product',
              image: BusinessImageView(
                cid: 'bafy-$account-image',
                mime: 'image/jpeg',
                size: 10,
                alt: '$account product',
                thumb: 'https://cdn.example/$account/thumb',
                fullsize: 'https://cdn.example/$account/full',
              ),
            ),
          ],
  ),
);

BusinessEventDraft _eventDraft(String name) => BusinessEventDraft(
  name: name,
  startsAt: DateTime(2026, 9, 5, 10),
  endsAt: DateTime(2026, 9, 5, 12),
  roles: const ['vendor'],
  mode: 'in-person',
  status: 'scheduled',
  timeZone: 'UTC',
  isAllDay: false,
);

ProductDraft _productDraft(String title) => ProductDraft(
  id: 'alice-product',
  title: title,
  destination: 'https://alice.example/product',
  image: const UploadedBusinessImageDraft(
    cid: 'bafy-alice-saved-image',
    mime: 'image/jpeg',
    size: 10,
  ),
);

ProfileImagePickResult _uploadedPick(String cid) => ProfileImagePickResult(
  previewBytes: Uint8List.fromList(
    base64Decode(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwC'
      'AAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=',
    ),
  ),
  uploaded: UploadedImageBlob(
    blob: UploadedBlob(
      type: 'blob',
      ref: UploadedBlobRef(link: cid),
      mimeType: 'image/jpeg',
      size: 3,
    ),
    cid: cid,
    mime: 'image/jpeg',
    size: 3,
  ),
);

BusinessEvent _event(String name, {String owner = 'did:plc:business'}) =>
    BusinessEvent(
      did: owner,
      rkey: name,
      uri: 'at://$owner/social.craftsky.business.event/$name',
      cid: 'bafy-$name',
      name: name,
      startsAt: DateTime.utc(2026, 9, 5, 10),
      endsAt: DateTime.utc(2026, 9, 5, 12),
      roles: const [],
      status: const BusinessOpenValue(value: 'scheduled', known: true),
      isAllDay: false,
      createdAt: DateTime.utc(2026, 8, 30),
      past: false,
      publicSuppressionReasons: const [],
      upcomingExclusionReasons: const [],
    );

final class _MutationBusinessRepository extends Fake
    implements BusinessRepository {
  final accountType = Completer<AccountType>();
  final products = Completer<RecordMutationResult>();
  final event = Completer<RecordMutationResult>();
  final report = Completer<ReportResult>();
  final combined = Completer<RecordMutationResult>();
  final calls = <String>[];
  var _profileWrites = 0;

  @override
  Future<AccountType> updateAccountType(AccountType value) {
    calls.add('account-type:${value.name}');
    return accountType.future;
  }

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) {
    calls.add('profile:$expectedCid:${body['tagline']}');
    _profileWrites++;
    return _profileWrites == 1 ? products.future : combined.future;
  }

  @override
  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    BusinessEventDraft draft,
  ) {
    calls.add('event:$owner:$rkey:$expectedCid:${draft.name}');
    return event.future;
  }

  @override
  Future<ReportResult> reportEvent(
    Did owner,
    RecordKey rkey,
    ReportSubmission submission,
  ) {
    calls.add('report:$owner:$rkey:${submission.reasonType}');
    return report.future;
  }
}
