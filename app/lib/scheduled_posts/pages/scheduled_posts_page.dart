import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post_row_model.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_posts_provider.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_media_thumbnail.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ScheduledPostsPage extends ConsumerWidget {
  const ScheduledPostsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final activeLease = ref.watch(sessionRegistryProvider).value?.activeLease;
    final account = activeLease?.session.account;
    if (activeLease == null || account == null) {
      return Scaffold(
        appBar: AppBar(
          leading: BackButton(
            onPressed: () => const ProfileRoute().go(context),
          ),
          title: Text(l10n.scheduledPostsTitle),
        ),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    final scheduled = ref.watch(scheduledPostsProvider(account));
    return Scaffold(
      appBar: AppBar(
        leading: BackButton(
          onPressed: () => const ProfileRoute().go(context),
        ),
        title: Text(l10n.scheduledPostsTitle),
      ),
      body: switch (scheduled) {
        AsyncData(:final value) => ScheduledPostsPageContent(
          items: value.items,
          showTitle: false,
          onRefresh: ref.read(scheduledPostsProvider(account).notifier).refresh,
          thumbnailBuilder: (mediaId) => ScheduledMediaThumbnail(
            account: account,
            mediaId: mediaId,
          ),
          onEdit: (item) async {
            final repository = await ref.read(
              accountScheduledPostRepositoryProvider(account).future,
            );
            if (!_isActiveLeaseCurrent(ref, activeLease)) return;
            final detail = await repository.get(item.id);
            if (!context.mounted || !_isActiveLeaseCurrent(ref, activeLease)) {
              return;
            }
            if (item.kind == ScheduledPostKind.project) {
              await showProjectComposerSheet(
                context,
                scheduledPost: detail,
                scheduledOwner: activeLease,
              );
            } else {
              await showPostComposerSheet(
                context,
                scheduledPost: detail,
                scheduledOwner: activeLease,
              );
            }
            if (!_isActiveLeaseCurrent(ref, activeLease)) return;
            await ref.read(scheduledPostsProvider(account).notifier).refresh();
          },
          onDelete: (item) async {
            final confirmed = await showCraftskyConfirmDialog(
              context,
              title: l10n.scheduledPostsDeleteTitle,
              message: l10n.scheduledPostsDeleteMessage,
              confirmLabel: l10n.scheduledPostsDeleteAction,
              cancelLabel: l10n.languageCancel,
            );
            if (!confirmed || !_isActiveLeaseCurrent(ref, activeLease)) return;
            final repository = await ref.read(
              accountScheduledPostRepositoryProvider(account).future,
            );
            if (!_isActiveLeaseCurrent(ref, activeLease)) return;
            await repository.delete(item.id);
            if (!_isActiveLeaseCurrent(ref, activeLease)) return;
            await ref.read(scheduledPostsProvider(account).notifier).refresh();
          },
        ),
        AsyncError() => _ScheduledPostsError(
          onRetry: ref.read(scheduledPostsProvider(account).notifier).refresh,
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

bool _isActiveLeaseCurrent(WidgetRef ref, ActiveAccountLease lease) =>
    ref.read(sessionRegistryProvider).value?.isCurrent(lease) ?? false;

class ScheduledPostsPageContent extends StatelessWidget {
  const ScheduledPostsPageContent({
    required this.items,
    required this.onRefresh,
    required this.onEdit,
    required this.onDelete,
    this.thumbnailBuilder,
    this.showTitle = true,
    super.key,
  });

  final List<ScheduledPostSummary> items;
  final Future<void> Function() onRefresh;
  final Future<void> Function(ScheduledPostSummary item) onEdit;
  final Future<void> Function(ScheduledPostSummary item) onDelete;
  final Widget Function(String mediaId)? thumbnailBuilder;
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
                l10n.scheduledPostsTitle,
                style: Theme.of(context).textTheme.headlineSmall,
              ),
            ),
          if (items.isEmpty)
            CraftskyEmptyState(
              icon: CraftskyIcons.schedule,
              title: l10n.scheduledPostsTitle,
              subtitle: l10n.scheduledPostsEmpty,
            )
          else
            for (final item in items)
              _ScheduledPostTile(
                item: item,
                onEdit: onEdit,
                onDelete: onDelete,
                thumbnailBuilder: thumbnailBuilder,
              ),
        ],
      ),
    );
  }
}

