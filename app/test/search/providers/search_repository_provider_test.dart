import 'package:craftsky_app/search/data/api_search_repository.dart';
import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/search/providers/search_api_client_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  test('IT-001 production providers use shared Dio and are overrideable', () {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    final container = ProviderContainer.test(
      overrides: [dioProvider.overrideWithValue(dio)],
    );

    expect(container.read(searchApiClientProvider), isA<SearchApiClient>());
    expect(
      container.read(searchRepositoryProvider),
      isA<ApiSearchRepository>(),
    );

    final fake = FakeSearchRepository();
    final overridden = ProviderContainer.test(
      overrides: [searchRepositoryProvider.overrideWithValue(fake)],
    );
    expect(identical(overridden.read(searchRepositoryProvider), fake), isTrue);
  });
}
