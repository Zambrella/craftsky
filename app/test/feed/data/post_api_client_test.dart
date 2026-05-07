import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

  Map<String, dynamic> samplePost({String text = 'hello'}) {
    return {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
      'cid': 'bafy123',
      'rkey': '3lf2abc',
      'text': text,
      'tags': <String>[],
      'createdAt': '2026-05-04T18:23:45.000Z',
      'indexedAt': '2026-05-04T18:23:47.000Z',
      'author': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
      },
    };
  }

  group('PostApiClient.createPost', () {
    test('POSTs /v1/posts with text body and parses response', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'hi')),
        data: {'text': 'hi'},
      );

      final post = await PostApiClient(dio).createPost(text: 'hi');
      expect(post.text, 'hi');
      expect(post.rkey, '3lf2abc');
    });

    test('422 validation_failed surfaces as ApiBadRequest', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(422, {'error': 'validation_failed'}),
        data: {'text': ''},
      );

      await expectLater(
        () => PostApiClient(dio).createPost(text: ''),
        throwsA(
          isA<ApiBadRequest>().having(
            (e) => e.code,
            'code',
            'validation_failed',
          ),
        ),
      );
    });
  });

  group('PostApiClient.getPost', () {
    test('GETs /v1/posts/{did}/{rkey} and parses', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/3lf2abc',
        (server) => server.reply(200, samplePost()),
      );

      final post = await PostApiClient(dio).getPost('did:plc:alice', '3lf2abc');
      expect(post.rkey, '3lf2abc');
    });

    test('404 surfaces as ApiBadRequest(post_not_found)', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/missing',
        (server) => server.reply(404, {'error': 'post_not_found'}),
      );

      await expectLater(
        () => PostApiClient(dio).getPost('did:plc:alice', 'missing'),
        throwsA(
          isA<ApiBadRequest>().having((e) => e.code, 'code', 'post_not_found'),
        ),
      );
    });
  });

  group('PostApiClient.deletePost', () {
    test('DELETEs /v1/posts/{did}/{rkey} and returns on 204', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/posts/did:plc:alice/3lf2abc',
        (server) => server.reply(204, null),
      );

      await PostApiClient(dio).deletePost('did:plc:alice', '3lf2abc');
    });

    test('403 forbidden surfaces as ApiBadRequest', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/posts/did:plc:bob/3lf2abc',
        (server) => server.reply(403, {'error': 'forbidden'}),
      );

      await expectLater(
        () => PostApiClient(dio).deletePost('did:plc:bob', '3lf2abc'),
        throwsA(
          isA<ApiBadRequest>().having((e) => e.code, 'code', 'forbidden'),
        ),
      );
    });
  });

  group('PostApiClient.listPostsByAuthor', () {
    test('GETs /v1/profiles/@{handleOrDid}/posts (no cursor)', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/posts',
        (server) => server.reply(200, {
          'items': [samplePost()],
          'cursor': 'next-cursor',
        }),
      );

      final page = await PostApiClient(dio).listPostsByAuthor(
        'alice.craftsky.social',
      );
      expect(page.items, hasLength(1));
      expect(page.cursor, 'next-cursor');
    });

    test('passes cursor and limit as query params', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/posts',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        queryParameters: {'cursor': 'c1', 'limit': '50'},
      );

      final page = await PostApiClient(dio).listPostsByAuthor(
        'alice.craftsky.social',
        cursor: 'c1',
        limit: 50,
      );
      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    });
  });
}
