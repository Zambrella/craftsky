import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-010 projects saved failures without private server data', () {
    const sentinels = [
      'Secret Folder',
      'folder-private-42',
      'at://did:plc:alice/social.craftsky.feed.post/private',
      'alice-owner-target',
      'cursor-private-42',
    ];
    final raw = ApiServerError(
      sentinels.join(' '),
      details: const ApiFailureDetails(
        appViewError: 'internal_error',
        endpointCategory: 'appview.saved_posts',
      ),
    );
    final failure = SavedPostFailure.from(
      raw,
      operation: SavedPostOperation.loadPosts,
    );

    expect(failure.kind, SavedPostFailureKind.server);
    expect(failure.canRetry, isTrue);
    expect(
      failure.copyWith(operation: SavedPostOperation.unsave),
      const SavedPostFailure(
        kind: SavedPostFailureKind.server,
        operation: SavedPostOperation.unsave,
      ),
    );
    expect(
      failure.localizedMessage(AppLocalizationsEn()),
      "Saved posts couldn't load.",
    );
    final exposed =
        '$failure '
        '${failure.localizedMessage(AppLocalizationsEn())}';
    for (final sentinel in sentinels) {
      expect(exposed, isNot(contains(sentinel)));
    }

    expect(
      SavedPostFailure.from(
        const ApiNetworkError('private network detail'),
        operation: SavedPostOperation.loadFolders,
      ).kind,
      SavedPostFailureKind.network,
    );
    expect(
      SavedPostFailure.from(
        const ApiBadRequest('validation_failed'),
        operation: SavedPostOperation.saveOrMove,
      ).canRetry,
      isFalse,
    );
    expect(
      SavedPostFailure.from(
        const ApiCanceled(),
        operation: SavedPostOperation.unsave,
      ).shouldPresent,
      isFalse,
    );
  });
}
