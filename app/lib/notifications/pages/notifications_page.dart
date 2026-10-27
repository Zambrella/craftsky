import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:craftsky_app/notifications/providers/notification_filter_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_seen_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_filter.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_label.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

class NotificationsPage extends ConsumerStatefulWidget {
  const new({super.key});

  @override
  ConsumerState<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends ConsumerState<NotificationsPage> {
  GoRouter? _router;
  bool _visible = true;
  bool _waitingForSeen = false;
  Future<void> Function()? _acknowledge;
  VoidCallback? _invalidate;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final router = GoRouter.maybeOf(context);
    if (_router == router) return;
    _router?.routerDelegate.removeListener(_routeChanged);
    _router = router;
    router?.routerDelegate.addListener(_routeChanged);
    _visible = _isCurrent;
  }

  bool get _isCurrent =>
      _router == null ||
      _router!
              .routerDelegate
              .currentConfiguration
              .lastOrNull
              ?.matchedLocation ==
          RouteLocations.notifications;

  void _routeChanged() {
    final visible = _isCurrent;
    if (_visible && !visible) unawaited(_finishVisit());
    if (!_visible && visible && !_waitingForSeen) {
      final owner = ref
          .read(sessionRegistryProvider)
          .value
          ?.activeLease
          ?.session;
      if (owner == null) {
        ref.invalidate(notificationsProvider);
      } else {
        ref.invalidate(accountNotificationsProvider(owner.account));
      }
    }
    if (_visible != visible) setState(() => _visible = visible);
  }

  Future<void> _finishVisit() async {
    final acknowledge = _acknowledge;
    final invalidate = _invalidate;
    _acknowledge = null;
    _invalidate = null;
    if (acknowledge == null) return;
    _waitingForSeen = true;
    await acknowledge();
    invalidate?.call();
    if (!mounted) return;
    setState(() => _waitingForSeen = false);
  }

  void _rendered(int token, AccountSessionLease? owner) {
    if (!_visible || !_isCurrent || _waitingForSeen) return;
    // Capture the account coordinator before leaving or switching accounts.
    final coordinator = owner == null
        ? Future.value(ref.read(notificationSeenProvider))
        : ref.read(accountNotificationSeenProvider(owner.account).future);
    _acknowledge = () async =>
        await (await coordinator).afterSuccessfulRender(token);
    final container = ProviderScope.containerOf(context, listen: false);
    _invalidate = () {
      if (owner == null) {
        container.invalidate(notificationsProvider);
      } else {
        container.invalidate(accountNotificationsProvider(owner.account));
      }
    };
  }

  @override
  void dispose() {
    _router?.routerDelegate.removeListener(_routeChanged);
    unawaited(_finishVisit());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(
      sessionRegistryProvider.select(
        (value) => value.value?.activeLease?.session.account,
      ),
      (previous, next) {
        if (previous != next) unawaited(_finishVisit());
      },
    );
    final owner = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    final loaded = owner == null
        ? ref.watch(notificationsProvider)
        : ref.watch(accountNotificationsProvider(owner.account));
    final notifications = _waitingForSeen
        ? const AsyncLoading<NotificationsState>()
        : loaded;
    final l10n = AppLocalizations.of(context);
    final category = ref.watch(notificationFilterProvider);
    return Scaffold(
      body: YarnRefreshScrollView(
        onRefresh: () => owner == null
            ? ref.refresh(notificationsProvider.future)
            : ref.refresh(accountNotificationsProvider(owner.account).future),
        headerSliverCount: 1,
        slivers: [
          SliverAppBar(
            leading: AppShellDrawerScope.maybeOf(context) == null
                ? null
                : const AppShellDrawerButton(),
            title: Text(l10n.notificationsTitle),
            pinned: true,
            actions: [
              Padding(
                padding: const EdgeInsetsDirectional.only(end: 16),
                child: ConstrainedBox(
                  constraints: BoxConstraints(
                    maxWidth: MediaQuery.sizeOf(context).width * .45,
                  ),
                  child: const NotificationCategoryFilter(),
                ),
              ),
            ],
          ),
          switch (notifications) {
            AsyncValue(:final value?) when value.category == category =>
              _NotificationsLoadedSlivers(
                items: value.items,
                hasMore: value.hasMore,
                isLoadingMore: notifications.isLoading,
                hasLoadMoreError: notifications.hasError,
                renderToken: value.renderToken,
                owner: value.owner,
                onRendered: _rendered,
              ),
            _ when notifications.hasError && !notifications.isLoading =>
              _NotificationsErrorSliver(
                onRetry: () {
                  if (owner != null) {
                    ref.invalidate(
                      accountNotificationsProvider(owner.account),
                    );
                  } else {
                    ref.invalidate(notificationsProvider);
                  }
                },
              ),
            _ => const CraftskySkeletonSliverList(
              itemBuilder: _buildActivitySkeleton,
            ),
          },
        ],
        scrollViewKey: ValueKey((owner?.account, category)),
      ),
    );
  }
}

Widget _buildActivitySkeleton(BuildContext context, int index) =>
    const ActivityRowSkeleton();

class _NotificationsLoadedSlivers extends ConsumerWidget {
  const new({
    required this.items,
    required this.hasMore,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.renderToken,
    required this.owner,
    required this.onRendered,
  });

  final List<CraftskyNotification> items;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final int renderToken;
  final AccountSessionLease? owner;
  final void Function(int, AccountSessionLease?) onRendered;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final category = ref.watch(notificationFilterProvider);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!context.mounted ||
          category != ref.read(notificationFilterProvider)) {
        return;
      }
      onRendered(renderToken, owner);
    });
    final l10n = AppLocalizations.of(context);
    if (items.isEmpty) {
      return SliverFillRemaining(
        hasScrollBody: false,
        child: CraftskyEmptyState(
          icon: CraftskyIcons.notifications,
          title: category == null
              ? l10n.notificationsTitle
              : notificationCategoryLabel(l10n, category),
          subtitle: category == null
              ? l10n.notificationsEmpty
              : l10n.notificationFilterEmpty,
          actionLabel: category == null ? null : l10n.notificationFilterShowAll,
          onAction: category == null
              ? null
              : () =>
                    ref.read(notificationFilterProvider.notifier).select(null),
        ),
      );
    }
    return SliverMainAxisGroup(
      slivers: [
        SliverList.separated(
          itemCount: items.length,
          itemBuilder: (context, index) =>
              NotificationRow(notification: items[index], owner: owner),
          separatorBuilder: (context, index) => Divider(
            height: 1,
            thickness: .5,
            indent: 16,
            endIndent: 16,
            color: Theme.of(context).colorScheme.outlineVariant.withValues(
              alpha: .5,
            ),
          ),
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
                    icon: const Icon(CraftskyIconsBold.refresh),
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
  const new({required this.onRetry});

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
