import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('saved-post provider keys support immutable copies and equality', () {
    final account = AccountKey('did:plc:alice');
    final uri = AtUri.parse(
      'at://did:plc:alice/social.craftsky.feed.post/private',
    );
    final postKey = SavedPostKey(account: account, uri: uri);
    final dialogKey = SavePostDialogKey(
      account: account,
      uri: uri,
      initialFolderId: 'private-folder',
    );
    final listKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.folder('private-folder'),
      sort: SavedPostSort.newest,
    );

    expect(postKey.copyWith(), postKey);
    expect(dialogKey.copyWith(initialFolderId: null).initialFolderId, isNull);
    expect(
      listKey.copyWith(sort: SavedPostSort.oldest),
      SavedPostListKey(
        account: account,
        scope: const SavedPostScope.folder('private-folder'),
        sort: SavedPostSort.oldest,
      ),
    );
    expect('$postKey $dialogKey $listKey', isNot(contains('private')));
  });
}
