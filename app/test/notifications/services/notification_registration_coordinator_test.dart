import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final alice = Did.parse('did:plc:alice');

  group('UT-013 notification registration coordinator', () {
    test(
      'defers refreshes and registers only the latest token when ready',
      () async {
        final calls = <String>[];
        final saved = <AccountSubscriptionId>[];
        final coordinator = NotificationRegistrationCoordinator(
          platform: NotificationPlatform.android,
          getToken: () async => null,
          register: ({required platform, required token}) async {
            calls.add(token);
            return AccountSubscriptionId.parse('binding_${calls.length}');
          },
          saveBinding: ({required did, required binding}) async {
            expect(did, alice);
            saved.add(binding);
          },
        );

        await coordinator.onTokenRefresh('tokenA');
        await coordinator.onTokenRefresh('tokenB');
        expect(calls, isEmpty);

        await coordinator.onReadinessChanged(did: alice, eligible: true);

        expect(calls, ['tokenB']);
        expect(saved, hasLength(1));
      },
    );

    test(
      'skips empty tokens and retries failure only on a later trigger',
      () async {
        var currentToken = '';
        var attempts = 0;
        final coordinator = NotificationRegistrationCoordinator(
          platform: NotificationPlatform.ios,
          getToken: () async => currentToken,
          register: ({required platform, required token}) async {
            attempts++;
            if (attempts == 1) throw Exception('transient');
            return AccountSubscriptionId.parse('binding_ok');
          },
          saveBinding: ({required did, required binding}) async {},
        );

        await coordinator.onReadinessChanged(did: alice, eligible: true);
        expect(attempts, 0);

        currentToken = 'tokenC';
        await coordinator.retry();
        expect(attempts, 1);

        await Future<void>.delayed(Duration.zero);
        expect(
          attempts,
          1,
          reason: 'failure must not create a tight retry loop',
        );

        await coordinator.retry();
        expect(attempts, 2);
      },
    );

    test('never registers after readiness becomes ineligible', () async {
      var calls = 0;
      final coordinator = NotificationRegistrationCoordinator(
        platform: NotificationPlatform.android,
        getToken: () async => 'token',
        register: ({required platform, required token}) async {
          calls++;
          return AccountSubscriptionId.parse('binding');
        },
        saveBinding: ({required did, required binding}) async {},
      );

      await coordinator.onReadinessChanged(did: alice, eligible: false);
      await coordinator.onTokenRefresh('newToken');

      expect(calls, 0);
    });
  });
}
