import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('production account invalidation resets mutation state first', () async {
    final container = ProviderContainer.test();
    final controller = container.read(pdsRecordOperationControllerProvider);
    final scope = PdsMutationScope(
      lease: AccountSessionLease(
        account: AccountKey('did:plc:account-boundary'),
        sessionGeneration: 1,
      ),
      identity: 'like:post-1',
    );
    final token = controller.begin(
      scope: scope,
      operationKey: 'operation',
      endpoint: '/mutation',
      immutableBody: '{}',
    );
    controller.markAmbiguous(token, retryAfterSeconds: 1);

    await container.read(accountStateInvalidatorProvider)();

    expect(controller.operationFor(scope), isNull);
  });
}
