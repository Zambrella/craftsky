import 'dart:async';

import 'package:craftsky_app/notifications/services/notification_sign_out_cleanup.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-019 coalesces overlap but cleans a later same-DID session',
    () async {
      var deletes = 0;
      var bindingRemovals = 0;
      final firstRemovalStarted = Completer<void>();
      final releaseFirstRemoval = Completer<void>();
      final cleanup = NotificationSignOutCleanup(
        deleteProviderToken: () async => deletes++,
        removeRoutingBinding: (_) async {
          bindingRemovals++;
          if (bindingRemovals == 1) {
            firstRemovalStarted.complete();
            await releaseFirstRemoval.future;
          }
        },
      );

      final first = cleanup.run(
        did: 'did:plc:alice',
        confirmedLogout: true,
      );
      await firstRemovalStarted.future;
      final overlapping = cleanup.run(
        did: 'did:plc:alice',
        confirmedLogout: true,
      );
      releaseFirstRemoval.complete();
      await Future.wait([first, overlapping]);

      await cleanup.run(did: 'did:plc:alice', confirmedLogout: false);

      expect(deletes, 1);
      expect(bindingRemovals, 2);
    },
  );

  test(
    'AT-010 best-effort deletes token after failed or forced logout',
    () async {
      var bindingRemovals = 0;
      final cleanup = NotificationSignOutCleanup(
        deleteProviderToken: () async =>
            throw Exception('provider unavailable'),
        removeRoutingBinding: (_) async => bindingRemovals++,
      );

      await cleanup.run(did: 'did:plc:alice', confirmedLogout: false);

      expect(bindingRemovals, 1);
    },
  );
}
