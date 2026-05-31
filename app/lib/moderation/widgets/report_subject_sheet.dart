import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
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
  static const _detailsField = 'details';
  static const _detailsMaxLength = 1000;

  final _formKey = GlobalKey<FormBuilderState>();
  ReportReason? _reason;
  String _details = '';

  bool get _detailsTooLong => _details.length > _detailsMaxLength;
  bool get _canSubmit =>
      _reason != null && !_detailsTooLong && !widget.isSubmitting;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final keyboardInset = MediaQuery.viewInsetsOf(context).bottom;
    return SafeArea(
      child: SingleChildScrollView(
        key: const ValueKey('reportSubjectSheetScrollView'),
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp4,
          spacing.sp4,
          spacing.sp4 + keyboardInset,
        ),
        child: FormBuilder(
          key: _formKey,
          autovalidateMode: AutovalidateMode.onUserInteraction,
          onChanged: _syncFormState,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                switch (widget.subjectType) {
                  ReportSubjectType.post => l10n.postReportAction,
                  ReportSubjectType.profile => l10n.profileReportAction,
                },
                style: theme.textTheme.titleLarge,
              ),
              SizedBox(height: spacing.sp3),
              FormBuilderRadioGroup<ReportReason>(
                name: _reasonField,
                enabled: !widget.isSubmitting,
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
                validator: FormBuilderValidators.required(),
              ),
              SizedBox(height: spacing.sp3),
              FormBuilderField<String>(
                name: _detailsField,
                enabled: !widget.isSubmitting,
                initialValue: '',
                validator: FormBuilderValidators.maxLength(
                  _detailsMaxLength,
                  errorText: l10n.reportDetailsTooLong,
                  checkNullOrEmpty: false,
                ),
                builder: (field) => BrandTextField(
                  label: l10n.reportDetailsLabel,
                  initialValue: field.value ?? '',
                  minLines: 4,
                  maxLines: 4,
                  keyboardType: TextInputType.multiline,
                  textInputAction: TextInputAction.newline,
                  enabled: !widget.isSubmitting,
                  onChanged: field.didChange,
                  errorText: field.errorText,
                ),
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
              SizedBox(height: spacing.sp4),
              ChunkyButton(
                onPressed: _canSubmit ? _submit : null,
                child: widget.isSubmitting
                    ? const StitchProgressIndicator(size: 18)
                    : Text(l10n.reportSubmit),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _syncFormState() {
    _formKey.currentState?.save();
    final values = _formKey.currentState?.value ?? const <String, dynamic>{};
    setState(() {
      _reason = values[_reasonField] as ReportReason?;
      _details = values[_detailsField] as String? ?? '';
    });
    widget.onChanged?.call();
  }

  void _submit() {
    if (!_canSubmit || !(_formKey.currentState?.saveAndValidate() ?? false)) {
      return;
    }
    final values = _formKey.currentState!.value;
    final reason = values[_reasonField]! as ReportReason;
    final details = values[_detailsField] as String? ?? '';

    final trimmed = details.trim();
    widget.onSubmit(
      ReportSubmission(
        reasonType: reason.reasonType,
        details: trimmed.isEmpty ? null : trimmed,
      ),
    );
  }
}
