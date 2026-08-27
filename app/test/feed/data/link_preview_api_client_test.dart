// Adapter registrations stay beside the assertions for each wire scenario.
// ignore_for_file: cascade_invocations

import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/create_post_external.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  // UT-014: Flutter strictly decodes preview bytes and round-trips external
  // create/full/compact wire shapes without loss.
  test('UT-014 preview and external wire models', () async {
    final exactBytes = Uint8List(1000000);
    exactBytes[0] = 1;
    exactBytes[999999] = 2;
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    addTearDown(() => dio.close(force: true));
    final adapter = DioAdapter(dio: dio);
    adapter.onPost(
      '/v1/link-previews',
      (server) => server.reply(200, {
        'url': 'https://final.example/pattern#section',
        'title': 'Pattern',
        'description': 'A knitting pattern',
        'thumbnail': {
          'bytes': base64Encode(exactBytes),
          'mimeType': 'image/png',
          'width': 1200,
          'height': 630,
        },
      }),
      data: {'url': 'https://source.example/pattern'},
    );

    final preview = await PostApiClient(
      dio,
    ).fetchLinkPreview('https://source.example/pattern');
    expect(preview.url.toString(), 'https://final.example/pattern#section');
    expect(preview.thumbnail?.bytes.length, 1000000);
    expect(preview.thumbnail?.bytes.first, 1);
    expect(preview.thumbnail?.bytes.last, 2);
    expect(preview.thumbnail?.mimeType, 'image/png');

    for (final encoded in ['***=', 'AQI', base64Encode(Uint8List(1000001))]) {
      adapter.onPost(
        '/v1/link-previews',
        (server) => server.reply(200, {
          'url': 'https://final.example',
          'title': 'Pattern',
          'description': '',
          'thumbnail': {
            'bytes': encoded,
            'mimeType': 'image/png',
            'width': 1,
            'height': 1,
          },
        }),
        data: {'url': 'https://source.example/$encoded'},
      );
      expect(
        () => PostApiClient(
          dio,
        ).fetchLinkPreview('https://source.example/$encoded'),
        throwsA(isA<FormatException>()),
      );
    }

    const blob = CreatePostBlob(
      ref: CreatePostBlobRef(link: 'bafythumb'),
      mimeType: 'image/png',
      size: 321,
    );
    const external = CreatePostExternal(
      uri: 'https://final.example/pattern#section',
      title: 'Pattern',
      description: 'A knitting pattern',
      thumb: blob,
    );
    adapter.onPost(
      '/v1/posts',
      (server) => server.reply(201, _postMap(external: _externalMap)),
      data: {
        'text': 'Pattern link',
        'langs': ['en'],
        'embed': {'external': external.toMap()},
      },
    );
    final created = await PostApiClient(
      dio,
    ).createPost(text: 'Pattern link', langs: const ['en'], external: external);
    expect(created.external?.thumb?.cid.toString(), 'bafythumb');
    expect(created.external?.uri, 'https://final.example/pattern#section');

    final compact = PostMapper.fromMap(
      _postMap(
        quoteView: {
          'state': 'visible',
          'post': {
            'uri': 'at://did:plc:bob/social.craftsky.feed.post/quoted',
            'cid': 'bafyquoted',
            'text': 'quoted',
            'createdAt': '2026-08-25T12:00:00Z',
            'author': {'did': 'did:plc:bob', 'handle': 'bob.example'},
            'external': _externalMap,
          },
        },
      ),
    );
    expect(compact.quoteView?.post?.external?.title, 'Pattern');
    expect(
      compact.quoteView?.post?.external?.thumb?.url,
      contains('bafythumb'),
    );
  });
}

const _externalMap = <String, dynamic>{
  'uri': 'https://final.example/pattern#section',
  'title': 'Pattern',
  'description': 'A knitting pattern',
  'thumb': {
    'cid': 'bafythumb',
    'mime': 'image/png',
    'size': 321,
    'url': 'https://cdn.example/bafythumb.png',
  },
};

Map<String, dynamic> _postMap({
  Map<String, dynamic>? external,
  Map<String, dynamic>? quoteView,
}) => {
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/post',
  'cid': 'bafypost',
  'rkey': 'post',
  'text': 'Pattern link',
  'tags': <String>[],
  'langs': <String>['en'],
  'likeCount': 0,
  'repostCount': 0,
  'quoteCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasReplied': false,
  'viewerHasSaved': false,
  'createdAt': '2026-08-25T12:00:00Z',
  'indexedAt': '2026-08-25T12:00:01Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.example'},
  'external': ?external,
  'quoteView': ?quoteView,
};
