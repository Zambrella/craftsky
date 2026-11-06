import 'package:craftsky_app/shared/browsing/guest_session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-003 Explore dismisses automatic home prompt for the process session',
    () {
      final session = GuestSession();
      expect(session.shouldPrompt('/profiles/did:plc:alice'), false);
      expect(session.shouldPrompt('/feed'), true);
      session.explore();
      expect(session.shouldPrompt('/feed'), false);
      expect(GuestSession().shouldPrompt('/feed'), true);
    },
  );
  test('UT-005 auth return accepts only public local reading destinations', () {
    final session = GuestSession();
    for (final path in [
      'https://evil.test',
      '//evil.test',
      '/settings',
      '/profiles/me',
      '/search?token=secret',
    ]) {
      session.prepareSignIn(path);
      expect(session.returnTo, '/feed');
    }
    session.prepareSignIn('/posts/did:plc:alice/key?showComments=true');
    expect(session.returnTo, '/posts/did:plc:alice/key?showComments=true');
  });
  test('UT-005 authenticated return is consumed once after onboarding', () {
    final session = GuestSession()
      ..prepareSignIn('/profiles/did:plc:alice?tab=posts');
    expect(session.consumeReturnTo(), '/profiles/did:plc:alice?tab=posts');
    expect(session.returnTo, '/feed');
  });
}
