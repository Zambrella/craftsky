import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../profile/fakes/fake_profile_repository.dart';

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  test(
    'AT-008 identity save preserves an unsaved craft draft in session',
    () async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final baseline = Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        displayName: 'Alice',
        crafts: const ['sewing'],
      );
      List<String>? submittedCrafts;
      final repository = FakeProfileRepository(
        onFetchMe: () async => baseline,
        onUpdateMe:
            ({
              displayName,
              description,
              crafts,
              avatar,
              clearAvatar = false,
              banner,
              clearBanner = false,
            }) async {
              submittedCrafts = crafts;
              return Profile(
                did: baseline.did,
                handle: baseline.handle,
                displayName: displayName,
                crafts: const ['sewing'],
              );
            },
      );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountProfileRepositoryProvider.overrideWith(
            (ref, lease) async => repository,
          ),
        ],
      );
      final provider = onboardingFlowProvider(registry.activeLease!);
      final prefilled = Completer<void>();
      final subscription = container.listen(provider, (_, next) {
        if ((next.value?.baseline.crafts.contains('sewing') ?? false) &&
            !prefilled.isCompleted) {
          prefilled.complete();
        }
      });
      addTearDown(subscription.close);
      await container.read(provider.future);
      await prefilled.future.timeout(const Duration(seconds: 1));
      final notifier = container.read(provider.notifier);

      await (notifier
            ..next()
            ..toggleCraft('quilting')
            ..previous()
            ..updateIdentity(displayName: 'Alicia'))
          .saveAndNext();

      final state = container.read(provider).requireValue;
      expect(submittedCrafts, ['sewing']);
      expect(state.step, OnboardingStep.crafts);
      expect(state.selectedCraftIds, {'sewing', 'quilting'});
      expect(state.craftsDirty, isTrue);
    },
  );
}
