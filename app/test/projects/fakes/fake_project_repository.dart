import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/data/project_repository.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';

class FakeProjectRepository implements ProjectRepository {
  FakeProjectRepository({this.onListProjects});

  final Future<PostPage> Function({
    required ProjectBrowseQuery query,
    int? limit,
    String? cursor,
  })?
  onListProjects;

  @override
  Future<PostPage> listProjects({
    required ProjectBrowseQuery query,
    int? limit,
    String? cursor,
  }) =>
      onListProjects?.call(query: query, limit: limit, cursor: cursor) ??
      Future<PostPage>.error(UnimplementedError('listProjects not stubbed'));
}
