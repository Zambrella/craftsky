import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'search_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
SearchApiClient searchApiClient(Ref ref) =>
    SearchApiClient(ref.watch(dioProvider));
