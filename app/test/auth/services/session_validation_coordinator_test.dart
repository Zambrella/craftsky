import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'validates active first then inactive accounts with concurrency two',
    () async {
      AccountSessionLease lease(String name, int generation) =>
          AccountSessionLease(
            account: AccountKey('did:plc:$name'),
            sessionGeneration: generation,
          );
      final active = lease('alice', 1);
      final inactive = [lease('bob', 2), lease('carol', 3), lease('dave', 4)];
      final releases =
          <AccountSessionLease, Completer<SessionValidationResult>>{};
      final started = <AccountSessionLease>[];
      var concurrent = 0;
      var maxConcurrent = 0;
      final coordinator = SessionValidationCoordinator(
        validate: (target) async {
          started.add(target);
          concurrent++;
          if (concurrent > maxConcurrent) maxConcurrent = concurrent;
          final completer = Completer<SessionValidationResult>();
          releases[target] = completer;
          final result = await completer.future;
          concurrent--;
          return result;
        },
        applyResult: (_, _) async {},
      );

      final run = coordinator.run(active: active, inactive: inactive);
      await Future<void>.delayed(Duration.zero);
      expect(started, [active]);

      releases[active]!.complete(SessionValidationResult.valid);
      await Future<void>.delayed(Duration.zero);
      expect(started, [active, inactive[0], inactive[1]]);
      expect(maxConcurrent, 2);

      releases[inactive[0]]!.complete(SessionValidationResult.transientFailure);
      await Future<void>.delayed(Duration.zero);
      expect(started, [active, ...inactive]);
      releases[inactive[1]]!.complete(SessionValidationResult.unauthorized);
      releases[inactive[2]]!.complete(SessionValidationResult.identityMismatch);
      await run;
      expect(maxConcurrent, 2);
    },
  );
}
