import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'shared runner retries ambiguous responses with one frozen token',
    () async {
      final controller = PdsRecordOperationController();
      final token = controller.begin(
        scope: PdsMutationScope(
          lease: AccountSessionLease(
            account: AccountKey('did:plc:alice'),
            sessionGeneration: 1,
          ),
          identity: 'like:post',
        ),
        operationKey: newPdsMutationOperationKey(),
        endpoint: '/v1/likes',
        immutableBody: 'POST',
      );
      var sends = 0;
      final provider = Provider<Future<int>>(
        (ref) => runPdsMutation<int>(
          ref: ref,
          controller: controller,
          token: token,
          isCurrent: () => true,
          send: () async {
            sends++;
            if (sends == 1) {
              throw const PdsMutationAmbiguousException(retryAfterSeconds: 1);
            }
            return 42;
          },
        ),
      );
      final container = ProviderContainer.test(
        overrides: [
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
      );
      expect(await container.read(provider), 42);
      expect(sends, 2);
      expect(
        controller.operationFor(token.scope)?.token.operationKey,
        token.operationKey,
      );
    },
  );
  test('retry delay uses clamped server value, local backoff, and jitter', () {
    const policy = PdsMutationRetryPolicy();
    final expected = <Duration>[
      const Duration(milliseconds: 1250),
      const Duration(milliseconds: 2250),
      const Duration(milliseconds: 4250),
      const Duration(milliseconds: 5250),
      const Duration(milliseconds: 5250),
      const Duration(milliseconds: 5250),
    ];
    var elapsed = Duration.zero;
    for (var retry = 0; retry < expected.length; retry++) {
      final delay = policy.nextDelay(
        retryIndex: retry,
        retryAfterSeconds: retry == 0 ? 0 : null,
        elapsed: elapsed,
        jitterMillis: (_) => 250,
      );
      expect(delay, expected[retry]);
      elapsed += delay!;
    }
    expect(
      policy.nextDelay(
        retryIndex: 6,
        retryAfterSeconds: 1,
        elapsed: elapsed,
        jitterMillis: (_) => 0,
      ),
      isNull,
    );
  });

  test('automatic retry stops before crossing thirty seconds', () {
    const policy = PdsMutationRetryPolicy();
    expect(
      policy.nextDelay(
        retryIndex: 0,
        retryAfterSeconds: 5,
        elapsed: const Duration(seconds: 26),
        jitterMillis: (_) => 0,
      ),
      isNull,
    );
  });

  test('retry uses the greater of server guidance and local backoff', () {
    const policy = PdsMutationRetryPolicy();
    expect(
      policy.nextDelay(
        retryIndex: 2,
        retryAfterSeconds: 3,
        elapsed: Duration.zero,
        jitterMillis: (_) => 0,
      ),
      const Duration(seconds: 4),
    );
    expect(
      policy.nextDelay(
        retryIndex: 1,
        retryAfterSeconds: 5,
        elapsed: Duration.zero,
        jitterMillis: (_) => 0,
      ),
      const Duration(seconds: 5),
    );
  });
}
