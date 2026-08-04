import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'draft_save_controller.g.dart';

@riverpod
class DraftSaveController extends _$DraftSaveController {
  @override
  FutureOr<LocalPostDraft?> build(AccountKey account) => null;

  Future<LocalPostDraft?> save(DraftWriteRequest request) async {
    if (request.owner != account) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.invalidRequest,
      );
    }
    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading();
    try {
      final repository = await ref.read(
        accountLocalPostDraftRepositoryProvider(account).future,
      );
      if (!isActiveAccountOperationCurrent(ref, ownership)) return null;
      final saved = await repository.save(request);
      if (!isActiveAccountOperationCurrent(ref, ownership)) return null;
      await ref.read(localPostDraftsProvider(account).notifier).refresh();
      if (!isActiveAccountOperationCurrent(ref, ownership)) return null;
      state = AsyncData(saved);
      return saved;
    } on Object catch (error, stackTrace) {
      if (isActiveAccountOperationCurrent(ref, ownership)) {
        state = AsyncError(error, stackTrace);
      }
      rethrow;
    }
  }
}
