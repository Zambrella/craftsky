import 'package:craftsky_app/feed/composer/published_post_embed_edit.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-002 external cards are retained or removed without '
      'changing generated metadata', () {
    final edit = PublishedPostEmbedEdit({
      r'$type': 'app.bsky.embed.external',
      'external': {
        'uri': 'https://example.com',
        'title': 'Old title',
        'description': 'Old description',
        'thumb': {
          'ref': {r'$link': 'bafythumb'},
        },
      },
    });
    expect(edit.externalTitle, 'Old title');
    expect(edit.externalUri, 'https://example.com');
    expect(edit.externalDescription, 'Old description');
    expect(edit.changes, isEmpty);
    edit.remove();
    expect(edit.changes, {'embed': null});
  });
  test('UT-002 retained video alt changes omit unchanged media references', () {
    final edit = PublishedPostEmbedEdit({
      r'$type': 'app.bsky.embed.video',
      'video': {
        r'$type': 'blob',
        'ref': {r'$link': 'bafyvideo'},
        'size': 100,
      },
      'alt': 'Original alt',
    });
    expect(edit.hasRetainedVideo, isTrue);
    expect(edit.videoAlt, 'Original alt');
    expect(edit.changes, isEmpty);
    edit.videoAlt = 'Updated alt';
    expect(edit.changes, {
      'embed': {'alt': 'Updated alt'},
    });
    edit.videoAlt = 'Original alt';
    expect(edit.changes, isEmpty);
    edit.remove();
    expect(edit.hasRetainedVideo, isFalse);
    expect(edit.changes, {'embed': null});
  });
}
