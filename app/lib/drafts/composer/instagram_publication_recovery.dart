import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/drafts/composer/composer_draft_session.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/observability/consumed_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

InstagramPublicationCheckpoint? instagramPublicationCheckpoint(
  WidgetRef ref,
  ComposerDraftSession session,
  ActiveAccountLease? owner,
) {
  if (owner == null || session.draft?.instagramImport == null) return null;
  final operation = ActiveAccountOperation.fromWidget(ref, owner);
  return (publication) async {
    final repository = await operation.step(
      () => ref.read(
        accountLocalPostDraftRepositoryProvider(owner.session.account).future,
      ),
    );
    if (repository is! InstagramPublicationDraftRepository) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.storageUnavailable,
      );
    }
    await session.setInstagramPublication(
      repository: repository as InstagramPublicationDraftRepository,
      publication: publication,
    );
  };
}

/// A pending import can be closed or replayed without mutating its draft.
class InstagramPublicationRecovery extends ConsumerStatefulWidget {
  const new({
    required this.session,
    required this.owner,
    required this.onCleared,
    super.key,
  });

  final ComposerDraftSession session;
  final ActiveAccountLease owner;
  final VoidCallback onCleared;

  @override
  ConsumerState<InstagramPublicationRecovery> createState() =>
      _InstagramPublicationRecoveryState();
}

class _InstagramPublicationRecoveryState
    extends ConsumerState<InstagramPublicationRecovery> {
  bool _busy = false;

  Future<void> _retry() async {
    final pending = widget.session.draft?.instagramPublication;
    if (_busy || pending == null) return;
    final owner = widget.owner;
    final operation = ActiveAccountOperation.fromWidget(ref, owner);
    if (!operation.isCurrent) return;
    final l10n = AppLocalizations.of(context);
    final reporter = ref.read(errorReporterProvider);
    final notifier = ref.read(createPostProvider.notifier);
    final checkpoint = instagramPublicationCheckpoint(
      ref,
      widget.session,
      owner,
    )!;
    setState(() => _busy = true);
    try {
      final post = await notifier.retryInstagramPublication(
        pending,
        ownership: owner,
        clearCheckpoint: () => checkpoint(null),
      );
      if (!mounted || !operation.isCurrent) return;
      final error = ref.read(createPostProvider).error;
      notifier.reset();
      if (post == null) {
        context.showError(
          switch (error) {
            ApiBadRequest(code: 'instagram_verification_required') =>
              l10n.instagramPostImportVerificationRequired,
            ApiBadRequest(code: 'instagram_post_already_imported') =>
              l10n.instagramPostImportAlreadyPublished,
            _ when widget.session.draft?.instagramPublication == null =>
              l10n.postCreateError,
            _ => l10n.instagramPostImportPendingPublication,
          },
        );
        if (widget.session.draft?.instagramPublication == null) {
          widget.onCleared();
        }
        return;
      }
      try {
        await widget.session.deleteAfterSuccess(
          repositoryFor: (account) =>
              ref.read(accountLocalPostDraftRepositoryProvider(account).future),
          refresh: (account) async {
            if (operation.isCurrent) {
              ref.invalidate(localPostDraftsProvider(account));
            }
          },
        );
      } on Object catch (error, stack) {
        reportConsumedFailure(
          error,
          stack,
          reporter: reporter,
          feature: 'InstagramPostImport',
          operation: 'publication.cleanup',
          accountDid: owner.session.account.did.value,
        );
        if (mounted && operation.isCurrent) {
          context.showError(l10n.draftCleanupError);
        }
      }
      if (!mounted || !operation.isCurrent) return;
      context.showInfo(l10n.postCreateSuccess);
      Navigator.of(context).pop(post);
    } on Object catch (error, stack) {
      reportConsumedFailure(
        error,
        stack,
        reporter: reporter,
        feature: 'InstagramPostImport',
        operation: 'publication.retry',
        accountDid: owner.session.account.did.value,
      );
      if (mounted && operation.isCurrent) {
        context.showError(l10n.instagramPostImportPendingPublication);
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(createPostProvider);
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_busy,
      child: Scaffold(
        appBar: AppBar(
          title: Text(l10n.instagramPostImportRetryPublication),
        ),
        body: SafeArea(
          child: Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                spacing: 16,
                children: [
                  Text(l10n.instagramPostImportPendingPublication),
                  ChunkyButton(
                    onPressed: _busy ? null : _retry,
                    child: ActionProgress(
                      isLoading: _busy,
                      child: Text(l10n.instagramPostImportRetryPublication),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
