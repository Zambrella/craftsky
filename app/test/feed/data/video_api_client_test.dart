import 'dart:convert';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test(
    'IT-014 downloads WebVTT through the authenticated AppView client',
    () async {
      final dio = Dio(
        BaseOptions(
          baseUrl: 'https://appview.example',
          headers: {'authorization': 'Bearer app-session'},
        ),
      );
      const route = '/v1/posts/did:plc:alice/rk1/video-captions/bafkcaption';
      DioAdapter(dio: dio).onGet(
        route,
        (server) => server.reply(
          200,
          utf8.encode('WEBVTT\n\n00:00.000 --> 00:01.000\nHello'),
        ),
        headers: {'authorization': 'Bearer app-session'},
      );

      final captions = await PostApiClient(dio).downloadVideoCaption(route);

      expect(captions, startsWith('WEBVTT'));
    },
  );

  test('IT-013 parses limits and exact authorization response', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example'));
    DioAdapter(dio: dio)
      ..onGet(
        '/v1/blobs/videos/limits',
        (server) => server.reply(200, {
          'canUpload': true,
          'remainingDailyVideos': 1,
          'remainingDailyBytes': 299999999,
        }),
      )
      ..onPost(
        '/v1/blobs/videos/authorization',
        (server) => server.reply(200, {
          'token': 'ephemeral-only',
          'expiresAt': '2030-01-01T00:00:00Z',
        }),
      );
    final api = PostApiClient(dio);

    final limits = await api.getVideoUploadLimits();
    final authorization = await api.authorizeVideoUpload();

    expect(limits.canUpload, isTrue);
    expect(limits.shouldShowQuota, isTrue);
    expect(authorization.expiresAt, DateTime.utc(2030));
    expect(authorization.toString(), isNot(contains('ephemeral-only')));
  });

  test('IT-005 serializes final proof only under embed.video', () {
    const proof = CreatePostVideo(
      jobId: 'job',
      blob: CreatePostVideoBlob(
        cid: 'bafyvideo',
        mimeType: 'video/mp4',
        size: 10,
      ),
      alt: 'A loom in motion',
      aspectRatio: CreatePostVideoAspectRatio(width: 16, height: 9),
    );

    expect(proof.toMap(), {
      'jobId': 'job',
      'blob': {
        r'$type': 'blob',
        'ref': {r'$link': 'bafyvideo'},
        'mimeType': 'video/mp4',
        'size': 10,
      },
      'alt': 'A loom in motion',
      'aspectRatio': {'width': 16, 'height': 9},
    });
    expect(proof.toString(), 'CreatePostVideo(<redacted>)');
  });
}
