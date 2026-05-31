import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/report_post_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/widgets/report_subject_sheet.dart';
import 'package:craftsky_app/profile/providers/report_profile_provider.dart';
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
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useRootNavigator: true,
    builder: (sheetContext) => Consumer(
      builder: (sheetContext, sheetRef, _) {
        final submitState = sheetRef.watch(reportPostProvider);
        sheetRef.listen<AsyncValue<ReportResult?>>(reportPostProvider, (
          _,
          next,
        ) {
          _handleAcceptedReport(
            context: context,
            sheetContext: sheetContext,
            successMessage: successMessage,
            reset: () => sheetRef.read(reportPostProvider.notifier).reset(),
            state: next,
          );
        });

        return ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          isSubmitting: submitState.isLoading,
          submitError: submitState.hasError
              ? AppLocalizations.of(sheetContext).reportSubmitError
              : null,
          onChanged: submitState.hasError
              ? () => sheetRef.read(reportPostProvider.notifier).reset()
              : null,
          onSubmit: (submission) {
            unawaited(
              sheetRef
                  .read(reportPostProvider.notifier)
                  .submit(
                    did: post.author.did,
                    rkey: post.rkey,
                    submission: submission,
                  ),
            );
          },
        );
      },
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
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useRootNavigator: true,
    builder: (sheetContext) => Consumer(
      builder: (sheetContext, sheetRef, _) {
        final submitState = sheetRef.watch(reportProfileProvider);
        sheetRef.listen<AsyncValue<ReportResult?>>(reportProfileProvider, (
          _,
          next,
        ) {
          _handleAcceptedReport(
            context: context,
            sheetContext: sheetContext,
            successMessage: successMessage,
            reset: () => sheetRef.read(reportProfileProvider.notifier).reset(),
            state: next,
          );
        });

        return ReportSubjectSheet(
          subjectType: ReportSubjectType.profile,
          isSubmitting: submitState.isLoading,
          submitError: submitState.hasError
              ? AppLocalizations.of(sheetContext).reportSubmitError
              : null,
          onChanged: submitState.hasError
              ? () => sheetRef.read(reportProfileProvider.notifier).reset()
              : null,
          onSubmit: (submission) {
            unawaited(
              sheetRef
                  .read(reportProfileProvider.notifier)
                  .submit(
                    handleOrDid: handleOrDid,
                    submission: submission,
                  ),
            );
          },
        );
      },
    ),
  );
}

void _handleAcceptedReport({
  required BuildContext context,
  required BuildContext sheetContext,
  required String successMessage,
  required VoidCallback reset,
  required AsyncValue<ReportResult?> state,
}) {
  if (state case AsyncData(value: != null)) {
    reset();
    if (sheetContext.mounted) _dismissReportSheet(sheetContext);
    if (context.mounted) _showReportSuccess(context, successMessage);
  }
}

void _dismissReportSheet(BuildContext sheetContext) {
  try {
    Navigator.of(sheetContext, rootNavigator: true).pop();
  } on Object {
    // The report has already been accepted. Do not turn a best-effort UI
    // dismissal problem into a false submission failure in the sheet.
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
