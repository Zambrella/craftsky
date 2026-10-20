import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/widgets.dart';
import 'package:patrol/patrol.dart';

final class PostsModule {
  new(this.$);

  final PatrolIntegrationTester $;

  Future<void> compose(String text) async {
    await $(keys.feed.compose).tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.feed.regularPost).tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.feed.composerBody).enterText(text);
    FocusManager.instance.primaryFocus?.unfocus();
    await $(keys.composerImages.add('composer')).scrollTo();
    await $(keys.composerImages.add('composer')).tap();
    await $(keys.imageSource.photos('composer')).tap(
      settlePolicy: SettlePolicy.noSettle,
    );
  }

  Future<void> publish() async {
    await $.platform.android.pressBack();
    await $(keys.feed.submit)
        .which<ChunkyButton>((button) => button.onPressed != null)
        .tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.dialog.confirm).tap(settlePolicy: SettlePolicy.noSettle);
  }

  Future<void> like(Post post) async {
    await $(keys.feed.post(post.uri)).scrollTo();
    await $(keys.feed.post(post.uri)).$(keys.feed.action('Like')).scrollTo();
    await $(keys.feed.post(post.uri))
        .$(keys.feed.action('Like'))
        .tap(settlePolicy: SettlePolicy.noSettle);
  }

  Future<void> delete(Post post) async {
    await $(keys.feed.post(post.uri)).$(keys.feed.menu).tap();
    await $(keys.feed.delete).tap();
    await $(keys.dialog.confirm).tap(settlePolicy: SettlePolicy.noSettle);
  }
}
