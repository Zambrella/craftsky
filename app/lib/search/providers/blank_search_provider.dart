import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/search/models/blank_search_data.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'blank_search_provider.g.dart';

@riverpod
Future<BlankSearchData> blankSearch(Ref ref) async {
  final repo = ref.watch(searchRepositoryProvider);
  final recentSearches = await ref.watch(recentSearchPageProvider.future);
  final topHashtags = await repo.topHashtags(
    craftTypes: ProjectOptionCatalogs.defaultSupportedCraftTokens,
  );

  return BlankSearchData(
    recentSearches: recentSearches,
    topHashtags: topHashtags,
  );
}
