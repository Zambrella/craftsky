import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

ProviderContainer _container(SharedPreferences prefs) =>
    ProviderContainer.test(
      overrides: [sharedPreferencesProvider.overrideWithValue(prefs)],
    );

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  test('build returns false when no flag stored for this DID', () async {
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);
    expect(container.read(onboardingStatusProvider('did:plc:a')), isFalse);
  });

  test('build returns true when prefs has flag', () async {
    SharedPreferences.setMockInitialValues(
      {'flutter.onboarded_did:plc:a': true},
    );
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);
    expect(container.read(onboardingStatusProvider('did:plc:a')), isTrue);
  });

  test('finish writes flag and flips state for the DID', () async {
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);

    await container
        .read(onboardingStatusProvider('did:plc:a').notifier)
        .finish();

    expect(container.read(onboardingStatusProvider('did:plc:a')), isTrue);
    expect(prefs.getBool('onboarded_did:plc:a'), isTrue);
  });

  test('finish for one DID does not affect a different DID', () async {
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);

    await container
        .read(onboardingStatusProvider('did:plc:a').notifier)
        .finish();

    expect(container.read(onboardingStatusProvider('did:plc:a')), isTrue);
    expect(container.read(onboardingStatusProvider('did:plc:b')), isFalse);
  });
}
