import 'dart:async';

import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/models/event_diagnostics.dart';
import 'package:craftsky_app/business/pages/event_editor_dialog.dart';
import 'package:craftsky_app/business/providers/business_event_mutation_controller.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/widgets/event_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class EventsSettingsPage extends ConsumerWidget {
  const EventsSettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final identity = ref.watch(activeAccountIdentityProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.businessEventsSettingsTitle)),
      body: identity.when(
        loading: () => Center(
          child: CircularProgressIndicator(
            semanticsLabel: l10n.businessLoading,
          ),
        ),
        error: (_, _) => _InitialError(
          onRetry: () => ref.invalidate(activeAccountIdentityProvider),
        ),
        data: (value) {
          if (value == null ||
              value.profile.accountType != AccountType.business) {
            return Center(child: Text(l10n.businessEventsUnavailable));
          }
          return const _EventsManager();
        },
      ),
      floatingActionButton:
          identity.value?.profile.accountType == AccountType.business
          ? CraftskyFloatingActionButton.extended(
              onPressed: () => _openEditor(context),
              icon: const Icon(Icons.add),
              label: Text(l10n.businessEventCreateTitle),
            )
          : null,
    );
  }
}

class _EventsManager extends StatefulWidget {
  const _EventsManager();

  @override
  State<_EventsManager> createState() => _EventsManagerState();
}

class _EventsManagerState extends State<_EventsManager>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 2, vsync: this);
  }

  @override
  void dispose() {
    _tabs.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      children: [
        TabBar(
          controller: _tabs,
          tabs: [
            Tab(text: l10n.businessEventsUpcomingTab),
            Tab(text: l10n.businessEventsHistoryTab),
          ],
        ),
        Expanded(
          child: TabBarView(
            controller: _tabs,
            children: const [
              _EventView(filter: OwnerEventFilter.upcoming),
              _EventView(filter: OwnerEventFilter.history),
            ],
          ),
        ),
      ],
    );
  }
}

class _EventView extends ConsumerWidget {
  const _EventView({required this.filter});

  final OwnerEventFilter filter;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final events = ref.watch(ownerBusinessEventsProvider(filter));
    return events.when(
      loading: () => Center(
        child: CircularProgressIndicator(
          semanticsLabel: AppLocalizations.of(context).businessLoading,
        ),
      ),
      error: (_, _) => _InitialError(
        onRetry: () => unawaited(
          ref.read(ownerBusinessEventsProvider(filter).notifier).retryInitial(),
        ),
      ),
      data: (state) => _EventList(filter: filter, state: state),
    );
  }
}

class _EventList extends ConsumerWidget {
  const _EventList({required this.filter, required this.state});

  final OwnerEventFilter filter;
  final BusinessEventListState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = ownerBusinessEventsProvider(filter);
    final controller = ref.read(provider.notifier);
    final mutation = ref.watch(businessEventMutationControllerProvider);
    final rows = <Widget>[
      if (mutation.status == EventMutationStatus.conflict)
        MaterialBanner(
          content: Text(l10n.businessEventConflict),
          actions: [
            TextButton(
              onPressed: () => unawaited(
                ref
                    .read(businessEventMutationControllerProvider.notifier)
                    .reloadConflict(),
              ),
              child: Text(l10n.businessEventReload),
            ),
            TextButton(
              onPressed: () => unawaited(
                ref
                    .read(businessEventMutationControllerProvider.notifier)
                    .retryConflict(),
              ),
              child: Text(l10n.businessEventsRetryAction),
            ),
          ],
        )
      else if (mutation.status == EventMutationStatus.error)
        MaterialBanner(
          content: Text(l10n.businessEventSaveError),
          actions: const [SizedBox.shrink()],
        ),
      if (state.refreshError != null)
        _InlineError(
          message: l10n.businessEventsOwnerRefreshError,
          onRetry: () => unawaited(controller.refresh()),
        ),
      if (state.items.isEmpty)
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 48, horizontal: 24),
          child: Center(
            child: Text(
              filter == OwnerEventFilter.upcoming
                  ? l10n.businessEventsUpcomingEmpty
                  : l10n.businessEventsHistoryEmpty,
              textAlign: TextAlign.center,
            ),
          ),
        )
      else
        for (final event in state.items) _OwnerEventCard(event: event),
      if (state.incrementalError != null)
        _InlineError(
          message: l10n.businessEventsLoadMoreError,
          onRetry: () => unawaited(controller.loadMore()),
        )
      else if (state.isLoadingMore)
        Padding(
          padding: const EdgeInsets.all(16),
          child: Center(
            child: CircularProgressIndicator(
              semanticsLabel: l10n.businessLoading,
            ),
          ),
        )
      else if (state.hasMore)
        Center(
          child: TextButton(
            onPressed: () => unawaited(controller.loadMore()),
            child: Text(l10n.businessEventsLoadMoreAction),
          ),
        )
      else if (state.items.isNotEmpty)
        Center(child: Text(l10n.businessEventsEnd)),
      const SizedBox(height: 88),
    ];

    return RefreshIndicator(
      onRefresh: controller.refresh,
      child: ListView(
        key: PageStorageKey(filter),
        padding: const EdgeInsets.all(16),
        children: rows,
      ),
    );
  }
}

