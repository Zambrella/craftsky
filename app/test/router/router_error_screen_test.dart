import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/error_screen.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('router error screen hides raw routing errors', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: ErrorScreen(
          error: StateError('route failed for did:plc:alice /profile?x=secret'),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Something went wrong'), findsOneWidget);
    expect(find.text("That page couldn't be opened."), findsOneWidget);
    expect(find.textContaining('route failed'), findsNothing);
    expect(find.textContaining('did:plc:alice'), findsNothing);
    expect(find.textContaining('/profile?x=secret'), findsNothing);
    expect(find.widgetWithText(ElevatedButton, 'Go home'), findsOneWidget);
  });
}
