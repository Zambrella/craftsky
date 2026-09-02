import 'dart:async';

import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/widgets/event_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef OpenBusinessEvent = void Function(Did did, RecordKey rkey);

class ProfileEventsTab extends ConsumerWidget {
  const ProfileEventsTab({
    required this.target,
    required this.isOwnProfile,
    required this.onOpen,
    this.onManage,
    super.key,
  });

  final ProfileBusinessEventsTarget target;
  final bool isOwnProfile;
  final OpenBusinessEvent onOpen;
  final VoidCallback? onManage;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final events = ref.watch(profileBusinessEventsProvider(target));
    return switch (events) {
      AsyncData(:final value) => _content(context, ref, value),
      AsyncError() => _initialError(context, ref),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }

  Widget _content(
    BuildContext context,
    WidgetRef ref,
    BusinessEventListState state,
  ) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    if (state.items.isEmpty) {
      return SliverFillRemaining(
        hasScrollBody: false,
        child: Center(
          child: Padding(
            padding: EdgeInsets.all(spacing.sp6),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  isOwnProfile
                      ? l10n.businessEventsOwnerEmpty
                      : l10n.businessEventsVisitorEmpty,
                  textAlign: TextAlign.center,
                ),
                if (isOwnProfile && onManage != null) ...[
                  SizedBox(height: spacing.sp2),
                  TextButton(
                    onPressed: onManage,
                    child: Text(l10n.businessEventsManageAction),
                  ),
                ],
              ],
            ),
          ),
        ),
      );
    }

    return SliverMainAxisGroup(
      slivers: [
        SliverPadding(
          padding: EdgeInsets.all(spacing.sp3),
          sliver: SliverList.separated(
            itemCount: state.items.length,
            separatorBuilder: (_, _) => SizedBox(height: spacing.sp2),
            itemBuilder: (context, index) {
              final event = state.items[index];
              return EventCard(
                event: event,
                onTap: () => onOpen(event.did, event.rkey),
              );
            },
          ),
        ),
        SliverToBoxAdapter(
          child: Padding(
            padding: EdgeInsets.fromLTRB(
              spacing.sp3,
              0,
              spacing.sp3,
              spacing.sp4,
            ),
            child: Column(
              children: [
                if (state.refreshError != null)
                  _RetryRow(
                    message: l10n.businessEventsRefreshError,
                    onRetry: () => unawaited(
                      ref
                          .read(profileBusinessEventsProvider(target).notifier)
                          .refresh(),
                    ),
                  ),
                if (state.incrementalError != null)
                  _RetryRow(
                    message: l10n.businessEventsLoadMoreError,
                    onRetry: () => unawaited(
                      ref
                          .read(profileBusinessEventsProvider(target).notifier)
                          .loadMore(),
                    ),
                  )
                else if (state.isLoadingMore)
                  const Padding(
                    padding: EdgeInsets.all(8),
                    child: StitchProgressIndicator(size: 24),
                  )
                else if (state.hasMore)
                  TextButton(
                    onPressed: () => unawaited(
                      ref
                          .read(profileBusinessEventsProvider(target).notifier)
                          .loadMore(),
                    ),
                    child: Text(l10n.businessEventsLoadMoreAction),
                  )
                else
                  Text(l10n.businessEventsEnd),
                TextButton.icon(
                  onPressed: state.isRefreshing
                      ? null
                      : () => unawaited(
                          ref
                              .read(
                                profileBusinessEventsProvider(target).notifier,
                              )
                              .refresh(),
                        ),
                  icon: state.isRefreshing
                      ? const StitchProgressIndicator(size: 18)
                      : const Icon(Icons.refresh),
                  label: Text(l10n.businessEventsRefreshAction),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _initialError(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return SliverFillRemaining(
      hasScrollBody: false,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.businessEventsLoadError),
            TextButton(
              onPressed: () => unawaited(
                ref
                    .read(profileBusinessEventsProvider(target).notifier)
                    .retryInitial(),
              ),
              child: Text(l10n.businessEventsRetryAction),
            ),
          ],
        ),
      ),
    );
  }
}

class _RetryRow extends StatelessWidget {
  const _RetryRow({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisAlignment: MainAxisAlignment.center,
    children: [
      Flexible(child: Text(message)),
      TextButton(
        onPressed: onRetry,
        child: Text(AppLocalizations.of(context).businessEventsRetryAction),
      ),
    ],
  );
}
