import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef BusinessRecordReadFence = ({
  AccountSessionLease lease,
  Set<String> identityPrefixes,
  Set<(String, int)> overlays,
});

typedef BusinessEventListReconciliation = ({
  List<BusinessEvent> events,
  bool isStale,
});

PdsMutationScope businessEventMutationScope(
  AccountSessionLease lease,
  String uri,
) => PdsMutationScope(lease: lease, identity: 'business-event:$uri');

PdsMutationScope businessEventCreateMutationScope(
  AccountSessionLease lease,
) => PdsMutationScope(
  lease: lease,
  identity: 'business-event-create',
);

PdsMutationScope businessProfileMutationScope(
  AccountSessionLease lease,
  Did owner,
) => PdsMutationScope(
  lease: lease,
  identity:
      'business-profile:at://$owner/social.craftsky.business.profile/self',
);

BusinessRecordReadFence captureBusinessEventListRead(
  Ref ref,
  AccountSessionLease lease,
) => _captureBusinessRecordRead(ref, lease, const {'business-event:'});

BusinessRecordReadFence captureBusinessEventRead(
  Ref ref,
  AccountSessionLease lease,
  String uri,
) => _captureBusinessRecordRead(ref, lease, {'business-event:$uri'});

BusinessRecordReadFence captureBusinessProfileRead(
  Ref ref,
  AccountSessionLease lease,
  Did owner,
) => _captureBusinessRecordRead(ref, lease, {
  businessProfileMutationScope(lease, owner).identity,
});

bool isBusinessRecordReadCurrent(Ref ref, BusinessRecordReadFence fence) =>
    _sameSignatures(
      fence.overlays,
      _overlaySignatures(
        ref.read(pdsRecordOperationControllerProvider),
        fence.lease,
        fence.identityPrefixes,
      ),
    );

PdsRecordProjection businessEventProjection(BusinessEvent? event) =>
    PdsRecordProjection(
      uri: event?.uri.toString() ?? '',
      cid: event?.cid.toString() ?? '',
      content: event == null
          ? const {}
          : businessEventControlledContent({
              'name': event.name,
              'startsAt': _canonicalUtc(event.startsAt),
              'endsAt': _canonicalUtc(event.endsAt),
              'roles': event.roles.map((value) => value.value).toList(),
              'mode': event.mode?.value,
              'status': event.status.value,
              'timeZone': event.timeZone,
              'isAllDay': event.isAllDay,
              'summary': event.summary,
              'venueName': event.venueName,
              'eventUri': event.eventUri,
              'registrationUri': event.registrationUri,
              'image': _imageContent(event.image),
            }),
    );

Map<String, Object?> businessEventControlledContent(
  Map<String, dynamic> content,
) => <String, Object?>{
  ...content,
  for (final field in const [
    'mode',
    'timeZone',
    'summary',
    'venueName',
    'eventUri',
    'registrationUri',
    'image',
  ])
    field: content[field],
};

PdsRecordProjection businessProfileProjection(
  Did owner,
  BusinessProfile? profile,
) => PdsRecordProjection(
  uri: 'at://$owner/social.craftsky.business.profile/self',
  cid: profile?.cid.toString() ?? '',
  content: profile == null
      ? const {}
      : businessProfileControlledContent({
          'businessTypes': profile.businessTypes
              .map((value) => value.value)
              .toList(),
          'offerings': profile.offerings.map((value) => value.value).toList(),
          'products': profile.products.map(_productContent).toList(),
          'tagline': profile.tagline,
          'hoursNote': profile.hoursNote,
          'serviceArea': profile.serviceArea,
          'location': switch (profile.location) {
            final value? => {
              'country': value.country,
              if (value.locality != null) 'locality': value.locality,
            },
            null => null,
          },
          'primaryAction': switch (profile.primaryAction) {
            final value? => {
              'type': value.type,
              'destination': value.destination,
            },
            null => null,
          },
        }),
);

Map<String, Object?> businessProfileControlledContent(
  Map<String, dynamic> content,
) => <String, Object?>{
  ...content,
  for (final field in const [
    'tagline',
    'hoursNote',
    'serviceArea',
    'location',
    'primaryAction',
  ])
    field: content[field],
};

BusinessEvent? applyBusinessEventOverlay(
  Ref ref,
  AccountSessionLease lease,
  String uri,
  BusinessEvent? authoritative,
) {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = businessEventMutationScope(lease, uri);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return authoritative;
  if (controller.reconcile(
    scope,
    authoritative == null ? null : businessEventProjection(authoritative),
  )) {
    return authoritative;
  }
  final optimistic = overlay.optimisticValue;
  return optimistic is BusinessEvent ? optimistic : null;
}

