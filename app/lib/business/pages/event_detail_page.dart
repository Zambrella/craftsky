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
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
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
    return Scaffold(
      appBar: AppBar(title: Text(l10n.businessEventDetailTitle)),
      body: switch (detail) {
        AsyncData(value: BusinessEventDetailAvailable(:final event)) =>
          _EventDetail(
            account: account,
            event: event,
            launchExternal: launchExternal,
            confirmExternal: confirmExternal,
          ),
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
    );
  }
}

class _EventDetail extends ConsumerWidget {
  const _EventDetail({
    required this.account,
    required this.event,
    required this.launchExternal,
    required this.confirmExternal,
  });

  final AccountKey account;
  final BusinessEvent event;
  final ExternalLinkLauncher launchExternal;
  final ExternalLinkConfirmer confirmExternal;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final display = BusinessFormatters.event(event, l10n);
    final eventDestination = hydratedExternalActionUri(event.eventUri);
    final registrationDestination = hydratedExternalActionUri(
      event.registrationUri,
    );

    return SingleChildScrollView(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: Padding(
            padding: EdgeInsets.all(spacing.sp4),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (event.image case final image?) ...[
                  Semantics(
                    image: true,
                    label: image.alt,
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(
                        theme.extension<RadiusTheme>()!.r2,
                      ),
                      child: AspectRatio(
                        aspectRatio: switch (image.aspectRatio) {
                          final ratio? => ratio.width / ratio.height,
                          null => 16 / 9,
                        },
                        child: BusinessImage(
                          image: image,
                          networkUrl: image.fullsize,
                          fit: BoxFit.cover,
                        ),
                      ),
                    ),
                  ),
                  SizedBox(height: spacing.sp4),
                ],
                Text(event.name, style: theme.textTheme.headlineSmall),
                if (event.summary case final summary?) ...[
                  SizedBox(height: spacing.sp2),
                  Text(summary),
                ],
                SizedBox(height: spacing.sp4),
                _DetailRow(
                  label: l10n.businessEventDateLabel,
                  value: display.date,
                ),
                _DetailRow(
                  label: l10n.businessEventTimeLabel,
                  value: display.time,
                ),
                _DetailRow(
                  label: l10n.businessEventRolesLabel,
                  value: event.roles
                      .map((role) => BusinessLabels.eventRole(role, l10n))
                      .join(', '),
                ),
                if (event.mode case final mode?)
                  _DetailRow(
                    label: l10n.businessEventModeLabel,
                    value: BusinessLabels.eventMode(mode, l10n),
                  ),
                _DetailRow(
                  label: l10n.businessEventStatusLabel,
                  value: BusinessLabels.eventStatus(event.status, l10n),
                ),
                _DetailRow(
                  label: l10n.businessEventLifecycleLabel,
                  value: event.past
                      ? l10n.businessEventLifecyclePast
                      : l10n.businessEventLifecycleUpcoming,
                ),
                if (event.timeZone case final timeZone?)
                  _DetailRow(
                    label: l10n.businessEventTimeZoneLabel,
                    value: timeZone,
                  ),
                if (event.venueName case final venue?)
                  _DetailRow(
                    label: l10n.businessEventVenueLabel,
                    value: venue,
                  ),
                _DetailRow(
                  label: l10n.businessEventPublishedLabel,
                  value: DateFormat.yMMMd(
                    l10n.localeName,
                  ).format(event.createdAt.toLocal()),
                ),
                if (eventDestination != null || registrationDestination != null)
                  Padding(
                    padding: EdgeInsets.only(top: spacing.sp3),
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
                            icon: const Icon(Icons.open_in_new),
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
                            icon: const Icon(Icons.open_in_new),
                            label: Text(l10n.businessEventRegistrationAction),
                          ),
                      ],
                    ),
                  ),
                if (account.did != event.did)
                  Padding(
                    padding: EdgeInsets.only(top: spacing.sp3),
                    child: Align(
                      alignment: AlignmentDirectional.centerStart,
                      child: OutlinedButton.icon(
                        onPressed: () => unawaited(
                          showBusinessEventReportSheet(
                            context,
                            ref,
                            account: account,
                            owner: event.did,
                            rkey: event.rkey,
                          ),
                        ),
                        icon: const Icon(Icons.flag_outlined),
                        label: Text(l10n.businessEventReportActionShort),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _DetailRow extends StatelessWidget {
  const _DetailRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 4),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 104,
          child: Text(
            label,
            style: Theme.of(context).textTheme.labelLarge,
          ),
        ),
        Expanded(child: Text(value)),
      ],
    ),
  );
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
