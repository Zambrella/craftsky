import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/providers/notification_seen_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class NotificationsPage extends ConsumerWidget {
  const NotificationsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final owner = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    final notifications = owner == null
        ? ref.watch(notificationsProvider)
        : ref.watch(accountNotificationsProvider(owner.account));
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      body: CustomScrollView(
        slivers: [
          SliverAppBar(
            title: Text(l10n.notificationsTitle),
            pinned: true,
            actions: [
              IconButton(
                tooltip: l10n.notificationSettingsAction,
                onPressed: () =>
                    const NotificationSettingsRoute().push<void>(context),
                icon: const Icon(Icons.settings_outlined),
              ),
            ],
          ),
          switch (notifications) {
            AsyncValue(:final value?) => _NotificationsLoadedSlivers(
              items: value.items,
              hasMore: value.hasMore,
              isLoadingMore: notifications.isLoading,
              hasLoadMoreError: notifications.hasError,
              renderToken: value.renderToken,
              owner: value.owner,
            ),
            _ when notifications.hasError => _NotificationsErrorSliver(
              onRetry: () {
                if (owner != null) {
                  ref.invalidate(accountNotificationsProvider(owner.account));
                } else {
                  ref.invalidate(notificationsProvider);
                }
              },
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
    required this.renderToken,
    required this.owner,
  });

  final List<CraftskyNotification> items;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final int renderToken;
  final AccountSessionLease? owner;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final rowOwner = owner;
      if (rowOwner == null) {
        unawaited(
          ref.read(notificationSeenProvider).afterSuccessfulRender(renderToken),
        );
      } else {
        unawaited(
          ref
              .read(accountNotificationSeenProvider(rowOwner.account).future)
              .then((seen) => seen.afterSuccessfulRender(renderToken)),
        );
      }
    });
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
              NotificationRow(notification: items[index], owner: owner),
        ),
        if (isLoadingMore || hasLoadMoreError || hasMore)
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Center(
                child: switch ((isLoadingMore, hasLoadMoreError)) {
                  (true, _) => const StitchProgressIndicator(),
                  (_, true) => TextButton.icon(
                    onPressed: () => _loadMore(ref),
                    icon: const Icon(Icons.refresh),
                    label: Text(l10n.retryButton),
                  ),
                  _ => TextButton(
                    onPressed: () => _loadMore(ref),
                    child: Text(l10n.notificationsLoadMore),
                  ),
                },
              ),
            ),
          ),
      ],
    );
  }

  void _loadMore(WidgetRef ref) {
    final rowOwner = owner;
    if (rowOwner == null) {
      unawaited(ref.read(notificationsProvider.notifier).loadMore());
    } else {
      unawaited(
        ref
            .read(accountNotificationsProvider(rowOwner.account).notifier)
            .loadMore(),
      );
    }
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
