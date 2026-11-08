import 'package:craftsky_app/settings/logged_out_visibility_controller.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-006 pending saves retain confirmed state and retry original identity',
    () async {
      final calls = <VisibilityIntent>[];
      var attempts = 0;
      final model = LoggedOutVisibilityController(
        read: () async => const VisibilitySetting(
          hideFromLoggedOut: false,
          expectedCid: 'old',
        ),
        save: (intent) async {
          calls.add(intent);
          if (++attempts == 1) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 1);
          }
          return const VisibilitySetting(
            hideFromLoggedOut: true,
            expectedCid: 'new',
          );
        },
        newKey: () => '11111111-1111-4111-8111-111111111111',
      );
      expect(model.value, isNull);
      await model.load();
      await model.setHidden(true);
      expect(model.value!.hideFromLoggedOut, false);
      expect(model.pending, isNotNull);
      await model.retry();
      expect(identical(calls[0], calls[1]), true);
      expect(model.value!.hideFromLoggedOut, true);
      expect(model.pending, isNull);
    },
  );
  test(
    'UT-006 pending acknowledgement is not displayed as a saved preference',
    () async {
      const intent = VisibilityIntent(
        hideFromLoggedOut: true,
        expectedCid: 'old',
        operationKey: 'original-key',
      );
      final model = LoggedOutVisibilityController(
        read: () async => const VisibilitySetting(
          hideFromLoggedOut: false,
          expectedCid: 'old',
          pending: intent,
        ),
        save: (_) async => const VisibilitySetting(
          hideFromLoggedOut: false,
          expectedCid: 'old',
          pending: intent,
        ),
      );
      await model.load();
      await model.retry();
      expect(model.pending, same(intent));
      expect(model.notice, isNull);
      expect(model.value!.hideFromLoggedOut, false);
    },
  );
}
