import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 AT-005 IT-009 builds reply subject route with source focus', () {
    const subjectUri = 'at://did:plc:subject/social.craftsky.feed.post/subject';
    const sourceUri = 'at://did:plc:source/social.craftsky.feed.post/source';

    final route = postThreadRouteForNotification(
      PostDestination(
        AtUri.parse(subjectUri),
        focusUri: AtUri.parse(sourceUri),
      ),
    )!;

    expect(route.did, 'did:plc:subject');
    expect(route.rkey, 'subject');
    expect(route.focus, sourceUri);
    expect(
      route.location,
      '/posts/did%3Aplc%3Asubject/subject?focus='
      'at%3A%2F%2Fdid%3Aplc%3Asource%2Fsocial.craftsky.feed.post%2Fsource',
    );
  });
}
