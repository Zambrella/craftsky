import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const accepted = PdsRecordProjection(
    uri: 'at://did:plc:owner/social.craftsky.feed.post/selected',
    cid: 'cid-accepted',
    content: {
      'text': 'accepted',
      'embed': {
        'images': ['a', 'b'],
      },
    },
  );

  group('append create', () {
    final predicate = PdsAppendReconciliation(
      selectedUri: accepted.uri,
      acceptedCid: accepted.cid,
      controlledContent: accepted.content,
    );

    test('requires selected URI and accepted CID or controlled content', () {
      expect(predicate.agrees(accepted), isTrue);
      expect(
        predicate.agrees(
          PdsRecordProjection(
            uri: accepted.uri,
            cid: 'cid-projected-differently',
            content: accepted.content,
          ),
        ),
        isTrue,
      );
      expect(
        predicate.agrees(
          PdsRecordProjection(
            uri: 'at://did:plc:owner/social.craftsky.feed.post/other',
            cid: accepted.cid,
            content: accepted.content,
          ),
        ),
        isFalse,
      );
      expect(
        predicate.agrees(
          PdsRecordProjection(
            uri: accepted.uri,
            cid: 'cid-other',
            content: const {'text': 'other'},
          ),
        ),
        isFalse,
      );
    });
  });

  test('addressed update uses the same exact-identity agreement', () {
    final predicate = PdsAddressedUpdateReconciliation(
      uri: accepted.uri,
      acceptedCid: accepted.cid,
      controlledContent: accepted.content,
    );
    expect(predicate.agrees(accepted), isTrue);
    expect(
      predicate.agrees(
        PdsRecordProjection(
          uri: accepted.uri,
          cid: 'cid-other',
          content: accepted.content,
        ),
      ),
      isTrue,
    );
  });

  test('addressed delete agrees only with exact-scope absence', () {
    final predicate = PdsAddressedDeleteReconciliation(uri: accepted.uri);
    expect(predicate.agrees(null), isTrue);
    expect(predicate.agrees(accepted), isFalse);
  });

  test('set reconciliation compares logical activity only', () {
    const active = PdsSetReconciliation(active: true);
    const inactive = PdsSetReconciliation(active: false);
    expect(active.agrees(true), isTrue);
    expect(active.agrees(false), isFalse);
    expect(inactive.agrees(false), isTrue);
  });

  test('fixed key compares URI and only controlled fields', () {
    const predicate = PdsFixedKeyReconciliation(
      uri: 'at://did:plc:owner/social.craftsky.actor.profile/self',
      controlledContent: {'businessName': 'Needle & Thread'},
    );
    expect(
      predicate.agrees(
        const PdsRecordProjection(
          uri: 'at://did:plc:owner/social.craftsky.actor.profile/self',
          cid: 'cid-newer',
          content: {
            'businessName': 'Needle & Thread',
            'uncontrolled': 'ignored',
          },
        ),
      ),
      isTrue,
    );
    expect(
      predicate.agrees(
        const PdsRecordProjection(
          uri: 'at://did:plc:owner/social.craftsky.actor.profile/self',
          cid: 'cid-newer',
          content: {'businessName': 'Different'},
        ),
      ),
      isFalse,
    );
  });

  test('compound profile requires both controlled projections to agree', () {
    const predicate = PdsCompoundProfileReconciliation(
      bluesky: PdsFixedKeyReconciliation(
        uri: 'at://did:plc:owner/app.bsky.actor.profile/self',
        controlledContent: {'displayName': 'Maker'},
      ),
      craftsky: PdsFixedKeyReconciliation(
        uri: 'at://did:plc:owner/social.craftsky.actor.profile/self',
        controlledContent: {
          'crafts': ['weaving', 'sewing'],
        },
      ),
    );
    const bluesky = PdsRecordProjection(
      uri: 'at://did:plc:owner/app.bsky.actor.profile/self',
      cid: 'cid-bsky',
      content: {'displayName': 'Maker'},
    );
    const craftsky = PdsRecordProjection(
      uri: 'at://did:plc:owner/social.craftsky.actor.profile/self',
      cid: 'cid-craftsky',
      content: {
        'crafts': ['weaving', 'sewing'],
      },
    );

    expect(
      predicate.agrees(
        const PdsCompoundProfileProjection(
          bluesky: bluesky,
          craftsky: craftsky,
        ),
      ),
      isTrue,
    );
    expect(
      predicate.agrees(
        PdsCompoundProfileProjection(
          bluesky: bluesky,
          craftsky: PdsRecordProjection(
            uri: craftsky.uri,
            cid: 'cid-other',
            content: const {
              'crafts': ['knitting'],
            },
          ),
        ),
      ),
      isFalse,
    );
  });
}
