import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'offline activation stays selected and resets navigation to Home',
    () async {
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-alice',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'token-bob',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final target = registry.leaseFor(AccountKey('did:plc:alice'))!;
      var location = '/profile/did:plc:bob';
      final coordinator = AccountActivationCoordinator(
        readRegistry: () => registry,
        commitActivation: (lease) async => registry = registry.activate(lease),
        invalidateAccountState: () async {},
        resetToHome: () async => location = RouteLocations.home,
      );

      await coordinator.activate(target);
      // A later content failure has no authority to roll activation back.
      final contentFailure = StateError('offline');

      expect(contentFailure, isA<StateError>());
      expect(registry.activeDid, 'did:plc:alice');
      expect(location, RouteLocations.home);
    },
  );
}
