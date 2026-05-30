import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:flutter/material.dart';

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
  ReportReason? _reason;
  String _details = '';
  bool _isSubmitting = false;
  String? _submitError;

  bool get _detailsTooLong => _details.length > 1000;
  bool get _canSubmit => _reason != null && !_detailsTooLong && !_isSubmitting;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              switch (widget.subjectType) {
                ReportSubjectType.post => l10n.postReportAction,
                ReportSubjectType.profile => l10n.profileReportAction,
              },
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            for (final reason in ReportReason.values)
              RadioListTile<ReportReason>(
                title: Text(reason.label(l10n)),
                value: reason,
                groupValue: _reason,
                onChanged: _isSubmitting
                    ? null
                    : (value) => setState(() {
                        _reason = value;
                        _submitError = null;
                      }),
              ),
            const SizedBox(height: 12),
            TextField(
              maxLines: 4,
              enabled: !_isSubmitting,
              decoration: InputDecoration(
                labelText: l10n.reportDetailsLabel,
                errorText: _detailsTooLong ? l10n.reportDetailsTooLong : null,
              ),
              onChanged: (value) => setState(() {
                _details = value;
                _submitError = null;
              }),
            ),
            if (_submitError case final error?) ...[
              const SizedBox(height: 8),
              Text(
                error,
                style:
                    Theme.of(
                      context,
                    ).textTheme.bodyMedium?.copyWith(
                      color: Theme.of(context).colorScheme.error,
                    ),
              ),
            ],
            const SizedBox(height: 16),
            FilledButton(
              onPressed: _canSubmit ? _submit : null,
              child: Text(
                _isSubmitting ? l10n.reportSubmitting : l10n.reportSubmit,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final reason = _reason;
    if (reason == null || !_canSubmit) return;
    setState(() {
      _isSubmitting = true;
      _submitError = null;
    });
    try {
      final trimmed = _details.trim();
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
