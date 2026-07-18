import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-015 clean work passes and overlapping dirty checks coalesce',
    () async {
      final owner = AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 1,
      );
      final guard = UnsavedWorkGuard();
      var dirty = false;
      var confirmations = 0;
      final result = Completer<bool>();
      final registration = guard.register(
        owner: owner,
        isDirty: () => dirty,
        confirmAndClose: () {
          confirmations++;
          return result.future;
        },
      );

      expect(await guard.confirmLeave(owner), isTrue);
      dirty = true;
      final first = guard.confirmLeave(owner);
      final overlap = guard.confirmLeave(owner);
      expect(confirmations, 1);
      result.complete(false);
      expect(await first, isFalse);
      expect(await overlap, isFalse);

      guard.unregister(registration);
      expect(await guard.confirmLeave(owner), isTrue);
    },
  );
}
