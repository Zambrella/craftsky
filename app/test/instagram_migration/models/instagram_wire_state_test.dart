import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-014 decodes every closed wire state with a safe unknown', () {
    expect(
      {
        for (final wire in [
          'pendingDm',
          'processing',
          'pendingConfirmation',
          'confirmed',
          'expired',
          'cancelled',
          'superseded',
          'rejected',
          'conflicted',
        ])
          InstagramVerificationState.fromWire(wire).name,
      },
      containsAll(
        InstagramVerificationState.values
            .where((state) => state != InstagramVerificationState.unknown)
            .map((state) => state.name),
      ),
    );
    expect(
      InstagramVerificationState.fromWire('future'),
      InstagramVerificationState.unknown,
    );

    expect(
      [
        'active',
        'membershipInactive',
        'revoked',
        'superseded',
        'disputed',
      ].map(InstagramAccountLinkState.fromWire).toSet(),
      InstagramAccountLinkState.values
          .where((state) => state != InstagramAccountLinkState.unknown)
          .toSet(),
    );
    expect(
      InstagramAccountLinkState.fromWire('future'),
      InstagramAccountLinkState.unknown,
    );

    expect(
      [
        'active',
        'membershipInactive',
        'expired',
      ].map(InstagramImportState.fromWire).toSet(),
      InstagramImportState.values
          .where((state) => state != InstagramImportState.unknown)
          .toSet(),
    );
    expect(
      InstagramImportState.fromWire('future'),
      InstagramImportState.unknown,
    );

    expect(
      [
        'pending',
        'accepting',
        'accepted',
        'alreadyFollowing',
        'dismissed',
        'invalidated',
      ].map(InstagramSuggestionState.fromWire).toSet(),
      InstagramSuggestionState.values
          .where((state) => state != InstagramSuggestionState.unknown)
          .toSet(),
    );
    expect(
      InstagramSuggestionState.fromWire('future'),
      InstagramSuggestionState.unknown,
    );
    expect(
      InstagramSuggestionReason.fromWire('future'),
      InstagramSuggestionReason.unknown,
    );
  });
}
