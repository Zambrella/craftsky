import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('authStatusProvider', () {
    test('defaults to false', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      expect(container.read(authStatusProvider), isFalse);
    });

    test('signIn flips state to true', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(authStatusProvider.notifier).signIn();

      expect(container.read(authStatusProvider), isTrue);
    });

    test('signOut flips state back to false', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(authStatusProvider.notifier).signIn();

      container.read(authStatusProvider.notifier).signOut();

      expect(container.read(authStatusProvider), isFalse);
    });
  });
}
