import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('dart_mappable decodes relationship wire state', () {
    final relationship = ProfileRelationshipMapper.fromMap({
      'muted': true,
      'blocking': false,
      'blockedBy': true,
      'uri': 'at://did:plc:alice/app.bsky.graph.block/3abc',
      'cid': 'bafy-block',
      'rkey': '3abc',
      'pendingAction': 'unblock',
      'lastError': 'server-controlled',
      'confirmedOverlay': true,
    });

    expect(relationship.muted, isTrue);
    expect(relationship.blocking, isFalse);
    expect(relationship.blockedBy, isTrue);
    expect(relationship.uri, 'at://did:plc:alice/app.bsky.graph.block/3abc');
    expect(relationship.cid, 'bafy-block');
    expect(relationship.rkey, '3abc');
    expect(relationship.pendingAction, isNull);
    expect(relationship.lastError, isNull);
    expect(relationship.confirmedOverlay, isFalse);
    expect(relationship.initialized, isTrue);
  });

  test('generated copyWith preserves and clears nullable runtime state', () {
    const pending = ProfileRelationship(
      muted: true,
      pendingAction: ProfileRelationshipAction.unmute,
      lastError: 'failed',
      initialized: true,
    );

    final cleared = pending.copyWith(pendingAction: null, lastError: null);

    expect(cleared.muted, isTrue);
    expect(cleared.pendingAction, isNull);
    expect(cleared.lastError, isNull);
    expect(cleared.initialized, isTrue);
    expect(cleared, const ProfileRelationship(muted: true, initialized: true));
  });
}
