import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:form_builder_validators/form_builder_validators.dart';

enum ReportSubjectType { post, profile }

class ReportSubjectSheet extends StatefulWidget {
  const ReportSubjectSheet({
    required this.subjectType,
    required this.onSubmit,
    super.key,
    this.isSubmitting = false,
    this.submitError,
    this.onChanged,
  });

  final ReportSubjectType subjectType;
  final void Function(ReportSubmission submission) onSubmit;
  final bool isSubmitting;
  final String? submitError;
  final VoidCallback? onChanged;

  @override
  State<ReportSubjectSheet> createState() => _ReportSubjectSheetState();
}

class _ReportSubjectSheetState extends State<ReportSubjectSheet> {
  static const _reasonField = 'reason';
  static const _detailsMaxLength = 1000;

  final _formKey = GlobalKey<FormBuilderState>();
  final _detailsController = TextEditingController();
  final _detailsFocusNode = FocusNode(debugLabel: 'reportDetails');
  ReportReason? _reason;
  String _details = '';

  bool get _detailsTooLong => _details.length > _detailsMaxLength;
  bool get _canSubmit =>
      _reason != null && !_detailsTooLong && !widget.isSubmitting;

  @override
  void dispose() {
    _detailsController.dispose();
    _detailsFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final title = switch (widget.subjectType) {
      ReportSubjectType.post => l10n.postReportAction,
      ReportSubjectType.profile => l10n.profileReportAction,
    };
    return Scaffold(
      backgroundColor: swatches.paper,
      appBar: AppBar(
        title: Text(title),
        actions: [
          Padding(
            padding: EdgeInsets.only(right: spacing.sp3),
            child: _ReportSubmitAction(
              isSubmitting: widget.isSubmitting,
              onPressed: _canSubmit ? _submit : null,
            ),
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        bottom: false,
        child: SingleChildScrollView(
          key: const ValueKey('reportSubjectRouteScrollView'),
          padding: EdgeInsets.fromLTRB(
            spacing.sp4,
            spacing.sp4,
            spacing.sp4,
            spacing.sp6,
          ),
          child: FormBuilder(
            key: _formKey,
            autovalidateMode: AutovalidateMode.onUserInteraction,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  l10n.reportReasonTitle,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w800,
                  ),
                ),
                SizedBox(height: spacing.sp2),
                FormBuilderRadioGroup<ReportReason>(
                  name: _reasonField,
                  enabled: !widget.isSubmitting,
                  onChanged: (reason) {
                    setState(() => _reason = reason);
                    widget.onChanged?.call();
                  },
                  decoration: const InputDecoration(
                    border: InputBorder.none,
                    contentPadding: EdgeInsets.zero,
                    filled: false,
                  ),
                  options: [
                    for (final reason in ReportReason.values)
                      FormBuilderFieldOption<ReportReason>(
                        value: reason,
                        child: Text(reason.label(l10n)),
                      ),
                  ],
                  orientation: OptionsOrientation.vertical,
                  validator: FormBuilderValidators.required(),
                ),
                SizedBox(height: spacing.sp3),
                BrandTextField(
                  label: l10n.reportDetailsLabel,
                  controller: _detailsController,
                  focusNode: _detailsFocusNode,
                  helperText: '${_details.length}/$_detailsMaxLength',
                  helperAlignment: AlignmentDirectional.centerEnd,
                  minLines: 4,
                  maxLines: 4,
                  keyboardType: TextInputType.multiline,
                  textInputAction: TextInputAction.newline,
                  enabled: !widget.isSubmitting,
                  onChanged: (value) {
                    setState(() => _details = value);
                    widget.onChanged?.call();
                  },
                  errorText: _detailsTooLong ? l10n.reportDetailsTooLong : null,
                ),
                if (widget.submitError case final error?) ...[
                  SizedBox(height: spacing.sp2),
                  Text(
                    error,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.error,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _submit() {
    if (!_canSubmit) return;

    final trimmed = _details.trim();
    widget.onSubmit(
      ReportSubmission(
        reasonType: _reason!.reasonType,
        details: trimmed.isEmpty ? null : trimmed,
      ),
    );
  }
}

class _ReportSubmitAction extends StatelessWidget {
  const _ReportSubmitAction({
    required this.isSubmitting,
    required this.onPressed,
  });

  final bool isSubmitting;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        textStyle: const TextStyle(fontWeight: FontWeight.w800),
      ),
      child: isSubmitting
          ? const StitchProgressIndicator(size: 18)
          : Text(l10n.reportSubmit),
    );
  }
}
