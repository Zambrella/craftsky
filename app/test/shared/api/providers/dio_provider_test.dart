import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('dioProvider builds with the debug-default base URL', () {
    final container = ProviderContainer.test();

    final dio = container.read(dioProvider);

    expect(dio.options.baseUrl, 'http://10.0.2.2:8080');
    // Dio adds ImplyContentTypeInterceptor by default, plus our
    // SessionAuthInterceptor + ErrorMappingInterceptor (total = 3).
    // Task 14b adds SignOutOn401Interceptor (total = 4).
    expect(dio.interceptors, hasLength(3));
  });
}
