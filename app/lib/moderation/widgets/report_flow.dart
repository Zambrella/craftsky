import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/report_post_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
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
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    builder: (sheetContext) => ReportSubjectSheet(
      subjectType: ReportSubjectType.post,
      onSubmit: (submission) => _submitPostReport(
        context: context,
        sheetContext: sheetContext,
        ref: ref,
        post: post,
        submission: submission,
        successMessage: successMessage,
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
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    builder: (sheetContext) => ReportSubjectSheet(
      subjectType: ReportSubjectType.profile,
      onSubmit: (submission) => _submitProfileReport(
        context: context,
        sheetContext: sheetContext,
        ref: ref,
        handleOrDid: handleOrDid,
        submission: submission,
        successMessage: successMessage,
      ),
    ),
  );
}

Future<void> _submitPostReport({
  required BuildContext context,
  required BuildContext sheetContext,
  required WidgetRef ref,
  required Post post,
  required ReportSubmission submission,
  required String successMessage,
}) async {
  await ref
      .read(reportPostProvider.notifier)
      .submit(
        did: post.author.did,
        rkey: post.rkey,
        submission: submission,
      );
  switch (ref.read(reportPostProvider)) {
    case AsyncError(:final error):
      throw error;
    case AsyncData(value: != null):
      if (sheetContext.mounted) Navigator.of(sheetContext).pop();
      if (context.mounted) context.showInfo(successMessage);
      ref.read(reportPostProvider.notifier).reset();
    case _:
      break;
  }
}

Future<void> _submitProfileReport({
  required BuildContext context,
  required BuildContext sheetContext,
  required WidgetRef ref,
  required String handleOrDid,
  required ReportSubmission submission,
  required String successMessage,
}) async {
  await ref
      .read(reportProfileProvider.notifier)
      .submit(handleOrDid: handleOrDid, submission: submission);
  switch (ref.read(reportProfileProvider)) {
    case AsyncError(:final error):
      throw error;
    case AsyncData(value: != null):
      if (sheetContext.mounted) Navigator.of(sheetContext).pop();
      if (context.mounted) context.showInfo(successMessage);
      ref.read(reportProfileProvider.notifier).reset();
    case _:
      break;
  }
}
