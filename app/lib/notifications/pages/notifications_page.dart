import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class NotificationsPage extends ConsumerWidget {
  const NotificationsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notifications = ref.watch(notificationsProvider);
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.notificationsTitle)),
      body: CustomScrollView(
        slivers: [
          switch (notifications) {
            AsyncValue(:final value?) => _NotificationsLoadedSlivers(
              items: value.items,
              hasMore: value.hasMore,
              isLoadingMore: notifications.isLoading,
              hasLoadMoreError: notifications.hasError,
            ),
            _ when notifications.hasError => _NotificationsErrorSliver(
              onRetry: () => ref.invalidate(notificationsProvider),
            ),
            _ => const SliverFillRemaining(
              hasScrollBody: false,
              child: Center(child: StitchProgressIndicator()),
            ),
          },
        ],
      ),
    );
  }
}

class _NotificationsLoadedSlivers extends ConsumerWidget {
  const _NotificationsLoadedSlivers({
    required this.items,
    required this.hasMore,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
  });

  final List<CraftskyNotification> items;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    if (items.isEmpty) {
      return SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: Text(l10n.notificationsEmpty)),
      );
    }
    return SliverMainAxisGroup(
      slivers: [
        SliverList.builder(
          itemCount: items.length,
          itemBuilder: (context, index) =>
              NotificationRow(notification: items[index]),
        ),
        if (isLoadingMore || hasLoadMoreError || hasMore)
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Center(
                child: switch ((isLoadingMore, hasLoadMoreError)) {
                  (true, _) => const StitchProgressIndicator(),
                  (_, true) => TextButton.icon(
                    onPressed: () =>
                        ref.read(notificationsProvider.notifier).loadMore(),
                    icon: const Icon(Icons.refresh),
                    label: Text(l10n.retryButton),
                  ),
                  _ => TextButton(
                    onPressed: () =>
                        ref.read(notificationsProvider.notifier).loadMore(),
                    child: Text(l10n.notificationsLoadMore),
                  ),
                },
              ),
            ),
          ),
      ],
    );
  }
}

class _NotificationsErrorSliver extends StatelessWidget {
  const _NotificationsErrorSliver({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SliverFillRemaining(
      hasScrollBody: false,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.notificationsLoadError),
            TextButton(onPressed: onRetry, child: Text(l10n.retryButton)),
          ],
        ),
      ),
    );
  }
}
