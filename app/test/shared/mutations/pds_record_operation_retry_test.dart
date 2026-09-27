import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
