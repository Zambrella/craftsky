import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/rich_text/data/appview_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  Dio buildDio() =>
      Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());

  test(
    'UT-006 maps mention suggestions to AppView endpoint and decodes items',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/facets/mentions',
        (server) => server.reply(200, {
          'items': [
            {
              'did': 'did:plc:alice',
              'handle': 'alice.craftsky.social',
              'displayName': 'Alice',
              'isCraftskyProfile': true,
              'viewerIsFollowing': true,
            },
            {
              'did': 'did:plc:alicia',
              'handle': 'alicia.craftsky.social',
              'displayName': null,
              'avatar': null,
            },
          ],
        }),
        queryParameters: {'q': 'ali', 'limit': 10},
      );

      final items = await AppViewAccountSuggestionRepository(
        dio,
      ).searchAccounts('ali');

      expect(items, hasLength(2));
      expect(items.first.did, 'did:plc:alice');
      expect(items.first.handle, 'alice.craftsky.social');
      expect(items.first.displayName, 'Alice');
      expect(items.first.avatar, isNull);
      expect(items.first.isCraftskyProfile, isTrue);
      expect(items.first.viewerIsFollowing, isTrue);
      expect(items.last.did, 'did:plc:alicia');
      expect(items.last.isCraftskyProfile, isFalse);
      expect(items.last.viewerIsFollowing, isFalse);
    },
  );

  test(
    'UT-008 maps hashtag suggestions to AppView endpoint and decodes counts',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/facets/hashtags',
        (server) => server.reply(200, {
          'items': [
            {'tag': 'sockkal', 'postsLast28Days': 12},
            {'tag': 'knitting'},
          ],
        }),
        queryParameters: {'q': 'sock', 'limit': 10},
      );

      final items = await AppViewHashtagSuggestionRepository(
        dio,
      ).searchHashtags('sock');

      expect(items, hasLength(2));
      expect(items.first.tag, 'sockkal');
      expect(items.first.postsLast28Days, 12);
      expect(items.last.tag, 'knitting');
      expect(items.last.postsLast28Days, 0);
    },
  );

  test(
    'UT-007 exact resolve maps success and mention_not_found to final facets',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio)
        ..onGet(
          '/v1/facets/mentions/resolve',
          (server) => server.reply(200, {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'isCraftskyProfile': true,
          }),
          queryParameters: {'handle': 'alice.craftsky.social'},
        )
        ..onGet(
          '/v1/facets/mentions/resolve',
          (server) => server.reply(404, {
            'error': 'mention_not_found',
            'message': 'mention not found',
            'requestId': 'req-1',
          }),
          queryParameters: {'handle': 'unknown.example'},
        );

      final repository = AppViewAccountSuggestionRepository(dio);
      final facets = await FacetGenerator(
        mentionResolver: repository,
      ).generate('Thanks @alice.craftsky.social and @unknown.example');

      final mentions = facets
          .where(
            (facet) =>
                ((facet['features']! as List).single
                    as Map<String, dynamic>)[r'$type'] ==
                'app.bsky.richtext.facet#mention',
          )
          .toList();
      expect(mentions, hasLength(1));
      expect(
        ((mentions.single['features']! as List).single
            as Map<String, dynamic>)['did'],
        'did:plc:alice',
      );
    },
  );
}
