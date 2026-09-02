import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('copying through navigation preserves in-session drafts', () {
    final baseline = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const ['sewing', 'future-craft'],
      displayName: 'Alice',
    );
    final initial = OnboardingFlowState.fromProfile(baseline);
    final edited = initial.copyWith(
      identity: initial.identity.copyWith(displayName: 'Alicia'),
      selectedCraftIds: const {'quilting'},
      step: OnboardingStep.instagram,
    );
    final returned = edited.copyWith(step: OnboardingStep.profile);

    expect(returned.identity.displayName, 'Alicia');
    expect(returned.selectedCraftIds, {'quilting'});
    expect(returned.unknownCraftIds, ['future-craft']);
    expect(returned.identityDirty, isTrue);
    expect(returned.craftsDirty, isTrue);
  });

  test('a reconstructed state starts on step one from persisted profile', () {
    final baseline = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const ['sewing'],
      displayName: 'Persisted',
    );
    final reconstructed = OnboardingFlowState.fromProfile(baseline);
    expect(reconstructed.step, OnboardingStep.profile);
    expect(reconstructed.identity.displayName, 'Persisted');
    expect(reconstructed.identityDirty, isFalse);
  });
}
