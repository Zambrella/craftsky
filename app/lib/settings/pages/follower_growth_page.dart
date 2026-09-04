import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/profile/providers/follower_growth_provider.dart';
import 'package:craftsky_app/settings/widgets/follower_growth_chart.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class FollowerGrowthPage extends ConsumerStatefulWidget {
  const FollowerGrowthPage({super.key});

  @override
  ConsumerState<FollowerGrowthPage> createState() => _FollowerGrowthPageState();
}

class _FollowerGrowthPageState extends ConsumerState<FollowerGrowthPage> {
  FollowerGrowthPeriod _period = FollowerGrowthPeriod.thirtyDays;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final account = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    final totalGrowth = account == null
        ? const AsyncLoading<FollowerGrowth>()
        : ref.watch(
            followerGrowthProvider(
              account,
              FollowerGrowthPeriod.thirtyDays,
            ),
          );
    final growth = account == null || _period == FollowerGrowthPeriod.thirtyDays
        ? totalGrowth
        : ref.watch(followerGrowthProvider(account, _period));

    return Scaffold(
      appBar: AppBar(title: Text(l10n.settingsGrowth)),
      body: SingleChildScrollView(
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp5,
          spacing.sp4,
          spacing.sp6,
        ),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                switch (totalGrowth) {
                  AsyncData(:final value) => _FollowerTotal(growth: value),
                  AsyncError() => const SizedBox.shrink(),
                  _ => const Center(child: StitchProgressIndicator()),
                },
                SizedBox(height: spacing.sp6),
                Text(
                  l10n.growthTrendLabel,
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                SizedBox(height: spacing.sp2),
                Text('${l10n.growthScopeCopy} ${l10n.growthFreshnessCopy}'),
                SizedBox(height: spacing.sp5),
                _PeriodSelector(
                  selectedPeriod: _period,
                  onPeriodChanged: (period) => setState(() => _period = period),
                ),
                SizedBox(height: spacing.sp5),
                switch (growth) {
                  AsyncData(:final value) => _GrowthContent(growth: value),
                  AsyncError() => _GrowthError(
                    onRetry: account == null
                        ? null
                        : () => ref.invalidate(
                            followerGrowthProvider(account, _period),
                          ),
                  ),
                  _ => const SizedBox.shrink(),
                },
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _PeriodSelector extends StatelessWidget {
  const _PeriodSelector({
    required this.selectedPeriod,
    required this.onPeriodChanged,
  });

  final FollowerGrowthPeriod selectedPeriod;
  final ValueChanged<FollowerGrowthPeriod> onPeriodChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SegmentedButton<FollowerGrowthPeriod>(
      expandedInsets: EdgeInsets.zero,
      segments: [
        ButtonSegment(
          value: FollowerGrowthPeriod.sevenDays,
          label: Text(l10n.growthPeriodSevenDays),
        ),
        ButtonSegment(
          value: FollowerGrowthPeriod.thirtyDays,
          label: Text(l10n.growthPeriodThirtyDays),
        ),
        ButtonSegment(
          value: FollowerGrowthPeriod.oneYear,
          label: Text(l10n.growthPeriodOneYear),
        ),
      ],
      selected: {selectedPeriod},
      onSelectionChanged: (selected) => onPeriodChanged(selected.single),
    );
  }
}

class _FollowerTotal extends StatelessWidget {
  const _FollowerTotal({required this.growth});

  final FollowerGrowth growth;

  @override
  Widget build(BuildContext context) {
    final count = growth.latestFollowerCount;
    if (count == null) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final locale = Localizations.localeOf(context).toString();
    final formattedCount = formatFollowerCount(count, locale);
    final label = l10n.growthLatestCount(formattedCount);
    final countStart = label.indexOf(formattedCount);
    final countEnd = countStart + formattedCount.length;
    final descriptor =
        '${label.substring(0, countStart)}${label.substring(countEnd)}'.trim();
    return Semantics(
      label: label,
      child: ExcludeSemantics(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              formattedCount,
              key: const ValueKey('follower-total-count'),
              style: theme.textTheme.headlineMedium?.copyWith(
                color: theme.colorScheme.primary,
              ),
            ),
            Text(
              descriptor,
              key: const ValueKey('follower-total-label'),
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _GrowthContent extends StatelessWidget {
  const _GrowthContent({required this.growth});

  final FollowerGrowth growth;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final observedCount = growth.points
        .where((point) => point.count != null)
        .length;
    final missingCount = growth.points.length - observedCount;
    final hasHistory = growth.availableFrom != null;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (!hasHistory)
          CraftskyEmptyState(
            icon: CraftskyIcons.trending,
            title: l10n.growthMetricLabel,
            subtitle: l10n.growthNoHistory,
          )
        else if (observedCount == 0)
          CraftskyEmptyState(
            icon: CraftskyIcons.trending,
            title: l10n.growthTrendLabel,
            subtitle: l10n.growthNoObservationsInPeriod,
          ),
        if (observedCount > 0)
          FollowerGrowthChart(growth: growth, period: growth.period),
        if (growth.availableFrom case final availableFrom?) ...[
          if (availableFrom.isAfter(growth.rangeStart)) ...[
            SizedBox(height: spacing.sp4),
            Text(
              l10n.growthHistoryAvailableSince(
                formatFollowerGrowthDate(
                  availableFrom,
                  Localizations.localeOf(context).toString(),
                ),
              ),
            ),
          ],
          if (missingCount > 0) ...[
            SizedBox(height: spacing.sp2),
            Text(l10n.growthMissingDays(missingCount)),
          ],
        ],
      ],
    );
  }
}

class _GrowthError extends StatelessWidget {
  const _GrowthError({required this.onRetry});

  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Center(
      child: Padding(
        padding: EdgeInsets.all(spacing.sp5),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.growthLoadError, textAlign: TextAlign.center),
            SizedBox(height: spacing.sp3),
            FilledButton(onPressed: onRetry, child: Text(l10n.retryButton)),
          ],
        ),
      ),
    );
  }
}
