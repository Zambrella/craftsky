import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('onboardingStatusProvider', () {
    test('defaults to false', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      expect(container.read(onboardingStatusProvider), isFalse);
    });

    test('finish flips state to true', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(onboardingStatusProvider.notifier).finish();

      expect(container.read(onboardingStatusProvider), isTrue);
    });
  });
}