bool hasBusinessEventOverlay(
  Ref ref,
  AccountSessionLease lease,
  String uri,
) =>
    ref
        .read(pdsRecordOperationControllerProvider)
        .overlayFor(businessEventMutationScope(lease, uri)) !=
    null;

BusinessProfile? applyBusinessProfileOverlay(
  Ref ref,
  AccountSessionLease lease,
  Did owner,
  BusinessProfile? authoritative,
) {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = businessProfileMutationScope(lease, owner);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return authoritative;
  if (controller.reconcile(
    scope,
    businessProfileProjection(owner, authoritative),
  )) {
    return authoritative;
  }
  final optimistic = overlay.optimisticValue;
  return optimistic is BusinessProfile ? optimistic : null;
}

bool hasBusinessProfileOverlay(
  Ref ref,
  AccountSessionLease lease,
  Did owner,
) =>
    ref
        .read(pdsRecordOperationControllerProvider)
        .overlayFor(businessProfileMutationScope(lease, owner)) !=
    null;

BusinessEventListReconciliation applyBusinessEventListOverlays(
  Ref ref, {
  required AccountSessionLease lease,
  required BusinessRecordReadFence fence,
  required Did owner,
  required Iterable<BusinessEvent> authoritative,
  bool Function(BusinessEvent event)? acceptedFilter,
}) {
  final isStale = !isBusinessRecordReadCurrent(ref, fence);
  final events = <BusinessEvent>[];
  final identities = <String>{};
  for (final event in authoritative) {
    final hadOverlay = hasBusinessEventOverlay(
      ref,
      lease,
      event.uri.toString(),
    );
    final resolved = applyBusinessEventOverlay(
      ref,
      lease,
      event.uri.toString(),
      event,
    );
    if (resolved != null &&
        (!hadOverlay || acceptedFilter == null || acceptedFilter(resolved)) &&
        identities.add(resolved.uri.toString())) {
      events.add(resolved);
    }
  }
  final controller = ref.read(pdsRecordOperationControllerProvider);
  for (final overlay in controller.activeOverlays) {
    final optimistic = overlay.optimisticValue;
    if (overlay.token.scope.lease == lease &&
        optimistic is BusinessEvent &&
        optimistic.did == owner &&
        (acceptedFilter == null || acceptedFilter(optimistic)) &&
        identities.add(optimistic.uri.toString())) {
      events.add(optimistic);
    }
  }
  return (events: events, isStale: isStale);
}

AccountSessionLease? firstBusinessEventOverlayLease(Ref ref) {
  for (final overlay
      in ref.read(pdsRecordOperationControllerProvider).activeOverlays) {
    if (overlay.optimisticValue is BusinessEvent) {
      return overlay.token.scope.lease;
    }
  }
  return null;
}

Set<(String, int)> _overlaySignatures(
  PdsRecordOperationController controller,
  AccountSessionLease lease,
  Set<String> identityPrefixes,
) => {
  for (final overlay in controller.activeOverlays)
    if (overlay.token.scope.lease == lease &&
        identityPrefixes.any(
          overlay.token.scope.identity.startsWith,
        ))
      (overlay.token.scope.identity, overlay.token.sequence),
};

BusinessRecordReadFence _captureBusinessRecordRead(
  Ref ref,
  AccountSessionLease lease,
  Set<String> identityPrefixes,
) => (
  lease: lease,
  identityPrefixes: identityPrefixes,
  overlays: _overlaySignatures(
    ref.read(pdsRecordOperationControllerProvider),
    lease,
    identityPrefixes,
  ),
);

bool _sameSignatures(Set<(String, int)> first, Set<(String, int)> second) =>
    first.length == second.length && first.containsAll(second);

Map<String, Object?> _productContent(BusinessProductView product) => {
  'title': product.title,
  if (product.uri != null) 'uri': product.uri,
  if (product.image != null) 'image': _imageContent(product.image),
  if (product.price case final price?)
    'price': {'amount': price.amount, 'currency': price.currency},
};

Map<String, Object?>? _imageContent(BusinessImageView? image) => image == null
    ? null
    : {
        'image': {
          r'$type': 'blob',
          'ref': {r'$link': image.cid.toString()},
          'mimeType': image.mime,
          'size': image.size,
        },
        'alt': image.alt,
        if (image.aspectRatio case final ratio?)
          'aspectRatio': {'width': ratio.width, 'height': ratio.height},
      };

String _canonicalUtc(DateTime value) {
  final utc = value.toUtc();
  String two(int component) => component.toString().padLeft(2, '0');
  return '${utc.year.toString().padLeft(4, '0')}-'
      '${two(utc.month)}-${two(utc.day)}T'
      '${two(utc.hour)}:${two(utc.minute)}:${two(utc.second)}Z';
}
