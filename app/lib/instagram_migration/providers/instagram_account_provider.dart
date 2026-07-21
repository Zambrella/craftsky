import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'instagram_account_provider.g.dart';

@riverpod
class InstagramAccount extends _$InstagramAccount {
  @override
  Future<InstagramAccountStatus> build(ActiveAccountLease lease) async {
    final repository = await ref.watch(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    final result = await repository.getAccount();
    ensureInstagramOperationCurrent(ref, lease);
    return result;
  }

  Future<bool> refresh() => _replace((repository) => repository.getAccount());

  Future<bool> setDiscoverable({required bool value}) => _replace(
    (repository) => repository.updateSettings(
      InstagramAccountSettingsPatch(discoverable: value),
    ),
  );

  Future<bool> reactivate() => _replace(
    (repository) => repository.updateSettings(
      const InstagramAccountSettingsPatch(reactivate: true),
    ),
  );

  Future<bool> revoke() async {
    final previous = state;
    state = const AsyncLoading();
    try {
      final repository = await _repository();
      await repository.revokeAccount();
      ensureInstagramOperationCurrent(ref, lease);
      state = AsyncData(
        InstagramAccountStatus(
          integrationAvailable: previous.value?.integrationAvailable ?? true,
          account: null,
        ),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object catch (error, stackTrace) {
      if (!ref.mounted || !_isCurrent) return false;
      state = AsyncError(error, stackTrace);
      return false;
    }
  }

  Future<bool> _replace(
    Future<InstagramAccountStatus> Function(
      InstagramMigrationRepository repository,
    )
    action,
  ) async {
    state = const AsyncLoading();
    try {
      final repository = await _repository();
      final result = await action(repository);
      ensureInstagramOperationCurrent(ref, lease);
      state = AsyncData(result);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object catch (error, stackTrace) {
      if (!ref.mounted || !_isCurrent) return false;
      state = AsyncError(error, stackTrace);
      return false;
    }
  }

  Future<InstagramMigrationRepository> _repository() async {
    final repository = await ref.read(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    return repository;
  }

  bool get _isCurrent {
    try {
      ensureInstagramOperationCurrent(ref, lease);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    }
  }
}
