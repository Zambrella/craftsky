import 'package:craftsky_app/projects/options/project_option.dart';

enum HomeFeedSource { following, projects }

/// Stable identity and capabilities, independent of tab order and presentation.
/// Additional built-in sources and saved queries can join this catalog later.
class HomeFeedDefinition {
  const new following()
    : id = 'following',
      source = HomeFeedSource.following,
      craftType = null,
      label = null;

  new project(ProjectOption craft)
    : id = projectId(craft.value),
      source = HomeFeedSource.projects,
      craftType = craft.value,
      label = craft.label;

  static String projectId(String craft) => 'projects:$craft';

  final String id;
  final HomeFeedSource source;
  final String? craftType;
  final String? label;

  bool get supportsFilters => source == HomeFeedSource.projects;
  bool get supportsSort => source == HomeFeedSource.projects;
}
