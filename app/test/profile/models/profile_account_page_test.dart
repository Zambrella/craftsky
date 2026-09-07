import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-007 decodes interaction account pages with optional cursor', () {
    final firstPage = ProfileAccountPageMapper.fromMap({
      'items': [
        {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
          'displayName': 'Alice',
          'description': 'Knitter and spinner',
          'avatar': 'https://cdn.example/alice.jpg',
          'isCraftskyProfile': true,
          'muted': true,
          'blocking': false,
          'blockedBy': false,
        },
      ],
      'totalCount': 2,
      'cursor': 'opaque-next',
    });
    final finalPage = ProfileAccountPageMapper.fromMap({
      'items': [
        {
          'did': 'did:plc:bob',
          'handle': 'bob.example',
          'isCraftskyProfile': false,
        },
      ],
      'totalCount': 2,
    });

    expect(firstPage.totalCount, 2);
    expect(firstPage.cursor, 'opaque-next');
    expect(firstPage.items, hasLength(1));
    final alice = firstPage.items.single;
    expect(alice.did.toString(), 'did:plc:alice');
    expect(alice.handle.toString(), 'alice.craftsky.social');
    expect(alice.displayName, 'Alice');
    expect(alice.description, 'Knitter and spinner');
    expect(alice.avatar, 'https://cdn.example/alice.jpg');
    expect(alice.isCraftskyProfile, isTrue);
    expect(alice.muted, isTrue);
    expect(alice.blocking, isFalse);
    expect(alice.blockedBy, isFalse);

    expect(finalPage.totalCount, 2);
    expect(finalPage.cursor, isNull);
    expect(finalPage.items.single.did.toString(), 'did:plc:bob');
    expect(finalPage.items.single.handle.toString(), 'bob.example');
    expect(finalPage.items.single.isCraftskyProfile, isFalse);
  });
}
