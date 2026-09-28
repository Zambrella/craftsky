import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/moderation/providers/moderation_providers.dart';
import 'package:craftsky_app/moderation/widgets/moderation_history_entry.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountStandingPage extends ConsumerWidget {
  const AccountStandingPage({this.account, super.key});

  final AccountKey? account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final resolvedAccount =
        account ??
        ref.watch(sessionRegistryProvider).value?.activeLease?.session.account;
    return Scaffold(
      appBar: AppBar(
        leading: BackButton(
          onPressed: () => const SettingsRoute().go(context),
        ),
        title: Text(l10n.accountStandingTitle),
      ),
      body: resolvedAccount == null
          ? const Center(child: CircularProgressIndicator())
          : _AccountStandingBody(
              account: resolvedAccount,
            ),
    );
  }
}

class _AccountStandingBody extends ConsumerWidget {
  const _AccountStandingBody({required this.account});

  final AccountKey account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final standing = ref.watch(accountStandingProvider(account));
    final history = ref.watch(accountModerationHistoryProvider(account));
    return RefreshIndicator(
      onRefresh: () async {
        ref
          ..invalidate(accountStandingProvider(account))
          ..invalidate(accountModerationHistoryProvider(account));
        await Future.wait([
          ref.read(accountStandingProvider(account).future),
          ref.read(accountModerationHistoryProvider(account).future),
        ]);
      },
      child: LayoutBuilder(
        builder: (context, constraints) => SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: EdgeInsets.symmetric(
            horizontal: constraints.maxWidth >= 840 ? 32 : 16,
            vertical: 16,
          ),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 760),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  switch (standing) {
                    AsyncData(:final value) => _StandingSummaryCard(
                      standing: value,
                    ),
                    AsyncError() => _LoadError(
                      message: AppLocalizations.of(
                        context,
                      ).moderationStandingLoadError,
                      onRetry: () =>
                          ref.invalidate(accountStandingProvider(account)),
                    ),
                    _ => const _StandingSkeleton(),
                  },
                  const SizedBox(height: 20),
                  Text(
                    AppLocalizations.of(context).moderationHistoryTitle,
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  switch (history) {
                    AsyncData(:final value) => _HistoryContent(
                      account: account,
                      state: value,
                    ),
                    AsyncError() => _LoadError(
                      message: AppLocalizations.of(
                        context,
                      ).moderationHistoryLoadError,
                      onRetry: () {
                        ref.invalidate(
                          accountModerationHistoryProvider(account),
                        );
                      },
                    ),
                    _ => const Center(
                      child: Padding(
                        padding: EdgeInsets.all(32),
                        child: CircularProgressIndicator(),
                      ),
                    ),
                  },
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _StandingSummaryCard extends StatelessWidget {
  const _StandingSummaryCard({required this.standing});

  final AccountStanding standing;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final title = standing.suspended
        ? l10n.moderationStandingSuspended
        : standing.activeStrikeCount == 0
        ? l10n.moderationStandingGood
        : l10n.moderationStandingActionRequired;
    final detail = standing.severeSuspended
        ? l10n.moderationStandingSevereDetail
        : standing.thresholdSuspended
        ? l10n.moderationStandingThresholdDetail
        : l10n.moderationStandingCount(
            standing.activeStrikeCount,
            standing.strikeThreshold,
          );
    final (tileColor, iconColor) = standing.suspended
        ? (semanticColors.errorSurface, semanticColors.error)
        : standing.activeStrikeCount == 0
        ? (semanticColors.successSurface, swatches.onMoss)
        : (semanticColors.warningSurface, theme.colorScheme.onSurface);
    return Semantics(
      container: true,
      header: true,
      label: '$title. $detail',
      child: CraftskyCard(
        key: const Key('account-standing-summary-card'),
        padding: EdgeInsets.all(spacing.sp5),
        clipBehavior: Clip.none,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              key: const Key('account-standing-summary-icon-tile'),
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: tileColor,
                border: Border.all(
                  color: theme.colorScheme.onSurface,
                  width: 1.5,
                ),
              ),
              alignment: Alignment.center,
              child: Icon(
                standing.suspended ? CraftskyIcons.lock : CraftskyIcons.privacy,
                color: iconColor,
                size: 30,
              ),
            ),
            SizedBox(width: spacing.sp4),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: theme.textTheme.titleLarge),
                  SizedBox(height: spacing.sp2),
                  Text(
                    detail,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HistoryContent extends ConsumerWidget {
  const _HistoryContent({
    required this.account,
    required this.state,
  });

  final AccountKey account;
  final ModerationHistoryState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    if (state.items.isEmpty) {
      return CraftskyEmptyState(
        icon: CraftskyIcons.privacy,
        title: l10n.moderationHistoryEmptyTitle,
        subtitle: l10n.moderationHistoryEmptyBody,
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final entry in state.items)
          ModerationHistoryEntryCard(
            key: ValueKey((
              entry.caseReference,
              entry.eventType,
              entry.occurredAt,
            )),
            entry: entry,
            onOpenSubject: _openPostAction(context, entry.safeSnapshot),
            onAppeal: () => unawaited(_appeal(context, ref, entry)),
            onCopyAddress: () => unawaited(
              _copy(context, ref, moderationAppealAddress),
            ),
            onCopyReference: () => unawaited(
              _copy(context, ref, entry.caseReference),
            ),
          ),
        if (state.hasMore)
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: FilledButton(
              onPressed: state.loadingMore
                  ? null
                  : () => ref
                        .read(
                          accountModerationHistoryProvider(account).notifier,
                        )
                        .loadMore(),
              child: Text(
                state.loadingMore
                    ? l10n.moderationHistoryLoadingMore
                    : state.loadMoreFailed
                    ? l10n.moderationHistoryRetryMore
                    : l10n.moderationHistoryLoadMore,
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _appeal(
    BuildContext context,
    WidgetRef ref,
    ModerationHistoryEntry entry,
  ) async {
    try {
      final launched = await ref.read(moderationMailLauncherProvider)(
        moderationAppealUri(entry.reference),
      );
      if (!launched && context.mounted) {
        _showError(context, AppLocalizations.of(context).moderationAppealError);
      }
    } on Object {
      if (context.mounted) {
        _showError(context, AppLocalizations.of(context).moderationAppealError);
      }
    }
  }

  Future<void> _copy(BuildContext context, WidgetRef ref, String value) async {
    try {
      await ref.read(moderationTextCopierProvider)(value);
      if (context.mounted) {
        _showInfo(context, AppLocalizations.of(context).moderationCopied);
      }
    } on Object {
      if (context.mounted) {
        _showError(context, AppLocalizations.of(context).moderationCopyError);
      }
    }
  }

  VoidCallback? _openPostAction(
    BuildContext context,
    ModerationSafeSnapshot snapshot,
  ) {
    final uri = snapshot.uri;
    if (snapshot.type != 'post' || uri == null) return null;
    try {
      final parts = parseCraftskyPostUri(AtUri.parse(uri));
      if (parts == null || parts.did.toString() != snapshot.did) return null;
      return () => PostThreadRoute(
        did: parts.did.toString(),
        rkey: parts.rkey.toString(),
      ).push<void>(context);
    } on Object {
      return null;
    }
  }
}

class _StandingSkeleton extends StatelessWidget {
  const _StandingSkeleton();

  @override
  Widget build(BuildContext context) => const Card(
    child: SizedBox(
      height: 120,
      child: Center(child: CircularProgressIndicator()),
    ),
  );
}

class _LoadError extends StatelessWidget {
  const _LoadError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 24),
    child: Column(
      children: [
        Text(message, textAlign: TextAlign.center),
        const SizedBox(height: 8),
        FilledButton(
          onPressed: onRetry,
          child: Text(AppLocalizations.of(context).moderationRetryAction),
        ),
      ],
    ),
  );
}

void _showInfo(BuildContext context, String message) {
  try {
    context.showInfo(message);
  } on Object {
    // Isolated widget hosts may not install the app-level messenger.
  }
}

void _showError(BuildContext context, String message) {
  try {
    context.showError(message);
  } on Object {
    // Isolated widget hosts may not install the app-level messenger.
  }
}
