import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/settings/services/account_product_data_cleaner.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'production cleaner attempts every local cleanup and reports first error',
    () async {
      final calls = <String>[];
      final cleaner = AccountProductDataCleaner([
        (_) async {
          calls.add('draftsAndStagedMedia');
          throw StateError('draft cleanup failed');
        },
        (_) async => calls.add('instagramVerificationSnapshot'),
        (_) async => calls.add('imageCaches'),
      ]);
      final lease = AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 1,
      );

      await expectLater(cleaner.clean(lease), throwsStateError);

      expect(calls, [
        'draftsAndStagedMedia',
        'instagramVerificationSnapshot',
        'imageCaches',
      ]);
    },
  );
}
