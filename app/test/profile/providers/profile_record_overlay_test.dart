import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('personal profile overlay waits for both fixed projections', () {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final container = ProviderContainer(
      overrides: [
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    addTearDown(container.dispose);
    final lease = AccountSessionLease(
      account: AccountKey('did:plc:owner'),
      sessionGeneration: 2,
    );
    final accepted = Profile(
      did: 'did:plc:owner',
      handle: 'maker.test',
      displayName: 'Maker',
      crafts: const ['weaving'],
    );
    final projection = personalProfileProjection(accepted);
    final reconciliation = PdsCompoundProfileReconciliation(
      bluesky: PdsFixedKeyReconciliation(
        uri: projection.bluesky!.uri,
        controlledContent: projection.bluesky!.content,
      ),
      craftsky: PdsFixedKeyReconciliation(
        uri: projection.craftsky!.uri,
        controlledContent: projection.craftsky!.content,
      ),
    );
    final token = controller.begin(
      scope: personalProfileMutationScope(lease, accepted.did),
      operationKey: 'profile-key',
      endpoint: '/v1/profiles/me',
      immutableBody: '{}',
    );
    controller.markAccepted(
      token,
      optimisticValue: accepted,
      agrees: (value) => reconciliation.agrees(
        value is PdsCompoundProfileProjection ? value : null,
      ),
    );

    final partialProjection = accepted.copyWith(crafts: const ['sewing']);
    final masked = _withRef(
      container,
      (ref) => applyPersonalProfileOverlay(ref, lease, partialProjection),
    );
    expect(masked, same(accepted));
    expect(controller.activeOverlays, hasLength(1));

    final settled = accepted.copyWith(followerCount: 10);
    final visible = _withRef(
      container,
      (ref) => applyPersonalProfileOverlay(ref, lease, settled),
    );
    expect(visible, same(settled));
    expect(controller.activeOverlays, isEmpty);
  });

  test('profile read composes personal and business overlays', () {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final container = ProviderContainer(
      overrides: [
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    addTearDown(container.dispose);
    final lease = AccountSessionLease(
      account: AccountKey('did:plc:owner'),
      sessionGeneration: 2,
    );
    final authoritative = Profile(
      did: 'did:plc:owner',
      handle: 'maker.test',
      displayName: 'Before',
      crafts: const ['sewing'],
      business: BusinessProfile(
        cid: 'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        tagline: 'Before business',
      ),
    );
    final acceptedPersonal = authoritative.copyWith(
      displayName: 'After',
      crafts: const ['weaving'],
    );
    final personalProjection = personalProfileProjection(acceptedPersonal);
    final personalToken = controller.begin(
      scope: personalProfileMutationScope(lease, authoritative.did),
      operationKey: 'personal-key',
      endpoint: '/v1/profiles/me',
      immutableBody: '{}',
    );
    final personalReconciliation = PdsCompoundProfileReconciliation(
      bluesky: PdsFixedKeyReconciliation(
        uri: personalProjection.bluesky!.uri,
        controlledContent: personalProjection.bluesky!.content,
      ),
      craftsky: PdsFixedKeyReconciliation(
        uri: personalProjection.craftsky!.uri,
        controlledContent: personalProjection.craftsky!.content,
      ),
    );
    controller.markAccepted(
      personalToken,
      optimisticValue: acceptedPersonal,
      agrees: (value) => personalReconciliation.agrees(
        value is PdsCompoundProfileProjection ? value : null,
      ),
    );

    final acceptedBusiness = BusinessProfile(
      cid: 'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
      tagline: 'After business',
    );
    final businessToken = controller.begin(
      scope: businessProfileMutationScope(lease, authoritative.did),
      operationKey: 'business-key',
      endpoint: '/v1/profiles/me/business',
      immutableBody: '{}',
    );
    final businessReconciliation = PdsFixedKeyReconciliation(
      uri: 'at://${authoritative.did}/social.craftsky.business.profile/self',
      controlledContent: businessProfileProjection(
        authoritative.did,
        acceptedBusiness,
      ).content,
    );
    controller.markAccepted(
      businessToken,
      optimisticValue: acceptedBusiness,
      agrees: (value) =>
          value is PdsRecordProjection && businessReconciliation.agrees(value),
    );

    final visible = _withRef(
      container,
      (ref) => applyUserProfileRecordOverlays(ref, lease, authoritative),
    );

    expect(visible.displayName, 'After');
    expect(visible.crafts, ['weaving']);
    expect(visible.business?.tagline, 'After business');
  });
}

T _withRef<T>(ProviderContainer container, T Function(Ref ref) operation) =>
    container.read(Provider<T>(operation));
