import 'dart:async';

import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/form_action_layout.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:material_ui/material_ui.dart';

enum ReportSubmissionState { confirmed, failed, uncertain }

class ReportPage extends StatefulWidget {
  const new({
    required this.onSubmit,
    this.onCheckStatus,
    this.onClose,
    this.isConversation = false,
    this.errorReporter = const NoopErrorReporter(),
    super.key,
  });

  final Future<ReportSubmissionState> Function(
    String reasonType,
    String? reason,
  )
  onSubmit;
  final VoidCallback? onClose;
  final bool isConversation;
  final ErrorReporter errorReporter;
  final Future<ReportSubmissionState> Function()? onCheckStatus;

  @override
  State<ReportPage> createState() => _ReportPageState();
}

class _ReportPageState extends State<ReportPage> {
  final _explanation = TextEditingController();
  final _explanationFocus = FocusNode();
  String? _reasonType;
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
    final reasonType = _reasonType;
    if (_submitted || _submitting || reasonType == null) return;
    if (validateReportInput(reasonType, _explanation.text) != null) {
      setState(() => _invalid = true);
      return;
    }
    _explanationFocus.unfocus();
    setState(() {
      _invalid = false;
      _submitting = true;
      _submitted = true;
    });
    ReportSubmissionState result;
    try {
      result = await widget.onSubmit(
        reasonType,
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
        leading: widget.onClose == null
            ? null
            : CloseButton(onPressed: widget.onClose),
        title: Text(
          widget.isConversation
              ? l10n.chatReportConversationTitle
              : l10n.chatReportTitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
      ),
      body: FormActionLayout(
        keyboardVisible: MediaQuery.viewInsetsOf(context).bottom > 0,
        label: l10n.reportSubmit,
        actionKey: const Key('chat-report-submit'),
        onPressed: _submitted || _reasonType == null ? null : _submit,
        actionChild: ActionProgress(
          isLoading: _submitting && _result == null,
          child: Text(l10n.reportSubmit),
        ),
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    widget.isConversation
                        ? l10n.chatConversationReportDisclosure
                        : l10n.chatReportDisclosure,
                  ),
                  const SizedBox(height: 24),
                  CraftskySingleSelectInput<String>(
                    enabled: !_submitted,
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
                  BrandTextField(
                    label: l10n.chatReportExplanationLabel,
                    textFieldKey: const Key('chat-report-explanation'),
                    controller: _explanation,
                    focusNode: _explanationFocus,
                    errorText: _invalid
                        ? l10n.chatReportInvalidExplanation
                        : null,
                    minLines: 3,
                    maxLines: 6,
                    keyboardType: TextInputType.multiline,
                    textInputAction: TextInputAction.newline,
                    enabled:
                        !_submitting &&
                        _result != ReportSubmissionState.confirmed,
                    onChanged: (_) {
                      if (_invalid) {
                        setState(
                          () => _invalid =
                              validateReportInput(
                                _reasonType!,
                                _explanation.text,
                              ) !=
                              null,
                        );
                      }
                    },
                  ),
                  const SizedBox(height: 24),
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
      ),
    );
  }
}
