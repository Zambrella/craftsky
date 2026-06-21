import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/data/project_api_client.dart';
import 'package:craftsky_app/projects/data/project_repository.dart';
import 'package:craftsky_app/search/models/search_sort.dart';

class ApiProjectRepository implements ProjectRepository {
  const ApiProjectRepository(this._api);

  final ProjectApiClient _api;

  @override
  Future<PostPage> listProjects({
    List<String>? craftTypes,
    SearchSort? sort,
    int? limit,
    String? cursor,
  }) => _api.listProjects(
    craftTypes: craftTypes,
    sort: sort,
    limit: limit,
    cursor: cursor,
  );
}
