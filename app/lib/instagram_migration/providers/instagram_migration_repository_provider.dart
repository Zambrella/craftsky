import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/instagram_migration/data/api_instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_api_client.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'instagram_migration_repository_provider.g.dart';

/// Internal signal used to discard a completion owned by a departed account
/// activation. It intentionally carries no account or Instagram data.
final class InstagramOperationDiscarded implements Exception {
  const InstagramOperationDiscarded();

  @override
  String toString() => 'InstagramOperationDiscarded()';
}

void ensureInstagramOperationCurrent(Ref ref, ActiveAccountLease lease) {
  if (!isActiveAccountOperationCurrent(ref, lease)) {
    throw const InstagramOperationDiscarded();
  }
}

@riverpod
Future<InstagramMigrationRepository> instagramMigrationRepository(
  Ref ref,
  ActiveAccountLease lease,
) async {
  final dio = await ref.watch(
    accountDioProvider(lease.session.account).future,
  );
  ensureInstagramOperationCurrent(ref, lease);
  return ApiInstagramMigrationRepository(InstagramMigrationApiClient(dio));
}
