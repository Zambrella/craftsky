import 'package:craftsky_app/auth/providers/handle_typeahead_provider.dart';
import 'package:craftsky_app/auth/services/handle_typeahead_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('can inject a service without AppView or session providers', () {
    final service = HandleTypeaheadService();
    final container = ProviderContainer(
      overrides: [
        handleTypeaheadServiceProvider.overrideWithValue(service),
      ],
    );
    addTearDown(container.dispose);
    addTearDown(service.close);
    expect(container.read(handleTypeaheadServiceProvider), same(service));
  });

  test('container disposal closes the dedicated transport', () async {
    final container = ProviderContainer();
    final service = container.read(handleTypeaheadServiceProvider);
    container.dispose();
    await expectLater(
      service.search('alice'),
      throwsA(
        isA<DioException>().having(
          (error) => error.message,
          'message',
          contains('closed'),
        ),
      ),
    );
  });
}
