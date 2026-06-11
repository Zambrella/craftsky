import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-010 chooser forwards the selected composer result', (
    tester,
  ) async {
    final created = _post(rkey: 'created');
    late Future<Post?> chooserResult;

    await tester.pumpWidget(
      ProviderScope(
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => Scaffold(
                body: Center(
                  child: ElevatedButton(
                    onPressed: () {
                      chooserResult = showTopLevelPostComposerChooser(
                        context,
                        position: RelativeRect.fill,
                        showProjectComposer: (_) async => created,
                      );
                    },
                    child: const Text('New post'),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('New post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Project post'));
    await tester.pumpAndSettle();

    expect(await chooserResult, same(created));
  });
}

Post _post({required String rkey}) {
  final now = DateTime.utc(2026, 6, 11);
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafyreibazjzrzibga2jwt5co2yus7j2w6p3n3cb6nn4njvkzcxwrlfvula',
    rkey: rkey,
    text: 'Created post',
    tags: const [],
    createdAt: now,
    indexedAt: now,
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
  );
}
