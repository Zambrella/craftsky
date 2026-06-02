// Suggestion fixtures are clearer without forcing every constructor const.
// ignore_for_file: prefer_const_constructors

import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MockAccountSuggestionRepository', () {
    test(
      'UT-012 filters non-Craftsky accounts and sorts followed first',
      () async {
        final repository = MockAccountSuggestionRepository(
          accounts: const [
            AccountSuggestion(
              did: 'did:plc:alicia',
              handle: 'alicia.craftsky.social',
              displayName: 'Alicia',
              avatar: 'https://example.com/alicia.jpg',
              isCraftskyProfile: true,
              viewerIsFollowing: false,
            ),
            AccountSuggestion(
              did: 'did:plc:mallory',
              handle: 'alice.elsewhere.example',
              displayName: 'Mallory',
              avatar: null,
              isCraftskyProfile: false,
              viewerIsFollowing: true,
            ),
            AccountSuggestion(
              did: 'did:plc:alice',
              handle: 'alice.craftsky.social',
              displayName: 'Alice',
              avatar: 'https://example.com/alice.jpg',
              isCraftskyProfile: true,
              viewerIsFollowing: true,
            ),
          ],
        );

        final results = await repository.searchAccounts('ali');

        expect(results.map((account) => account.handle), [
          'alice.craftsky.social',
          'alicia.craftsky.social',
        ]);
        expect(results.first.displayName, 'Alice');
        expect(results.first.avatar, 'https://example.com/alice.jpg');
        expect(
          await repository.didForHandle('alice.craftsky.social'),
          'did:plc:alice',
        );
      },
    );
  });
}
