import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'deletion confirmation accepts only the exact captured handle label',
    () {
      const requiredHandle = '@alice.test';

      expect(
        matchesDeletionConfirmationHandle(
          requiredHandle: requiredHandle,
          input: '@alice.test',
        ),
        isTrue,
      );

      for (final mismatch in [
        'alice.test',
        '@Alice.test',
        '@alice.test ',
        ' @alice.test',
        'Alice',
        'did:plc:alice',
        '@alice.example',
        '@bob.test',
        '',
      ]) {
        expect(
          matchesDeletionConfirmationHandle(
            requiredHandle: requiredHandle,
            input: mismatch,
          ),
          isFalse,
          reason: '$mismatch must not authorize deletion',
        );
      }
    },
  );
}
