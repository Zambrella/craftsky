import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/search/models/search_sort.dart';

// Keeps project discovery behind the same repository seam as other API areas.
// ignore: one_member_abstracts
abstract interface class ProjectRepository {
  Future<PostPage> listProjects({
    List<String>? craftTypes,
    SearchSort? sort,
    int? limit,
    String? cursor,
  });
}
