import 'support/test_app.dart';

void main() {
  testApp(
    'OAuth, onboarding bio, rotate photo, two-image post, '
    'like, edit and delete',
    (
      modules,
      system,
      api,
    ) async {
      final run = DateTime.now().toUtc().microsecondsSinceEpoch;
      final bio = 'CraftSky smoke test $run';
      final text = 'CraftSky E2E smoke $run';
      await modules.auth.signIn();
      await modules.onboarding.completeWithBio(bio);
      await api.verifyBio(bio);
      await modules.posts.compose(text);
      await system.chooseFixturePhotos();
      final editedImageSize = await modules.posts.rotateFirstSelectedPhoto();
      await modules.posts.publish();
      final post = await api.findPublishedPost(text);
      try {
        await api.verifyPublishedImages(post, editedImageSize);
        await modules.posts.like(post);
        await api.verifyLiked(post);
        final editedText = '$text edited';
        await modules.posts.editTextAndRemoveSecondImage(post, editedText);
        await api.verifyEdited(post, editedText);
      } finally {
        await modules.posts.delete(post);
        await api.verifyDeleted(post);
      }
    },
  );
}
