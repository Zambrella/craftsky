import 'dart:async';

import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:material_ui/material_ui.dart';

enum ReportSubmissionState { confirmed, failed, uncertain }

class ReportPage extends StatefulWidget {
  const new({
    required this.onSubmit,
    this.onCheckStatus,
    this.errorReporter = const NoopErrorReporter(),
    super.key,
  });

  final Future<ReportSubmissionState> Function(
    String reasonType,
    String? reason,
  )
  onSubmit;
  final ErrorReporter errorReporter;
  final Future<ReportSubmissionState> Function()? onCheckStatus;

  @override
  State<ReportPage> createState() => _ReportPageState();
}

class _ReportPageState extends State<ReportPage> {
  final _explanation = TextEditingController();
  final _explanationFocus = FocusNode();
  String _reasonType = 'com.atproto.moderation.defs#reasonOther';
  bool _submitting = false;
  bool _submitted = false;
  bool _invalid = false;
  ReportSubmissionState? _result;

  @override
  void dispose() {
    _explanation.dispose();
    _explanationFocus.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_submitted || _submitting) return;
    if (validateReportInput(_reasonType, _explanation.text) != null) {
      setState(() => _invalid = true);
      return;
    }
    setState(() {
      _invalid = false;
      _submitting = true;
      _submitted = true;
    });
    ReportSubmissionState result;
    try {
      result = await widget.onSubmit(
        _reasonType,
        _explanation.text.isEmpty ? null : _explanation.text,
      );
    } on Object catch (error, stack) {
      result = ReportSubmissionState.uncertain;
      unawaited(
        widget.errorReporter.captureException(
          error,
          context: const ReportContext(
            feature: 'Chat',
            operation: 'report',
            classification: 'chat.report',
          ),
          stackTrace: stack,
        ),
      );
    }
    if (!mounted) return;
    setState(() {
      _submitting = false;
      _result = result;
    });
  }

  Future<void> _checkStatus() async {
    final callback = widget.onCheckStatus;
    if (_submitting || callback == null) return;
    setState(() => _submitting = true);
    try {
      final result = await callback();
      if (mounted) setState(() => _result = result);
    } on Object {
      // The operation owner retains the typed cause; status failure cannot
      // confirm success or make another report submission.
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(
          l10n.chatReportTitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(l10n.chatReportDisclosure),
                const SizedBox(height: 16),
                Text(l10n.chatReportEvidenceDisclosure),
                const SizedBox(height: 24),
                CraftskySingleSelectInput<String>(
                  label: l10n.chatReportReasonLabel,
                  keyPrefix: 'chat-report-reason',
                  value: _reasonType,
                  options: [
                    CraftskySelectOption(
                      value: 'com.atproto.moderation.defs#reasonSpam',
                      label: l10n.chatReportReasonSpam,
                    ),
                    CraftskySelectOption(
                      value: 'com.atproto.moderation.defs#reasonViolation',
                      label: l10n.chatReportReasonViolation,
                    ),
                    CraftskySelectOption(
                      value: 'com.atproto.moderation.defs#reasonMisleading',
                      label: l10n.chatReportReasonMisleading,
                    ),
                    CraftskySelectOption(
                      value: 'com.atproto.moderation.defs#reasonSexual',
                      label: l10n.chatReportReasonSexual,
                    ),
                    CraftskySelectOption(
                      value: 'com.atproto.moderation.defs#reasonRude',
                      label: l10n.chatReportReasonRude,
                    ),
                    CraftskySelectOption(
                      value: 'com.atproto.moderation.defs#reasonOther',
                      label: l10n.chatReportReasonOther,
                    ),
                  ],
                  onChanged: _submitted
                      ? null
                      : (value) {
                          if (value != null) {
                            setState(() => _reasonType = value);
                          }
                        },
                ),
                const SizedBox(height: 16),
                CraftskyFieldScaffold(
                  label: l10n.chatReportExplanationLabel,
                  focusNode: _explanationFocus,
                  errorText: _invalid
                      ? l10n.chatReportInvalidExplanation
                      : null,
                  child: TextField(
                    key: const Key('chat-report-explanation'),
                    controller: _explanation,
                    focusNode: _explanationFocus,
                    minLines: 3,
                    maxLines: 6,
                    readOnly:
                        _submitting ||
                        _result == ReportSubmissionState.confirmed,
                  ),
                ),
                const SizedBox(height: 24),
                ChunkyButton(
                  onPressed: _submitted ? null : _submit,
                  child: ActionProgress(
                    isLoading: _submitting && _result == null,
                    child: Text(l10n.chatReportAction),
                  ),
                ),
                if (_result == ReportSubmissionState.uncertain &&
                    widget.onCheckStatus != null)
                  TextButton(
                    onPressed: _submitting
                        ? null
                        : () => unawaited(_checkStatus()),
                    child: ActionProgress(
                      isLoading: _submitting,
                      child: Text(l10n.chatCheckStatus),
                    ),
                  ),
                if (_result case final result?) ...[
                  const SizedBox(height: 16),
                  ChatMessageEffect(
                    severity: switch (result) {
                      ReportSubmissionState.confirmed => MessageSeverity.info,
                      ReportSubmissionState.failed => MessageSeverity.error,
                      ReportSubmissionState.uncertain =>
                        MessageSeverity.warning,
                    },
                    message: switch (result) {
                      ReportSubmissionState.confirmed =>
                        l10n.chatReportConfirmed,
                      ReportSubmissionState.failed => l10n.chatReportFailed,
                      ReportSubmissionState.uncertain =>
                        l10n.chatReportUncertain,
                    },
                    child: Text(switch (result) {
                      ReportSubmissionState.confirmed =>
                        l10n.chatReportConfirmed,
                      ReportSubmissionState.failed => l10n.chatReportFailed,
                      ReportSubmissionState.uncertain =>
                        l10n.chatReportUncertain,
                    }),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}
