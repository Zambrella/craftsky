import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/saved_posts/data/api_saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_api_client.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart'
    show Notifier, NotifierProvider;
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'saved_post_repository_provider.g.dart';

/// One non-family dependency shared by every private saved-post provider.
/// Invalidating it creates a hard boundary across all account-keyed instances.
final savedPostAccountBoundaryProvider =
    NotifierProvider<SavedPostAccountBoundary, int>(
      SavedPostAccountBoundary.new,
    );

final class SavedPostAccountBoundary extends Notifier<int> {
  @override
  int build() => 0;

  void advance() => state++;
}

int captureSavedPostAccountBoundary(Ref ref) =>
    ref.read(savedPostAccountBoundaryProvider);

bool isSavedPostAccountBoundaryCurrent(Ref ref, int generation) =>
    ref.mounted && ref.read(savedPostAccountBoundaryProvider) == generation;

@riverpod
Future<SavedPostRepository> accountSavedPostRepository(
  Ref ref,
  AccountKey account,
) async {
  ref.watch(savedPostAccountBoundaryProvider);
  return ApiSavedPostRepository(
    SavedPostApiClient(await ref.watch(accountDioProvider(account).future)),
  );
}
