import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';

// Keeps project discovery behind the same repository seam as other API areas.
// ignore: one_member_abstracts
abstract interface class ProjectRepository {
  Future<PostPage> listProjects({
    required ProjectBrowseQuery query,
    int? limit,
    String? cursor,
  });
}
