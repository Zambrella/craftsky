import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final type in [null, 'regular', 'business']) {
    test(
      'hydrated identity retains account type $type through decode and copy',
      () {
        final wire = <String, dynamic>{
          'did': 'did:plc:maker',
          'handle': 'maker.test',
          'displayName': 'Maker',
          'crafts': ['knitting'],
          'isCraftskyProfile': true,
          'viewerIsFollowing': false,
          'accountType': ?type,
        };
        final author = PostAuthorMapper.fromMap(wire);
        final summary = ProfileAccountSummaryMapper.fromMap(wire);
        final search = ProfileSearchResultMapper.fromMap(wire);
        final expected = type == 'business';
        expect(author.isBusiness, expected);
        expect(author.copyWith().isBusiness, expected);
        expect(summary.isBusiness, expected);
        expect(summary.copyWith().isBusiness, expected);
        expect(search.isBusiness, expected);
        expect(search.summary.isBusiness, expected);
        if (expected) {
          expect(author.accountType, AccountType.business);
          expect(author.toMap()['accountType'], 'business');
          expect(search.summary.toMap()['accountType'], 'business');
        }
        expect(author.crafts, ['knitting']);
      },
    );
  }
}
