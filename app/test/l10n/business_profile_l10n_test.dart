import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-012 every business copy category is generated', () {
    final arb =
        jsonDecode(
              File('lib/l10n/app_en.arb').readAsStringSync(),
            )
            as Map<String, dynamic>;
    final generated = AppLocalizationsEn();

    for (final key in [
      'businessProfileLabel',
      'businessProductDestinationHint',
      'businessEventDateRange',
      'businessEventAllDayDisplay',
      'businessLoading',
      'businessSaving',
      'businessImageUploading',
      'businessEventDeleteConfirmTitle',
      'businessEventValidationError',
      'businessEventDiagnosticOwnerNotBusiness',
      'businessEventDiagnosticInvalidTimeRange',
      'businessEventDiagnosticDurationExceedsLimit',
      'businessEventDiagnosticRecordModerated',
      'businessEventDiagnosticEnded',
      'businessEventDiagnosticCancelled',
      'businessEventDiagnosticPostponed',
    ]) {
      expect(arb, contains(key), reason: key);
    }

    expect(generated.businessProfileLabel, 'Business');
    expect(generated.businessImageUploading, 'Uploading image');
    expect(
      generated.businessEventDateRange('Aug 30', 'Sep 1', 2026),
      'Aug 30–Sep 1, 2026',
    );
  });

  test('AT-012 business widgets contain no hard-coded visible copy', () {
    final source = Directory('lib/business')
        .listSync(recursive: true)
        .whereType<File>()
        .where(
          (file) =>
              file.path.contains('/widgets/') || file.path.contains('/pages/'),
        )
        .map((file) => file.readAsStringSync())
        .join('\n');

    for (final literal in [
      "hintText: 'https://",
      "Text('Business')",
      'Text("Business")',
      "labelText: '",
      "tooltip: '",
      "semanticsLabel: '",
    ]) {
      expect(source, isNot(contains(literal)), reason: literal);
    }
  });
}
