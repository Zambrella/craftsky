import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('loading retains the action footprint and accessible label', (
    tester,
  ) async {
    Future<void> pump({required bool loading}) => tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: Center(
            child: TextButton(
              onPressed: loading ? null : () {},
              child: ActionProgress(
                isLoading: loading,
                child: const Text('Save changes'),
              ),
            ),
          ),
        ),
      ),
    );
    final semantics = tester.ensureSemantics();
    await pump(loading: false);
    final size = tester.getSize(find.byType(TextButton));
    expect(find.byType(StitchProgressIndicator), findsNothing);
    await pump(loading: true);
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(tester.getSize(find.byType(TextButton)), size);
    expect(find.bySemanticsLabel(RegExp('Save changes')), findsWidgets);
    expect(
      tester.widget<TextButton>(find.byType(TextButton)).onPressed,
      isNull,
    );
    await pump(loading: false);
    expect(find.byType(StitchProgressIndicator), findsNothing);
    semantics.dispose();
  });
}
