import 'support/test_app.dart';

void main() {
  testApp('OAuth, onboarding bio, two-image post, like and delete', (
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
    await modules.posts.publish();
    final post = await api.findPublishedPost(text);
    try {
      await modules.posts.like(post);
      await api.verifyLiked(post);
    } finally {
      await modules.posts.delete(post);
      await api.verifyDeleted(post);
    }
  });
}
