import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/projects/data/project_api_client.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

Map<String, dynamic> _samplePost() => {
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/project',
  'cid': 'bafy_project',
  'rkey': 'project',
  'text': 'project',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
};

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() =>
      Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());

  test(
    'IT-014 ProjectApiClient sends project browse filters to /v1/projects',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/projects',
        (server) => server.reply(200, {
          'items': [_samplePost()],
          'cursor': 'opaque:next',
        }),
        queryParameters: {
          'craftType': [
            ProjectOptionCatalogs.knittingCraftToken,
            ProjectOptionCatalogs.crochetCraftToken,
          ],
          'projectType': ['social.craftsky.project.defs#garment'],
          'patternDifficulty': ['social.craftsky.feed.defs#beginner'],
          'color': ['blue'],
          'material': ['alpaca'],
          'designTag': ['social.craftsky.project.defs#stripes'],
          'projectTag': ['gift'],
          'sort': 'popular',
          'limit': '25',
          'cursor': 'opaque:start',
        },
      );

      final page = await ProjectApiClient(dio).listProjects(
        query: const ProjectBrowseQuery(
          craftTypes: [
            ProjectOptionCatalogs.knittingCraftToken,
            ProjectOptionCatalogs.crochetCraftToken,
          ],
          filters: ProjectBrowseFilters(
            projectType: ['social.craftsky.project.defs#garment'],
            patternDifficulty: ['social.craftsky.feed.defs#beginner'],
            color: ['blue'],
            material: ['alpaca'],
            designTag: ['social.craftsky.project.defs#stripes'],
            projectTag: ['gift'],
          ),
          sort: SearchSort.popular,
        ),
        limit: 25,
        cursor: 'opaque:start',
      );

      expect(page.cursor, 'opaque:next');
      expect(page.items.single, isA<Post>());
    },
  );
}
