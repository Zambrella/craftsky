import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Project discovery AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed; each call is wrapped in [unwrapApi].
class ProjectApiClient {
  const ProjectApiClient(this._dio);

  final Dio _dio;

  /// GET /v1/projects — project discovery feed.
  Future<PostPage> listProjects({
    List<String>? craftTypes,
    SearchSort? sort,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/projects',
      queryParameters: {
        if (craftTypes != null && craftTypes.isNotEmpty)
          'craftType': craftTypes,
        'sort': ?sort?.wireValue,
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
      options: Options(listFormat: ListFormat.multi),
    );
    return PostPageMapper.fromMap(res.data!);
  });
}