class _ScheduledPostTile extends StatelessWidget {
  const _ScheduledPostTile({
    required this.item,
    required this.onEdit,
    required this.onDelete,
    this.thumbnailBuilder,
  });

  final ScheduledPostSummary item;
  final Future<void> Function(ScheduledPostSummary item) onEdit;
  final Future<void> Function(ScheduledPostSummary item) onDelete;
  final Widget Function(String mediaId)? thumbnailBuilder;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final locked = item.status == ScheduledPostStatus.publishing;
    final local = item.scheduledAt.utc.toLocal();
    final row = ScheduledPostRowModel.fromSummary(
      item,
      zoneName: local.timeZoneName,
      offset: local.timeZoneOffset,
    );
    final date = MaterialLocalizations.of(
      context,
    ).formatMediumDate(row.time.wallTime);
    final time = MaterialLocalizations.of(
      context,
    ).formatTimeOfDay(TimeOfDay.fromDateTime(row.time.wallTime));
    return ListTile(
      onTap: locked ? null : () => onEdit(item),
      leading: row.firstMediaId == null
          ? const Icon(CraftskyIcons.schedule)
          : thumbnailBuilder?.call(row.firstMediaId!) ??
                const Icon(CraftskyIcons.image),
      title: Text(
        row.projectTitle ?? row.preview,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (row.projectTitle != null)
            Text(
              row.preview,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
          Text(
            l10n.scheduledPostsRowDateTime(
              row.kind == ScheduledPostKind.project
                  ? l10n.scheduledPostsKindProject
                  : l10n.scheduledPostsKindStandard,
              date,
              '$time · ${row.time.zoneLabel}',
            ),
          ),
          Semantics(
            liveRegion: true,
            child: Text(_statusLabel(l10n, item.status)),
          ),
          if (item.needsAttentionExpiresAt case final expiresAt?)
            Text(
              l10n.scheduledPostsDeletedOn(
                MaterialLocalizations.of(
                  context,
                ).formatMediumDate(expiresAt.toLocal()),
              ),
            ),
          if (locked) Text(l10n.scheduledPostsPublishingLocked),
        ],
      ),
      trailing: locked
          ? Semantics(
              label: l10n.scheduledPostsPublishingLockSemantics,
              child: const ExcludeSemantics(
                child: Icon(CraftskyIcons.lock),
              ),
            )
          : Wrap(
              children: [
                IconButton(
                  tooltip: l10n.scheduledPostsEditTooltip,
                  onPressed: () => onEdit(item),
                  icon: const Icon(CraftskyIconsBold.edit),
                ),
                IconButton(
                  tooltip: l10n.scheduledPostsDeleteTooltip,
                  onPressed: () => onDelete(item),
                  icon: const Icon(CraftskyIconsBold.delete),
                ),
              ],
            ),
    );
  }
}

String _statusLabel(AppLocalizations l10n, ScheduledPostStatus status) =>
    switch (status) {
      ScheduledPostStatus.scheduled => l10n.scheduledPostsStatusScheduled,
      ScheduledPostStatus.publishing => l10n.scheduledPostsStatusPublishing,
      ScheduledPostStatus.retrying => l10n.scheduledPostsStatusRetrying,
      ScheduledPostStatus.needsAttention =>
        l10n.scheduledPostsStatusNeedsAttention,
    };

class _ScheduledPostsError extends StatelessWidget {
  const _ScheduledPostsError({required this.onRetry});
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.scheduledPostsLoadError),
          FilledButton(
            onPressed: onRetry,
            child: Text(l10n.scheduledPostsRetryAction),
          ),
        ],
      ),
    );
  }
}
