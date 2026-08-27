import 'dart:typed_data';

import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-018 frozen scheduled external round-trips exactly', () {
    const external = ScheduledPostExternal(
      sourceUri: 'https://source.example/pattern',
      uri: 'https://final.example/pattern#section',
      title: 'Frozen pattern',
      description: 'Frozen description',
      thumbMediaId: '55555555-5555-4555-8555-555555555555',
    );

    expect(ScheduledPostExternal.fromMap(external.toMap()).toMap(), {
      'sourceUri': 'https://source.example/pattern',
      'uri': 'https://final.example/pattern#section',
      'title': 'Frozen pattern',
      'description': 'Frozen description',
      'thumbMediaId': '55555555-5555-4555-8555-555555555555',
    });
  });

  test('UT-018 metadata-only scheduled external omits thumbnail identity', () {
    const external = ScheduledPostExternal(
      sourceUri: 'https://source.example/pattern',
      uri: 'https://final.example/pattern',
      title: 'Frozen pattern',
      description: '',
    );

    expect(external.toMap(), isNot(contains('thumbMediaId')));
  });

  test('IT-015 stages raw thumbnail before returning frozen state', () async {
    final events = <String>[];
    final selection = SelectedLinkPreview(
      candidate: LinkPreviewCandidate.parse('https://source.example/pattern'),
      preview: LinkPreview(
        url: Uri.parse('https://final.example/pattern'),
        title: 'Frozen pattern',
        description: 'Frozen description',
        thumbnail: LinkPreviewThumbnail(
          bytes: Uint8List.fromList([1, 2, 3]),
          mimeType: 'image/png',
          width: 20,
          height: 10,
        ),
      ),
    );

    final external = await materializeScheduledExternal(
      selection,
      mediaId: '55555555-5555-4555-8555-555555555555',
      stageMedia:
          ({
            required id,
            required bytes,
            required mimeType,
            cancelToken,
          }) async {
            events.add('stage:$id:$mimeType:${bytes.join(',')}');
          },
    );

    expect(events, [
      'stage:55555555-5555-4555-8555-555555555555:image/png:1,2,3',
    ]);
    expect(external.thumbMediaId, '55555555-5555-4555-8555-555555555555');
  });
}
