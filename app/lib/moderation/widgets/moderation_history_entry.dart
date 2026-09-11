import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ModerationHistoryEntryCard extends StatelessWidget {
  const ModerationHistoryEntryCard({
    required this.entry,
    required this.onAppeal,
    required this.onCopyAddress,
    required this.onCopyReference,
    this.onOpenSubject,
    super.key,
  });

  final ModerationHistoryEntry entry;
  final VoidCallback onAppeal;
  final VoidCallback onCopyAddress;
  final VoidCallback onCopyReference;
  final VoidCallback? onOpenSubject;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final date = MaterialLocalizations.of(
      context,
    ).formatMediumDate(entry.occurredAt.toLocal());
    final time = MaterialLocalizations.of(context).formatTimeOfDay(
      TimeOfDay.fromDateTime(entry.occurredAt.toLocal()),
    );
    final snapshot = entry.safeSnapshot.displayReference;
    return Semantics(
      container: true,
      label: l10n.moderationHistoryEntrySemantics(entry.caseReference),
      child: CraftskyCard(
        key: const Key('moderation-history-entry-card'),
        margin: EdgeInsets.symmetric(vertical: spacing.sp2),
        padding: EdgeInsets.all(spacing.sp4),
        clipBehavior: Clip.none,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Padding(
                  padding: EdgeInsetsDirectional.only(end: 12, top: 2),
                  child: Icon(CraftskyIcons.warning),
                ),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _reasonLabel(l10n, entry.reason),
                        style: theme.textTheme.titleMedium,
                      ),
                      Text(
                        l10n.moderationHistoryOccurredAt(date, time),
                        style: theme.textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
              ],
            ),
            if (entry.userSafeDetail.isNotEmpty) ...[
              const SizedBox(height: 12),
              Text(entry.userSafeDetail),
            ],
            if (snapshot != null) ...[
              const SizedBox(height: 12),
              Text(
                _subjectLabel(l10n, entry.safeSnapshot.type),
                style: theme.textTheme.labelLarge,
              ),
              const SizedBox(height: 2),
              SelectableText(snapshot),
              if (onOpenSubject case final onOpenSubject?)
                Align(
                  alignment: AlignmentDirectional.centerStart,
                  child: TextButton.icon(
                    onPressed: onOpenSubject,
                    icon: const Icon(CraftskyIcons.projectPost),
                    label: Text(l10n.moderationViewPost),
                  ),
                ),
            ],
            if (entry.effects.isNotEmpty) ...[
              const SizedBox(height: 12),
              Text(
                l10n.moderationConsequencesTitle,
                style: theme.textTheme.labelLarge,
              ),
              const SizedBox(height: 6),
              for (final effect in entry.effects) _EffectRow(effect: effect),
            ],
            if (entry.appealStatus != ModerationAppealState.none) ...[
              const SizedBox(height: 12),
              Text(
                _appealLabel(l10n, entry.appealStatus),
                style: theme.textTheme.labelLarge,
              ),
            ],
            const SizedBox(height: 12),
            SelectableText(
              entry.caseReference,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 4,
              children: [
                if (entry.canStartAppeal)
                  FilledButton.icon(
                    onPressed: onAppeal,
                    icon: const Icon(CraftskyIcons.email),
                    label: Text(l10n.moderationAppealAction),
                  ),
                TextButton.icon(
                  onPressed: onCopyAddress,
                  icon: const Icon(CraftskyIcons.copy),
                  label: Text(l10n.moderationCopyAppealAddress),
                ),
                TextButton.icon(
                  onPressed: onCopyReference,
                  icon: const Icon(CraftskyIcons.copy),
                  label: Text(l10n.moderationCopyCaseReference),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _EffectRow extends StatelessWidget {
  const _EffectRow({required this.effect});

  final ModerationHistoryEffect effect;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final dueAt = effect.dueAt;
    final state = _effectStateLabel(l10n, effect);
    final dueLabel = dueAt == null
        ? null
        : MaterialLocalizations.of(context).formatFullDate(dueAt.toLocal());
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            effect.action == ModerationEffectAction.apply
                ? CraftskyIcons.error
                : CraftskyIcons.history,
            size: 18,
            semanticLabel: state,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              dueLabel == null
                  ? '${_effectLabel(l10n, effect.type)} · $state'
                  : l10n.moderationEffectWithExpiry(
                      _effectLabel(l10n, effect.type),
                      state,
                      dueLabel,
                    ),
            ),
          ),
        ],
      ),
    );
  }
}

String _reasonLabel(AppLocalizations l10n, ModerationReason reason) =>
    switch (reason) {
      ModerationReason.harassment => l10n.moderationReasonHarassment,
      ModerationReason.hate => l10n.moderationReasonHate,
      ModerationReason.spam => l10n.moderationReasonSpam,
      ModerationReason.misleading => l10n.moderationReasonMisleading,
      ModerationReason.suspectedAiGenerated => l10n.moderationReasonSuspectedAi,
      ModerationReason.adultOrGraphic => l10n.moderationReasonAdultOrGraphic,
      ModerationReason.impersonation => l10n.moderationReasonImpersonation,
      ModerationReason.offTopic => l10n.moderationReasonOffTopic,
      ModerationReason.intellectualProperty =>
        l10n.moderationReasonIntellectualProperty,
      ModerationReason.other => l10n.moderationReasonOther,
      ModerationReason.unknown => l10n.moderationReasonPolicy,
    };

String _effectLabel(AppLocalizations l10n, ModerationEffectType type) =>
    switch (type) {
      ModerationEffectType.formalWarning => l10n.moderationEffectFormalWarning,
      ModerationEffectType.visibilityWarn => l10n.moderationEffectViewerWarning,
      ModerationEffectType.visibilityHide => l10n.moderationEffectHidden,
      ModerationEffectType.visibilityTakedown => l10n.moderationEffectRemoved,
      ModerationEffectType.strike => l10n.moderationEffectStrike,
      ModerationEffectType.severeSuspension =>
        l10n.moderationEffectSevereSuspension,
      ModerationEffectType.unknown => l10n.moderationEffectPolicyAction,
    };

String _effectStateLabel(
  AppLocalizations l10n,
  ModerationHistoryEffect effect,
) => switch (effect.action) {
  ModerationEffectAction.expire => l10n.moderationEffectExpired,
  ModerationEffectAction.negate => l10n.moderationEffectOverturned,
  ModerationEffectAction.restore => l10n.moderationEffectRestored,
  ModerationEffectAction.apply => l10n.moderationEffectApplied,
  ModerationEffectAction.unknown => l10n.moderationEffectNoLongerActive,
};

String _appealLabel(AppLocalizations l10n, ModerationAppealState state) =>
    switch (state) {
      ModerationAppealState.pending => l10n.moderationAppealPending,
      ModerationAppealState.upheld => l10n.moderationAppealUpheld,
      ModerationAppealState.changed => l10n.moderationAppealChanged,
      ModerationAppealState.none => l10n.moderationAppealAvailable,
      ModerationAppealState.unknown => l10n.moderationAppealUpdated,
    };

String _subjectLabel(AppLocalizations l10n, String type) => switch (type) {
  'account' => l10n.moderationSubjectAccount,
  'post' => l10n.moderationSubjectPost,
  'event' => l10n.moderationSubjectEvent,
  _ => l10n.moderationSubjectUnavailable,
};
