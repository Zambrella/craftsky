import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/drafts/widgets/draft_row.dart';
import 'package:craftsky_app/drafts/widgets/draft_thumbnail.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final class DraftsPage extends ConsumerWidget {
  const DraftsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final activeLease = ref
        .watch(activeAccountInitializationProvider)
        .value
        ?.lease;
    final account = activeLease?.session.account;
    if (activeLease == null || account == null) {
      return Scaffold(
        appBar: AppBar(
          leading: BackButton(
            onPressed: () => const ProfileRoute().go(context),
          ),
          title: Text(l10n.draftsTitle),
        ),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    final drafts = ref.watch(localPostDraftsProvider(account));
    final repository = ref.watch(
      accountLocalPostDraftRepositoryProvider(account),
    );
    return Scaffold(
      appBar: AppBar(
        leading: BackButton(
          onPressed: () => const ProfileRoute().go(context),
        ),
        title: Text(l10n.draftsTitle),
      ),
      body: switch (drafts) {
        AsyncData(:final value) => DraftsPageContent(
          items: value.items,
          showTitle: false,
          onRefresh: ref
              .read(localPostDraftsProvider(account).notifier)
              .refresh,
          thumbnailBuilder: (draftId, mediaId) => repository.when(
            data: (value) => DraftThumbnail(
              repository: value,
              draftId: draftId,
              mediaId: mediaId,
            ),
            error: (_, _) => const Icon(Icons.broken_image_outlined),
            loading: () => const CircularProgressIndicator(strokeWidth: 2),
          ),
          onEdit: (draft) => _openDraft(
            context,
            ref,
            activeLease,
            repository,
            draft,
          ),
          onDelete: (draft) async {
            final confirmed = await showCraftskyConfirmDialog(
              context,
              title: l10n.draftsDeleteTitle,
              message: l10n.draftsDeleteMessage,
              confirmLabel: l10n.draftsDeleteAction,
              cancelLabel: l10n.languageCancel,
            );
            if (!confirmed || !_isActiveLeaseCurrent(ref, activeLease)) return;
            await ref
                .read(localPostDraftsProvider(account).notifier)
                .delete(draft.id);
          },
        ),
        AsyncError() => _DraftsError(
          onRetry: ref.read(localPostDraftsProvider(account).notifier).refresh,
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }

  Future<void> _openDraft(
    BuildContext context,
    WidgetRef ref,
    ActiveAccountLease activeLease,
    AsyncValue<LocalPostDraftRepository> repository,
    LocalPostDraft draft,
  ) async {
    if (!draft.canEdit) return;
    if (!repository.hasValue || !_isActiveLeaseCurrent(ref, activeLease)) {
      return;
    }
    final detail = await repository.requireValue.get(draft.id);
    if (!context.mounted || !_isActiveLeaseCurrent(ref, activeLease)) return;
    final seed = await const DraftComposerHydrator().hydrate(
      repository: repository.requireValue,
      draft: detail,
    );
    if (!context.mounted || !_isActiveLeaseCurrent(ref, activeLease)) return;
    if (detail.kind == LocalPostDraftKind.project) {
      await showProjectComposerSheet(
        context,
        draftSeed: seed,
        draftOwner: activeLease,
      );
    } else {
      await showPostComposerSheet(
        context,
        draftSeed: seed,
        draftOwner: activeLease,
      );
    }
    if (_isActiveLeaseCurrent(ref, activeLease)) {
      await ref
          .read(localPostDraftsProvider(activeLease.session.account).notifier)
          .refresh();
    }
  }
}

bool _isActiveLeaseCurrent(WidgetRef ref, ActiveAccountLease lease) =>
    ref.read(sessionRegistryProvider).value?.isCurrent(lease) ?? false;

final class DraftsPageContent extends StatelessWidget {
  const DraftsPageContent({
    required this.items,
    required this.onRefresh,
    required this.onEdit,
    required this.onDelete,
    this.thumbnailBuilder,
    this.showTitle = true,
    super.key,
  });

  final List<LocalPostDraft> items;
  final Future<void> Function() onRefresh;
  final Future<void> Function(LocalPostDraft draft) onEdit;
  final Future<void> Function(LocalPostDraft draft) onDelete;
  final Widget Function(String draftId, String mediaId)? thumbnailBuilder;
  final bool showTitle;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          if (showTitle)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 20, 16, 8),
              child: Text(
                l10n.draftsTitle,
                style: Theme.of(context).textTheme.headlineSmall,
              ),
            ),
          if (items.isEmpty)
            Padding(
              padding: const EdgeInsets.all(32),
              child: Center(child: Text(l10n.draftsEmpty)),
            )
          else
            for (final draft in items)
              DraftRow(
                draft: draft,
                onEdit: onEdit,
                onDelete: onDelete,
                thumbnailBuilder: thumbnailBuilder,
              ),
        ],
      ),
    );
  }
}

final class _DraftsError extends StatelessWidget {
  const _DraftsError({required this.onRetry});

  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.draftsLoadError),
          FilledButton(
            onPressed: onRetry,
            child: Text(l10n.draftsRetryAction),
          ),
        ],
      ),
    );
  }
}
