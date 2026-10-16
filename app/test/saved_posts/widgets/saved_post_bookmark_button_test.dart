import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_bookmark_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  setUpAll(initializeMappers);

  for (final unmount in [false, true]) {
    testWidgets('deferred bookmark seed with unmount=$unmount', (tester) async {
      final container = ProviderContainer.test();
      final account = AccountKey('did:plc:alice');
      final post = PostMapper.fromMap({
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/one',
        'cid': 'bafy_one',
        'rkey': 'one',
        'text': 'A post',
        'tags': <String>[],
        'likeCount': 0,
        'repostCount': 0,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasSaved': false,
        'sponsored': false,
        'createdAt': '2026-10-16T12:00:00Z',
        'indexedAt': '2026-10-16T12:00:00Z',
        'author': {'did': 'did:plc:alice', 'handle': 'alice.test'},
      });
      final deferred = <void Function()>[];
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: Consumer(
                // Hold the build's microtask so disposal can win the race.
                builder: (context, ref, child) => runZoned(
                  () => SavedPostBookmarkButton(
                    account: account,
                    post: post,
                  ).build(context, ref),
                  zoneSpecification: ZoneSpecification(
                    scheduleMicrotask: (self, parent, zone, callback) {
                      deferred.add(() => zone.run(callback));
                    },
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      expect(deferred, isNotEmpty);
      if (unmount) await tester.pumpWidget(const SizedBox.shrink());
      for (final callback in [...deferred]) {
        callback();
      }
      await tester.pump();
      expect(tester.takeException(), isNull);
      expect(
        container
            .read(
              savedPostPresentationProvider(
                SavedPostKey(account: account, uri: post.uri),
              ),
            )
            .value
            ?.initialized,
        !unmount,
      );
    });
  }
}
