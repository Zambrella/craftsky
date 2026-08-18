import 'package:craftsky_app/instagram_migration/data/instagram_migration_api_client.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';

final class ApiInstagramMigrationRepository
    implements InstagramMigrationRepository {
  const ApiInstagramMigrationRepository(this._api);

  final InstagramMigrationApiClient _api;

  @override
  Future<InstagramVerificationAttempt> createVerification() =>
      _api.createVerification();

  @override
  Future<InstagramVerificationAttempt> getVerification(
    String verificationId,
  ) => _api.getVerification(verificationId);

  @override
  Future<InstagramVerificationAttempt?> getCurrentVerification() =>
      _api.getCurrentVerification();

  @override
  Future<void> cancelVerification(String verificationId) =>
      _api.cancelVerification(verificationId);

  @override
  Future<InstagramVerificationConfirmation> confirmVerification(
    String verificationId, {
    required bool discoverable,
  }) => _api.confirmVerification(
    verificationId,
    discoverable: discoverable,
  );

  @override
  Future<InstagramAccountStatus> getAccount() => _api.getAccount();

  @override
  Future<InstagramAccountStatus> updateSettings(
    InstagramAccountSettingsPatch patch,
  ) => _api.updateSettings(patch);

  @override
  Future<void> revokeAccount() => _api.revokeAccount();

  @override
  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  ) => _api.createImport(request);

  @override
  Future<InstagramImportPage> listImports({int? limit, String? cursor}) =>
      _api.listImports(limit: limit, cursor: cursor);

  @override
  Future<InstagramImportSummary> getImport(String importId) =>
      _api.getImport(importId);

  @override
  Future<InstagramImportSummary> updateImport(
    String importId,
    InstagramImportPatch patch,
  ) => _api.updateImport(importId, patch);

  @override
  Future<void> deleteImport(String importId) => _api.deleteImport(importId);

  @override
  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) => _api.listSuggestions(limit: limit, cursor: cursor);

  @override
  Future<InstagramSuggestionActionResult> acceptSuggestion(
    String suggestionId,
  ) => _api.acceptSuggestion(suggestionId);

  @override
  Future<void> dismissSuggestion(String suggestionId) =>
      _api.dismissSuggestion(suggestionId);
}
