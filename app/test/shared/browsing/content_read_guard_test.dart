import 'dart:async';

import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/content_read_guard.dart';
import 'package:flutter_test/flutter_test.dart';

// UT-005 / FR-007, NFR-004 / AC-007.
void main() {
  test(
    'late guest reads are discarded after the browsing epoch changes',
    () async {
      BrowsingContext current = const GuestBrowsingContext(epoch: 1);
      final guard = ContentReadGuard.withContext(current, () => current);
      final pending = Completer<int>();
      final result = guard.step(() => pending.future);
      current = const GuestBrowsingContext(epoch: 2);
      pending.complete(1);
      await expectLater(result, throwsA(isA<ContentReadDiscarded>()));
      var published = false;
      expect(guard.publish(() => published = true), false);
      expect(published, false);
    },
  );
  test('loading never dispatches a public read', () async {
    const context = BrowsingLoading(epoch: 1);
    final guard = ContentReadGuard.withContext(context, () => context);
    var dispatched = false;
    await expectLater(
      guard.step(() async {
        dispatched = true;
      }),
      throwsA(isA<ContentReadDiscarded>()),
    );
    expect(dispatched, false);
  });
}
