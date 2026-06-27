import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/data/project_api_client.dart';
import 'package:craftsky_app/projects/data/project_repository.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';

class ApiProjectRepository implements ProjectRepository {
  const ApiProjectRepository(this._api);

  final ProjectApiClient _api;

  @override
  Future<PostPage> listProjects({
    required ProjectBrowseQuery query,
    int? limit,
    String? cursor,
  }) => _api.listProjects(query: query, limit: limit, cursor: cursor);
}
