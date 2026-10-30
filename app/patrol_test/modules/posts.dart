import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/keys.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:image/image.dart' as image;
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

  Future<({int width, int height})> rotateFirstSelectedPhoto() async {
    // Native selection opens photo details. Return to the overview first so
    // this exercises the same details entry point as an ordinary composer.
    await $.tester.pageBack();
    await $(keys.composerImages.section('composer'))
        .which<ComposerImageAttachmentSection>(
          (section) => section.enabled && section.imagesState.isSubmitReady,
        )
        .waitUntilVisible();
    _verifySelectedFixtures();
    final before =
        $(keys.composerImages.section('composer')).evaluate().single.widget
            as ComposerImageAttachmentSection;
    final original = before.imagesState.images.first;
    final untouched = before.imagesState.images.last;
    await $(keys.composerImages.details('composer')).tap();
    await $(keys.composerImages.edit('composer', original.id)).scrollTo().tap();
    await $(keys.imageEditor.rotate).tap();
    await $(keys.imageEditor.done).tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.composerImages.page('composer')).waitUntilVisible();
    // The details page becomes hit-testable before the editor's reverse route
    // animation ends. Finish that transition before tapping its Back button.
    await $.pumpAndSettle();
    await $.tester.pageBack();
    await $(keys.composerImages.details('composer')).scrollTo();
    await $(keys.composerImages.section('composer'))
        .which<ComposerImageAttachmentSection>(
          (section) => section.enabled && section.imagesState.isSubmitReady,
        )
        .waitUntilVisible();
    final after =
        $(keys.composerImages.section('composer')).evaluate().single.widget
            as ComposerImageAttachmentSection;
    final images = after.imagesState.images;
    final bytes = images.first.previewBytes;
    final rotated = bytes == null ? null : image.decodeImage(bytes);
    if (images.length != 2 ||
        images.first.id != original.id ||
        images.last.id != untouched.id ||
        rotated == null ||
        rotated.width <= 0 ||
        rotated.height <= 0 ||
        listEquals(bytes, original.previewBytes) ||
        !listEquals(images.last.previewBytes, untouched.previewBytes)) {
      throw StateError(
        'Expected a changed, decodable first fixture while preserving '
        'image order and the second fixture. '
        'Actual: ${rotated?.width}x${rotated?.height}; '
        'count: ${images.length}; '
        'first ID retained: ${images.first.id == original.id}; '
        'last ID retained: ${images.last.id == untouched.id}; '
        'first bytes changed: ${!listEquals(bytes, original.previewBytes)}; '
        'second bytes retained: '
        '${listEquals(images.last.previewBytes, untouched.previewBytes)}.',
      );
    }
    return (width: rotated.width, height: rotated.height);
  }

  Future<void> publish() async {
    await $(keys.feed.submit).tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.dialog.confirm).tap(settlePolicy: SettlePolicy.noSettle);
  }

  void _verifySelectedFixtures() {
    final section =
        $(keys.composerImages.section('composer')).evaluate().single.widget
            as ComposerImageAttachmentSection;
    final fixtures = section.imagesState.images.map((draft) {
      final bytes = draft.previewBytes;
      final decoded = bytes == null ? null : image.decodeImage(bytes);
      if (decoded == null || decoded.width != 640 || decoded.height != 480) {
        return null;
      }
      final base = decoded.getPixel(10, 10);
      final alternate = decoded.getPixel(50, 10);
      bool matches(List<int> color, image.Pixel pixel) =>
          (pixel.r - color[0]).abs() <= 10 &&
          (pixel.g - color[1]).abs() <= 10 &&
          (pixel.b - color[2]).abs() <= 10;
      if (matches([226, 102, 91], base) &&
          matches([255, 214, 147], alternate)) {
        return 'coral';
      }
      if (matches([46, 135, 138], base) &&
          matches([197, 232, 216], alternate)) {
        return 'teal';
      }
      return null;
    }).toList();
    if (fixtures.length != 2 ||
        !fixtures.contains('coral') ||
        !fixtures.contains('teal')) {
      throw StateError(
        'Select the coral and teal test fixtures before posting. '
        'Detected fixture types: $fixtures.',
      );
    }
  }

  Future<void> like(Post post) async {
    await $(keys.feed.post(post.uri)).scrollTo();
    await $(keys.feed.post(post.uri)).$(keys.feed.action('Like')).scrollTo();
    await $(keys.feed.post(post.uri))
        .$(keys.feed.action('Like'))
        .tap(settlePolicy: SettlePolicy.noSettle);
  }

  Future<void> editTextAndRemoveSecondImage(Post post, String text) async {
    await $(keys.feed.post(post.uri)).$(keys.feed.menu).tap();
    await $(keys.feed.edit).tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.feed.composerBody).enterText(text);
    FocusManager.instance.primaryFocus?.unfocus();
    await $(keys.composerImages.details('composer')).scrollTo();
    final section =
        $(keys.composerImages.section('composer')).evaluate().single.widget
            as ComposerImageAttachmentSection;
    final images = section.imagesState.images;
    if (images.length != 2) {
      throw StateError('Expected two published images in the post editor.');
    }
    final remove = keys.composerImages.remove('composer', images[1].id);
    await $(keys.composerImages.details('composer')).tap();
    await $(remove).scrollTo();
    await $(remove).tap();
    await $.tester.pageBack();
    await $(keys.composerImages.section('composer'))
        .which<ComposerImageAttachmentSection>(
          (section) => section.enabled && section.imagesState.isSubmitReady,
        )
        .waitUntilVisible();
    await $(keys.feed.submit).tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.feed.post(post.uri)).waitUntilVisible();
  }

  Future<void> delete(Post post) async {
    await $(keys.feed.post(post.uri)).$(keys.feed.menu).tap();
    await $(keys.feed.delete).tap();
    await $(keys.dialog.confirm).tap(settlePolicy: SettlePolicy.noSettle);
  }
}
