import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
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
  });

  final ReportSubjectType subjectType;
  final Future<void> Function(ReportSubmission submission) onSubmit;

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
  bool _isSubmitting = false;
  String? _submitError;

  bool get _detailsTooLong => _details.length > _detailsMaxLength;
  bool get _canSubmit => _reason != null && !_detailsTooLong && !_isSubmitting;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return SafeArea(
      child: SingleChildScrollView(
        padding: EdgeInsets.all(spacing.sp4),
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
                enabled: !_isSubmitting,
                decoration: const InputDecoration(
                  border: InputBorder.none,
                  contentPadding: EdgeInsets.zero,
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
              FormBuilderTextField(
                name: _detailsField,
                maxLines: 4,
                enabled: !_isSubmitting,
                decoration: InputDecoration(labelText: l10n.reportDetailsLabel),
                validator: FormBuilderValidators.maxLength(
                  _detailsMaxLength,
                  errorText: l10n.reportDetailsTooLong,
                  checkNullOrEmpty: false,
                ),
              ),
              if (_submitError case final error?) ...[
                SizedBox(height: spacing.sp2),
                Text(
                  error,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.error,
                  ),
                ),
              ],
              SizedBox(height: spacing.sp4),
              FilledButton(
                onPressed: _canSubmit ? _submit : null,
                child: Text(
                  _isSubmitting ? l10n.reportSubmitting : l10n.reportSubmit,
                ),
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
      _submitError = null;
    });
  }

  Future<void> _submit() async {
    if (!_canSubmit || !(_formKey.currentState?.saveAndValidate() ?? false)) {
      return;
    }
    final values = _formKey.currentState!.value;
    final reason = values[_reasonField]! as ReportReason;
    final details = values[_detailsField] as String? ?? '';

    setState(() {
      _isSubmitting = true;
      _submitError = null;
    });
    try {
      final trimmed = details.trim();
      await widget.onSubmit(
        ReportSubmission(
          reasonType: reason.reasonType,
          details: trimmed.isEmpty ? null : trimmed,
        ),
      );
    } catch (_) {
      if (mounted) {
        setState(
          () => _submitError = AppLocalizations.of(context).reportSubmitError,
        );
      }
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }
}
