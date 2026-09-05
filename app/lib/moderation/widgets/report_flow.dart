import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/business/providers/report_business_event_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/report_post_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/widgets/report_subject_sheet.dart';
import 'package:craftsky_app/profile/providers/report_profile_provider.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<void> showPostReportSheet(
  BuildContext context,
  WidgetRef ref,
  Post post,
) {
  final successMessage = AppLocalizations.of(context).reportSubmitSuccess;
  ref.read(reportPostProvider.notifier).reset();
  return responsiveModalNavigator(context).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (routeContext) => _PostReportRouteBody(
        parentContext: context,
        successMessage: successMessage,
        post: post,
      ),
    ),
  );
}

Future<void> showProfileReportSheet(
  BuildContext context,
  WidgetRef ref,
  String handleOrDid,
) {
  final successMessage = AppLocalizations.of(context).reportSubmitSuccess;
  ref.read(reportProfileProvider.notifier).reset();
  return responsiveModalNavigator(context).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (routeContext) => _ProfileReportRouteBody(
        parentContext: context,
        successMessage: successMessage,
        handleOrDid: handleOrDid,
      ),
    ),
  );
}

Future<void> showBusinessEventReportSheet(
  BuildContext context,
  WidgetRef ref, {
  required AccountKey account,
  required Did owner,
  required RecordKey rkey,
}) {
  final successMessage = AppLocalizations.of(context).reportSubmitSuccess;
  ref.read(reportBusinessEventProvider(account).notifier).reset();
  return responsiveModalNavigator(context).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (routeContext) => _BusinessEventReportRouteBody(
        parentContext: context,
        successMessage: successMessage,
        account: account,
        owner: owner,
        rkey: rkey,
      ),
    ),
  );
}

class _PostReportRouteBody extends ConsumerWidget {
  const _PostReportRouteBody({
    required this.parentContext,
    required this.successMessage,
    required this.post,
  });

  final BuildContext parentContext;
  final String successMessage;
  final Post post;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final submitState = ref.watch(reportPostProvider);
    ref.listen<AsyncValue<ReportResult?>>(reportPostProvider, (_, next) {
      _handleAcceptedReport(
        context: parentContext,
        routeContext: context,
        successMessage: successMessage,
        reset: () => ref.read(reportPostProvider.notifier).reset(),
        state: next,
      );
    });

    return ReportSubjectSheet(
      subjectType: switch (post.reply) {
        null => ReportSubjectType.post,
        final reply when reply.parent.uri == reply.root.uri =>
          ReportSubjectType.comment,
        _ => ReportSubjectType.reply,
      },
      isSubmitting: submitState.isLoading,
      submitError: submitState.hasError
          ? AppLocalizations.of(context).reportSubmitError
          : null,
      onChanged: submitState.hasError
          ? () => ref.read(reportPostProvider.notifier).reset()
          : null,
      onSubmit: (submission) {
        unawaited(
          ref
              .read(reportPostProvider.notifier)
              .submit(
                did: post.author.did,
                rkey: post.rkey,
                submission: submission,
              ),
        );
      },
    );
  }
}

class _ProfileReportRouteBody extends ConsumerWidget {
  const _ProfileReportRouteBody({
    required this.parentContext,
    required this.successMessage,
    required this.handleOrDid,
  });

  final BuildContext parentContext;
  final String successMessage;
  final String handleOrDid;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final submitState = ref.watch(reportProfileProvider);
    ref.listen<AsyncValue<ReportResult?>>(reportProfileProvider, (_, next) {
      _handleAcceptedReport(
        context: parentContext,
        routeContext: context,
        successMessage: successMessage,
        reset: () => ref.read(reportProfileProvider.notifier).reset(),
        state: next,
      );
    });

    return ReportSubjectSheet(
      subjectType: ReportSubjectType.profile,
      isSubmitting: submitState.isLoading,
      submitError: submitState.hasError
          ? AppLocalizations.of(context).reportSubmitError
          : null,
      onChanged: submitState.hasError
          ? () => ref.read(reportProfileProvider.notifier).reset()
          : null,
      onSubmit: (submission) {
        unawaited(
          ref
              .read(reportProfileProvider.notifier)
              .submit(handleOrDid: handleOrDid, submission: submission),
        );
      },
    );
  }
}

class _BusinessEventReportRouteBody extends ConsumerWidget {
  const _BusinessEventReportRouteBody({
    required this.parentContext,
    required this.successMessage,
    required this.account,
    required this.owner,
    required this.rkey,
  });

  final BuildContext parentContext;
  final String successMessage;
  final AccountKey account;
  final Did owner;
  final RecordKey rkey;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final provider = reportBusinessEventProvider(account);
    final submitState = ref.watch(provider);
    ref.listen<AsyncValue<ReportResult?>>(provider, (_, next) {
      if (next case AsyncError(
        :final error,
      ) when error is ApiBadRequest && error.code == 'event_not_found') {
        ref.read(provider.notifier).reset();
        _dismissReportRoute(context);
        return;
      }
      _handleAcceptedReport(
        context: parentContext,
        routeContext: context,
        successMessage: successMessage,
        reset: () => ref.read(provider.notifier).reset(),
        state: next,
      );
    });

    return ReportSubjectSheet(
      subjectType: ReportSubjectType.event,
      isSubmitting: submitState.isLoading,
      submitError: submitState.hasError
          ? AppLocalizations.of(context).reportSubmitError
          : null,
      onChanged: submitState.hasError
          ? () => ref.read(provider.notifier).reset()
          : null,
      onSubmit: (submission) {
        unawaited(
          ref
              .read(provider.notifier)
              .submit(owner: owner, rkey: rkey, submission: submission),
        );
      },
    );
  }
}

void _handleAcceptedReport({
  required BuildContext context,
  required BuildContext routeContext,
  required String successMessage,
  required VoidCallback reset,
  required AsyncValue<ReportResult?> state,
}) {
  if (state case AsyncData(value: != null)) {
    reset();
    if (routeContext.mounted) _dismissReportRoute(routeContext);
    if (context.mounted) _showReportSuccess(context, successMessage);
  }
}

void _dismissReportRoute(BuildContext routeContext) {
  try {
    Navigator.of(routeContext).pop();
  } on Object {
    // The report has already been accepted. Do not turn a best-effort UI
    // dismissal problem into a false submission failure in the route.
  }
}

void _showReportSuccess(BuildContext context, String successMessage) {
  try {
    context.showInfo(successMessage);
  } on Object {
    // The report has already been accepted. A missing/unavailable messenger
    // should not make the report form show a retryable submit failure.
  }
}
