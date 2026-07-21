import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';

abstract interface class InstagramMigrationRepository {
  Future<InstagramVerificationAttempt> createVerification();

  Future<InstagramVerificationAttempt> getVerification(
    String verificationId,
  );

  Future<void> cancelVerification(String verificationId);

  Future<InstagramVerificationConfirmation> confirmVerification(
    String verificationId, {
    required bool discoverable,
  });

  Future<InstagramAccountStatus> getAccount();

  Future<InstagramAccountStatus> updateSettings(
    InstagramAccountSettingsPatch patch,
  );

  Future<void> revokeAccount();

  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  );

  Future<InstagramImportPage> listImports({int? limit, String? cursor});

  Future<InstagramImportSummary> getImport(String importId);

  Future<InstagramImportSummary> updateImport(
    String importId,
    InstagramImportPatch patch,
  );

  Future<void> deleteImport(String importId);

  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  });

  Future<InstagramSuggestionActionResult> acceptSuggestion(
    String suggestionId,
  );

  Future<void> dismissSuggestion(String suggestionId);
}
