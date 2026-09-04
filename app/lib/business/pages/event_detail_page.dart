import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/widgets/business_image.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

class EventDetailPage extends ConsumerWidget {
  const EventDetailPage({
    required this.account,
    required this.owner,
    required this.rkey,
    this.launchExternal = launchExternalLink,
    this.confirmExternal = showOpenLinkDialog,
    super.key,
  });

  final AccountKey account;
  final Did owner;
  final RecordKey rkey;
  final ExternalLinkLauncher launchExternal;
  final ExternalLinkConfirmer confirmExternal;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final target = BusinessEventDetailTarget(
      account: account,
      owner: owner,
      rkey: rkey,
    );
    final detail = ref.watch(businessEventDetailProvider(target));
    final event = switch (detail) {
      AsyncData(value: BusinessEventDetailAvailable(:final event)) => event,
      _ => null,
    };
    final reportAction = event != null && account.did != event.did
        ? CraftskyContextMenuButton(
            tooltip: l10n.businessEventReportAction,
            groups: [
              CraftskyContextMenuGroup(
                items: [
                  CraftskyContextMenuItem(
                    text: l10n.businessEventReportAction,
                    icon: CraftskyIconsBold.report,
                    onPressed: () => showBusinessEventReportSheet(
                      context,
                      ref,
                      account: account,
                      owner: event.did,
                      rkey: event.rkey,
                    ),
                  ),
                ],
              ),
            ],
          )
        : null;
    return switch (detail) {
      AsyncData(value: BusinessEventDetailAvailable(:final event)) => Scaffold(
        body: _AvailableEventDetail(
          event: event,
          isOwner: account.did == event.did,
          reportAction: reportAction,
          launchExternal: launchExternal,
          confirmExternal: confirmExternal,
        ),
      ),
      _ => Scaffold(
        appBar: AppBar(title: Text(l10n.businessEventDetailTitle)),
        body: switch (detail) {
          AsyncData(value: BusinessEventDetailUnavailable()) =>
            const _Unavailable(),
          AsyncError() => _LoadError(
            onRetry: () =>
                ref.read(businessEventDetailProvider(target).notifier).retry(),
          ),
          _ => Center(
            child: Semantics(
              liveRegion: true,
              label: l10n.businessLoading,
              child: const StitchProgressIndicator(),
            ),
          ),
        },
      ),
    };
  }
}

class _AvailableEventDetail extends StatefulWidget {
  const _AvailableEventDetail({
    required this.event,
    required this.isOwner,
    required this.reportAction,
    required this.launchExternal,
    required this.confirmExternal,
  });

  final BusinessEvent event;
  final bool isOwner;
  final Widget? reportAction;
  final ExternalLinkLauncher launchExternal;
  final ExternalLinkConfirmer confirmExternal;

  @override
  State<_AvailableEventDetail> createState() => _AvailableEventDetailState();
}

class _AvailableEventDetailState extends State<_AvailableEventDetail> {
  late final ScrollController _scrollController;
  double _collapseOffset = double.infinity;
  bool _collapsed = false;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController()..addListener(_updateCollapsed);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _updateCollapsed() {
    final collapsed = _scrollController.offset >= _collapseOffset;
    if (collapsed != _collapsed) setState(() => _collapsed = collapsed);
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final expandedHeight = constraints.maxWidth * 9 / 16;
        _collapseOffset =
            expandedHeight - kToolbarHeight - MediaQuery.paddingOf(context).top;
        return CustomScrollView(
          controller: _scrollController,
          slivers: [
            _EventDetailAppBar(
              event: widget.event,
              expandedWidth: constraints.maxWidth,
              collapsed: _collapsed,
              reportAction: widget.reportAction,
            ),
            SliverToBoxAdapter(
              child: _EventDetail(
                event: widget.event,
                isOwner: widget.isOwner,
                launchExternal: widget.launchExternal,
                confirmExternal: widget.confirmExternal,
              ),
            ),
          ],
        );
      },
    );
  }
}

class _EventDetailAppBar extends StatelessWidget {
  const _EventDetailAppBar({
    required this.event,
    required this.expandedWidth,
    required this.collapsed,
    required this.reportAction,
  });

  final BusinessEvent event;
  final double expandedWidth;
  final bool collapsed;
  final Widget? reportAction;

