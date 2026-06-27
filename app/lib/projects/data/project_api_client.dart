import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Project discovery AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed; each call is wrapped in [unwrapApi].
class ProjectApiClient {
  const ProjectApiClient(this._dio);

  final Dio _dio;

  /// GET /v1/projects — project discovery feed.
  Future<PostPage> listProjects({
    required ProjectBrowseQuery query,
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/projects',
      queryParameters: {
        if (query.craftTypes.isNotEmpty) 'craftType': query.craftTypes,
        ...query.filters.toQueryParameters(),
        'sort': query.sort.wireValue,
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
      options: Options(listFormat: ListFormat.multi),
    );
    return PostPageMapper.fromMap(res.data!);
  });
}
