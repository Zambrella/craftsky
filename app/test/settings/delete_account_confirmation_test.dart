import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-013 deletion confirmation accepts only the exact full DID',
    () {
      const confirmationDid = 'did:plc:alicefullidentifier';

      expect(
        matchesDeletionConfirmationDid(
          confirmationDid: confirmationDid,
          input: confirmationDid,
        ),
        isTrue,
      );

      for (final mismatch in [
        'did:plc:alicefullidentifieq',
        'did:plc:alicefullidentifier ',
        ' did:plc:alicefullidentifier',
        'DID:plc:alicefullidentifier',
        'did:plc:alice',
        '@alice.test',
        'did:plc:bob',
        '',
      ]) {
        expect(
          matchesDeletionConfirmationDid(
            confirmationDid: confirmationDid,
            input: mismatch,
          ),
          isFalse,
          reason: '$mismatch must not authorize deletion',
        );
      }
    },
  );
}
