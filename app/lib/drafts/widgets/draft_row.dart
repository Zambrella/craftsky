import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

final class DraftRow extends StatelessWidget {
  const DraftRow({
    required this.draft,
    required this.onEdit,
    required this.onDelete,
    this.thumbnailBuilder,
    super.key,
  });

  final LocalPostDraft draft;
  final Future<void> Function(LocalPostDraft draft) onEdit;
  final Future<void> Function(LocalPostDraft draft) onDelete;
  final Widget Function(String draftId, String mediaId)? thumbnailBuilder;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final unavailable = !draft.canEdit;
    final preview = unavailable ? l10n.draftsUnavailable : _preview(l10n);
    final firstMedia = draft.media.firstOrNull;
    final local = draft.updatedAt.toLocal();
    final date = MaterialLocalizations.of(context).formatMediumDate(local);
    final time = MaterialLocalizations.of(
      context,
    ).formatTimeOfDay(TimeOfDay.fromDateTime(local));
    final kind = draft.kind == LocalPostDraftKind.project
        ? l10n.draftsKindProject
        : l10n.draftsKindStandard;
    final projectBody = switch (draft.content) {
      ProjectDraftContent(:final body) => body.trim(),
      _ => null,
    };

    return ListTile(
      onTap: unavailable ? null : () => onEdit(draft),
      leading: _leading(l10n, firstMedia),
      title: Text(preview, maxLines: 2, overflow: TextOverflow.ellipsis),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!unavailable &&
              projectBody != null &&
              projectBody.isNotEmpty &&
              projectBody != preview)
            Text(
              projectBody,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
          if (!unavailable) Text(l10n.draftsRowDateTime(kind, date, time)),
          if (firstMedia?.availability == DraftMediaAvailability.unavailable)
            Text(l10n.draftsImageUnavailable),
        ],
      ),
      trailing: Wrap(
        spacing: 4,
        children: [
          if (!unavailable)
            IconButton(
              tooltip: l10n.draftsEditTooltip,
              onPressed: () => onEdit(draft),
              icon: const Icon(CraftskyIconsBold.edit),
            ),
          IconButton(
            tooltip: l10n.draftsDeleteTooltip,
            onPressed: () => onDelete(draft),
            icon: const Icon(CraftskyIconsBold.delete),
          ),
        ],
      ),
    );
  }

  Widget _leading(
    AppLocalizations l10n,
    DraftMediaDescriptor? firstMedia,
  ) {
    if (firstMedia == null) return const Icon(CraftskyIcons.draft);
    if (firstMedia.availability == DraftMediaAvailability.unavailable) {
      return const Icon(CraftskyIcons.brokenImage);
    }
    return thumbnailBuilder?.call(draft.id, firstMedia.mediaId) ??
        const Icon(CraftskyIcons.image);
  }

  String _preview(AppLocalizations l10n) {
    final raw = switch (draft.content) {
      StandardDraftContent(:final text) => text,
      ProjectDraftContent(:final body, :final knownProjectFieldValues) =>
        switch (knownProjectFieldValues['title']) {
          final String title when title.trim().isNotEmpty => title,
          _ => body,
        },
    };
    final normalized = raw.trim();
    if (normalized.isEmpty) return l10n.draftsUntitled;
    final runes = normalized.runes.toList(growable: false);
    if (runes.length <= 120) return normalized;
    return '${String.fromCharCodes(runes.take(119))}…';
  }
}