enum _EventAction { edit, cancel, postpone, delete }

class _OwnerEventCard extends ConsumerWidget {
  const _OwnerEventCard({required this.event});

  final BusinessEvent event;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final diagnostics = EventDiagnostics.localized(
      [...event.publicSuppressionReasons, ...event.upcomingExclusionReasons],
      l10n,
    );
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          EventCard(
            event: event,
            onTap: () => _openEditor(context, event),
            trailing: CraftskyContextMenuButton(
              tooltip: l10n.businessEventManage(event.name),
              groups: [
                CraftskyContextMenuGroup(
                  items: [
                    CraftskyContextMenuItem(
                      text: l10n.businessEventEditAction,
                      icon: Icons.edit_outlined,
                      onPressed: () => _performAction(
                        context,
                        ref,
                        event,
                        _EventAction.edit,
                      ),
                    ),
                  ],
                ),
                CraftskyContextMenuGroup(
                  items: [
                    if (event.status.value != 'cancelled')
                      CraftskyContextMenuItem(
                        text: l10n.businessEventCancelAction,
                        icon: Icons.event_busy_outlined,
                        onPressed: () => _performAction(
                          context,
                          ref,
                          event,
                          _EventAction.cancel,
                        ),
                      ),
                    if (event.status.value != 'postponed')
                      CraftskyContextMenuItem(
                        text: l10n.businessEventPostponeAction,
                        icon: Icons.schedule_outlined,
                        onPressed: () => _performAction(
                          context,
                          ref,
                          event,
                          _EventAction.postpone,
                        ),
                      ),
                  ],
                ),
                CraftskyContextMenuGroup(
                  items: [
                    CraftskyContextMenuItem(
                      text: l10n.businessEventDeleteAction,
                      icon: Icons.delete_outline,
                      style: CraftskyContextMenuItemStyle.destructive,
                      onPressed: () => _performAction(
                        context,
                        ref,
                        event,
                        _EventAction.delete,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
          for (final diagnostic in diagnostics)
            Padding(
              padding: EdgeInsets.only(top: spacing.sp2),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Icon(Icons.info_outline, size: 18),
                  SizedBox(width: spacing.sp2),
                  Expanded(child: Text(diagnostic)),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

Future<void> _performAction(
  BuildContext context,
  WidgetRef ref,
  BusinessEvent event,
  _EventAction action,
) async {
  final controller = ref.read(
    businessEventMutationControllerProvider.notifier,
  );
  switch (action) {
    case _EventAction.edit:
      await _openEditor(context, event);
    case _EventAction.cancel:
      await controller.changeStatus(event, 'cancelled');
    case _EventAction.postpone:
      await controller.changeStatus(event, 'postponed');
    case _EventAction.delete:
      if (await _confirmDelete(context)) {
        await controller.delete(event, confirmed: true);
      }
  }
}

Future<void> _openEditor(BuildContext context, [BusinessEvent? event]) =>
    showEventEditorSheet(context, event: event);

Future<bool> _confirmDelete(BuildContext context) async {
  final l10n = AppLocalizations.of(context);
  return showCraftskyDestructiveConfirmDialog(
    context,
    title: l10n.businessEventDeleteConfirmTitle,
    message: l10n.businessEventDeleteConfirmMessage,
    confirmLabel: l10n.businessEventDeleteConfirmAction,
    cancelLabel: l10n.businessProductCancel,
  );
}

class _InitialError extends StatelessWidget {
  const _InitialError({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.businessEventsOwnerLoadError),
          TextButton(
            onPressed: onRetry,
            child: Text(l10n.businessEventsRetryAction),
          ),
        ],
      ),
    );
  }
}

class _InlineError extends StatelessWidget {
  const _InlineError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 8),
    child: Row(
      children: [
        Expanded(child: Text(message)),
        TextButton(
          onPressed: onRetry,
          child: Text(AppLocalizations.of(context).businessEventsRetryAction),
        ),
      ],
    ),
  );
}
