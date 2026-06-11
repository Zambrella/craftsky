import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-005 shows sewing detail fields for sewing projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');

    expect(find.text('Sewing project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Size made'), findsOneWidget);
    expect(find.text('Fit notes'), findsOneWidget);
    expect(find.text('Yarn weight'), findsNothing);
    expect(find.text('Needle size'), findsNothing);
    expect(find.text('Hook size'), findsNothing);
  });

  testWidgets('AT-005 shows knitting detail fields for knitting projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Knitting');

    expect(find.text('Knitting project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Yarn weight'), findsOneWidget);
    expect(find.text('Needle size'), findsOneWidget);
    expect(find.text('Gauge stitches'), findsOneWidget);
    expect(find.text('Gauge rows'), findsOneWidget);
    expect(find.text('Gauge measurement'), findsOneWidget);
    expect(find.text('Gauge unit'), findsOneWidget);
    expect(find.text('Finished size'), findsOneWidget);
    expect(find.text('Fit notes'), findsNothing);
    expect(find.text('Hook size'), findsNothing);
  });

  testWidgets('AT-005 shows crochet detail fields for crochet projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Crochet');

    expect(find.text('Crochet project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Yarn weight'), findsOneWidget);
    expect(find.text('Hook size'), findsOneWidget);
    expect(find.text('Gauge stitches'), findsOneWidget);
    expect(find.text('Gauge rows'), findsOneWidget);
    expect(find.text('Gauge measurement'), findsOneWidget);
    expect(find.text('Gauge unit'), findsOneWidget);
    expect(find.text('Finished size'), findsOneWidget);
    expect(find.text('Needle size'), findsNothing);
    expect(find.text('Fit notes'), findsNothing);
  });

  testWidgets('AT-005 shows quilting detail fields for quilting projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Quilting');

    expect(find.text('Quilting project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Size'), findsOneWidget);
    expect(find.text('Piecing technique'), findsOneWidget);
    expect(find.text('Quilting method'), findsOneWidget);
    expect(find.text('Yarn weight'), findsNothing);
    expect(find.text('Gauge stitches'), findsNothing);
    expect(find.text('Fit notes'), findsNothing);
  });

  testWidgets('AT-005 filters and clears project subtype selections', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');

    await tester.ensureVisible(find.byType(DropdownButton<String>).at(2));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButton<String>).at(2));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Garment').last);
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.byType(DropdownButton<String>).at(3));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButton<String>).at(3));
    await tester.pumpAndSettle();
    expect(find.text('Dress'), findsOneWidget);
    expect(find.text('Bag'), findsNothing);
    await tester.tap(find.text('Dress').last);
    await tester.pumpAndSettle();
    expect(find.text('Dress'), findsOneWidget);

    await tester.ensureVisible(find.byType(DropdownButton<String>).at(2));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButton<String>).at(2));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Accessory').last);
    await tester.pumpAndSettle();

    expect(find.text('Dress'), findsNothing);
    await tester.ensureVisible(find.byType(DropdownButton<String>).at(3));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButton<String>).at(3));
    await tester.pumpAndSettle();
    expect(find.text('Bag'), findsOneWidget);
    expect(find.text('Dress'), findsNothing);
  });
}

Future<void> _openDetailsForCraft(WidgetTester tester, String craft) async {
  await tester.pumpWidget(
    ProviderScope(
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectComposerSheet(),
        ),
      ),
    ),
  );

  final craftDropdown = find.byType(DropdownButton<String>).first;
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craft).last);
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('More project details'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('More project details'));
  await tester.pumpAndSettle();
}
