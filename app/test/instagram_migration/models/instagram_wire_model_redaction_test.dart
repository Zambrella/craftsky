import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-010 every Instagram wire model has an opaque toString', () {
    final verification = InstagramVerificationAttempt.fromMap({
      'verificationId': 'synthetic-private-verification-id',
      'state': 'pendingConfirmation',
      'expiresAt': '2037-08-09T10:11:12Z',
      'candidateUsername': 'synthetic.private.username',
    });
    final account = InstagramAccountStatus.fromMap({
      'integrationAvailable': true,
      'account': {
        'state': 'active',
        'username': 'synthetic.private.username',
        'discoverable': true,
        'conflictPending': false,
        'reactivationRequired': false,
        'verifiedAt': '2037-08-09T10:11:12Z',
      },
    });
    final import = InstagramImportSummary.fromMap({
      'importId': 'synthetic-private-import-id',
      'state': 'active',
      'sourceType': 'instagramJson',
      'retainUnmatched': true,
      'retentionExpiresAt': '2037-08-09T10:11:12Z',
      'followingCount': 4242,
      'createdAt': '2036-08-09T10:11:12Z',
    });
    final suggestion = InstagramSuggestion.fromMap({
      'suggestionId': 'synthetic-private-suggestion-id',
      'profile': {
        'did': 'did:plc:synthetic-private-target',
        'handle': 'private-target.synthetic.invalid',
        'displayName': 'Synthetic Private Target',
      },
      'reason': 'verifiedInstagramFollow',
      'state': 'pending',
    });
    const privateValues = [
      'synthetic-private-verification-id',
      'synthetic.private.username',
      '2037-08-09',
      'synthetic-private-import-id',
      '4242',
      'synthetic-private-suggestion-id',
      'did:plc:synthetic-private-target',
      'private-target.synthetic.invalid',
      'Synthetic Private Target',
    ];

    for (final model in [verification, account, import, suggestion]) {
      expect(model.toString(), contains('[REDACTED]'));
      for (final privateValue in privateValues) {
        expect(model.toString(), isNot(contains(privateValue)));
      }
    }
  });
}
