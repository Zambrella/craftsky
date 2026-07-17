import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-003 / IT-006 exposes one shared HTTP adapter', () {
    final container = ProviderContainer.test(
      overrides: [dioProvider.overrideWithValue(Dio())],
    );

    final adapter = container.read(notificationApiRepositoryProvider);

    expect(container.read(notificationRepositoryProvider), same(adapter));
    expect(
      container.read(notificationNewnessRepositoryProvider),
      same(adapter),
    );
    expect(
      container.read(notificationPreferencesRepositoryProvider),
      same(adapter),
    );
    expect(container.read(notificationDeviceRepositoryProvider), same(adapter));
  });
}
