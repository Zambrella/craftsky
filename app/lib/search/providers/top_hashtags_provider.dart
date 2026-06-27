import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'top_hashtags_provider.g.dart';

@riverpod
Future<TopHashtagsResponse> topHashtags(Ref ref, TopHashtagsQuery query) {
  return ref
      .watch(searchRepositoryProvider)
      .topHashtags(
        craftTypes: query.craftTypes,
        limit: query.limit,
      );
}