  @override
  Widget build(BuildContext context) {
    final image = event.image;
    if (image == null) {
      return SliverAppBar(
        pinned: true,
        title: Text(event.name),
        actions: [?reportAction],
      );
    }
    final expandedHeight = expandedWidth * 9 / 16;
    final theme = Theme.of(context);
    final foreground = collapsed ? theme.colorScheme.onSurface : Colors.white;
    final overlayStyle = collapsed
        ? switch (theme.brightness) {
            Brightness.light => SystemUiOverlayStyle.dark,
            Brightness.dark => SystemUiOverlayStyle.light,
          }
        : SystemUiOverlayStyle.light;
    return SliverAppBar(
      key: const Key('event-detail-app-bar'),
      pinned: true,
      expandedHeight: expandedHeight,
      foregroundColor: foreground,
      backgroundColor:
          theme.appBarTheme.backgroundColor ?? theme.colorScheme.surface,
      systemOverlayStyle: overlayStyle,
      flexibleSpace: FlexibleSpaceBar(
        title: Text(event.name, style: TextStyle(color: foreground)),
        background: Semantics(
          image: true,
          label: image.alt,
          child: Stack(
            fit: StackFit.expand,
            children: [
              BusinessImage(
                image: image,
                networkUrl: image.fullsize,
                fit: BoxFit.cover,
              ),
              const DecoratedBox(
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Color(0x99000000),
                      Colors.transparent,
                      Color(0x99000000),
                    ],
                    stops: [0, 0.55, 1],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
      actions: [?reportAction],
    );
  }
}

class _EventDetail extends StatelessWidget {
  const _EventDetail({
    required this.event,
    required this.isOwner,
    required this.launchExternal,
    required this.confirmExternal,
  });

  final BusinessEvent event;
  final bool isOwner;
  final ExternalLinkLauncher launchExternal;
  final ExternalLinkConfirmer confirmExternal;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final display = BusinessFormatters.event(event, l10n);
    final eventDestination = hydratedExternalActionUri(event.eventUri);
    final registrationDestination = hydratedExternalActionUri(
      event.registrationUri,
    );
    final role = event.roles
        .map((value) => BusinessLabels.eventRole(value, l10n))
        .join(', ');

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 760),
        child: Padding(
          padding: EdgeInsets.all(spacing.sp4),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (event.summary case final summary?) ...[
                Text(summary, style: theme.textTheme.bodyLarge),
              ],
              SizedBox(height: spacing.sp3),
              Wrap(
                spacing: spacing.sp2,
                runSpacing: spacing.sp2,
                children: [
                  Chip(
                    avatar: const Icon(CraftskyIcons.eventAvailable),
                    label: Text(
                      BusinessLabels.eventStatus(event.status, l10n),
                    ),
                  ),
                  Chip(
                    avatar: Icon(
                      event.past
                          ? CraftskyIcons.history
                          : CraftskyIcons.upcoming,
                    ),
                    label: Text(
                      event.past
                          ? l10n.businessEventLifecyclePast
                          : l10n.businessEventLifecycleUpcoming,
                    ),
                  ),
                ],
              ),
              SizedBox(height: spacing.sp4),
              CraftskyCard(
                clipBehavior: Clip.none,
                padding: EdgeInsets.all(spacing.sp4),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _EventInfo(
                      icon: CraftskyIcons.date,
                      title: display.date,
                    ),
                    Divider(height: spacing.sp5),
                    _EventInfo(
                      icon: CraftskyIcons.schedule,
                      title: display.time,
                      subtitle: event.timeZone,
                    ),
                    if (event.venueName case final venue?) ...[
                      Divider(height: spacing.sp5),
                      _EventInfo(
                        icon: CraftskyIcons.location,
                        title: venue,
                      ),
                    ],
                    if (event.mode case final mode?) ...[
                      Divider(height: spacing.sp5),
                      _EventInfo(
                        icon: CraftskyIcons.people,
                        title: BusinessLabels.eventMode(mode, l10n),
                      ),
                    ],
                    if (isOwner && role.isNotEmpty) ...[
                      Divider(height: spacing.sp5),
                      _EventInfo(
                        icon: CraftskyIcons.businessIdentity,
                        title: role,
                        subtitle: l10n.businessEventRolesLabel,
                      ),
                    ],
                    if (isOwner) ...[
                      Divider(height: spacing.sp5),
                      _EventInfo(
                        icon: CraftskyIcons.publish,
                        title: l10n.businessEventPublishedOn(
                          DateFormat.yMMMd(
                            l10n.localeName,
                          ).format(event.createdAt.toLocal()),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              if (eventDestination != null || registrationDestination != null)
                Padding(
                  padding: EdgeInsets.only(top: spacing.sp4),
                  child: Wrap(
                    spacing: spacing.sp2,
                    runSpacing: spacing.sp2,
                    children: [
                      if (eventDestination != null)
                        OutlinedButton.icon(
                          onPressed: () => unawaited(
                            confirmAndLaunchExternalAction(
                              context,
                              uri: eventDestination,
                              launchUrl: launchExternal,
                              confirmOpenLink: confirmExternal,
                            ),
                          ),
                          icon: const Icon(CraftskyIconsBold.externalLink),
                          label: Text(l10n.businessEventWebsiteAction),
                        ),
                      if (registrationDestination != null)
                        OutlinedButton.icon(
                          onPressed: () => unawaited(
                            confirmAndLaunchExternalAction(
                              context,
                              uri: registrationDestination,
                              launchUrl: launchExternal,
                              confirmOpenLink: confirmExternal,
                            ),
                          ),
                          icon: const Icon(
                            CraftskyIconsBold.ticket,
                          ),
                          label: Text(l10n.businessEventRegistrationAction),
                        ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EventInfo extends StatelessWidget {
  const _EventInfo({required this.icon, required this.title, this.subtitle});

  final IconData icon;
  final String title;
  final String? subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 24),
        SizedBox(width: spacing.sp3),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: theme.textTheme.titleMedium),
              if (subtitle case final value?) ...[
                SizedBox(height: spacing.sp1),
                Text(
                  value,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class _Unavailable extends StatelessWidget {
  const _Unavailable();

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.businessEventUnavailableTitle),
          Text(l10n.businessEventUnavailableBody),
        ],
      ),
    );
  }
}

class _LoadError extends StatelessWidget {
  const _LoadError({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.businessEventDetailLoadError),
          TextButton(
            onPressed: onRetry,
            child: Text(l10n.businessEventsRetryAction),
          ),
        ],
      ),
    );
  }
}
