import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/draft_storage_paths.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

typedef AccountProductDataCleanup =
    Future<void> Function(AccountSessionLease lease);

final accountProductDataCleanerProvider = Provider<AccountProductDataCleanup>(
  (ref) => (lease) async {
    Object? firstError;
    StackTrace? firstStackTrace;

    Future<void> attempt(Future<void> Function() action) async {
      try {
        await action();
      } on Object catch (error, stackTrace) {
        firstError ??= error;
        firstStackTrace ??= stackTrace;
      }
    }

    await attempt(() async {
      final documents = await getApplicationDocumentsDirectory();
      final paths = DraftStoragePaths(
        documentsRoot: documents.path,
        owner: lease.account,
      );
      final files = IoDraftFileStore();
      if (await files.isSymbolicLink(paths.accountRoot)) {
        throw const DraftFileStoreException(
          DraftFileStoreFailureReason.accessDenied,
        );
      }
      await files.deleteDirectory(paths.accountRoot);
    });
    await attempt(
      () =>
          ref.read(instagramVerificationStorageProvider).delete(lease.account),
    );
    await attempt(() async {
      await Future.wait([
        ref.read(profileImageCacheManagerProvider).emptyCache(),
        ref.read(feedImageCacheManagerProvider).emptyCache(),
      ]);
    });

    if (firstError != null) {
      Error.throwWithStackTrace(firstError!, firstStackTrace!);
    }
  },
);
