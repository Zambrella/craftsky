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
    final accountOperation = ActiveAccountOperation.capture(ref);
    state = const AsyncLoading();
    try {
      final repository = await accountOperation.step(
        () => ref.read(accountLocalPostDraftRepositoryProvider(account).future),
      );
      final saved = await accountOperation.step(() => repository.save(request));
      await accountOperation.step(
        () => ref.read(localPostDraftsProvider(account).notifier).refresh(),
      );
      if (!accountOperation.publish(() => state = AsyncData(saved))) {
        return null;
      }
      return saved;
    } on AccountOperationDiscarded {
      return null;
    } on Object catch (error, stackTrace) {
      if (accountOperation.isCurrent) {
        state = AsyncError(error, stackTrace);
      }
      rethrow;
    }
  }
}
