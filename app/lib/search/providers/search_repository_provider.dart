import 'package:craftsky_app/search/data/api_search_repository.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/providers/search_api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'search_repository_provider.g.dart';

@Riverpod(keepAlive: true)
SearchRepository searchRepository(Ref ref) =>
    ApiSearchRepository(ref.watch(searchApiClientProvider));
