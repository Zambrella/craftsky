import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_verification_provider.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Instagram value models use generated deep equality and copyWith', () {
    final expiresAt = DateTime.utc(2026, 7, 27, 12);
    final attempt = InstagramVerificationAttempt(
      verificationId: 'verification-a',
      state: InstagramVerificationState.pendingConfirmation,
      expiresAt: expiresAt,
      candidateUsername: 'crafts.example',
    );
    final request = InstagramImportRequest(
      sourceType: InstagramImportSourceType.instagramJson,
      entries: const [InstagramImportEntry(username: 'crafts.example')],
    );
    final snapshot = InstagramVerificationSnapshot(
      verificationId: 'verification-a',
      challenge: 'CSKY-TEST',
      dmUrl: Uri.parse('https://instagram.example/direct'),
      expiresAt: expiresAt,
    );

    expect(
      request,
      InstagramImportRequest(
        sourceType: InstagramImportSourceType.instagramJson,
        entries: const [InstagramImportEntry(username: 'crafts.example')],
      ),
    );
    expect(
      attempt.copyWith(candidateUsername: null),
      InstagramVerificationAttempt(
        verificationId: 'verification-a',
        state: InstagramVerificationState.pendingConfirmation,
        expiresAt: expiresAt,
      ),
    );
    expect(
      snapshot.copyWith(challenge: 'CSKY-UPDATED').challenge,
      'CSKY-UPDATED',
    );
  });

  test('generated provider state copyWith can clear nullable values', () {
    final attempt = InstagramVerificationAttempt(
      verificationId: 'verification-a',
      state: InstagramVerificationState.pendingDm,
      expiresAt: DateTime.utc(2026, 7, 27, 12),
    );
    final verificationState = InstagramVerificationViewState(
      attempt: attempt,
      isBusy: true,
    );

    expect(
      verificationState.copyWith(attempt: null, isBusy: false),
      const InstagramVerificationViewState(),
    );
  });
}
