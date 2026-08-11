import 'package:craftsky_app/auth/models/account_deletion.dart' as model;
import 'package:craftsky_app/auth/providers/deletion_status_registry_storage.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'deletion_status_registry_provider.g.dart';

@Riverpod(keepAlive: true)
DeletionStatusRegistryStorage deletionStatusRegistryStorage(Ref ref) =>
    SecureDeletionStatusRegistryStorage(const FlutterSecureStorage());

@Riverpod(keepAlive: true)
class DeletionStatusRegistry extends _$DeletionStatusRegistry {
  Future<void> _pendingMutation = Future.value();

  @override
  Future<model.DeletionStatusRegistry> build() =>
      ref.watch(deletionStatusRegistryStorageProvider).read();

  Future<void> upsert(model.DeletionStatusEntry entry) =>
      _mutate((current) => current.upsert(entry));

  Future<void> remove(String jobId) =>
      _mutate((current) => current.remove(jobId));

  Future<void> _mutate(
    model.DeletionStatusRegistry Function(
      model.DeletionStatusRegistry current,
    )
    transform,
  ) {
    final operation = _pendingMutation.then((_) async {
      final current = state.requireValue;
      final next = transform(current);
      if (identical(next, current)) return;
      await ref.read(deletionStatusRegistryStorageProvider).write(next);
      if (ref.mounted) state = AsyncData(next);
    });
    _pendingMutation = operation.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    return operation;
  }
}
