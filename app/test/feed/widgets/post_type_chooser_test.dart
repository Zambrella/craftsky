import 'dart:async';

import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-001 compact chooser opens project composer', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => Scaffold(
                body: Center(
                  child: ElevatedButton(
                    onPressed: () {
                      unawaited(
                        showTopLevelPostComposerChooser(
                          context,
                          position: RelativeRect.fill,
                        ),
                      );
                    },
                    child: const Text('New post'),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('New post'));
    await tester.pumpAndSettle();

    expect(find.text('Regular post'), findsOneWidget);
    expect(find.text('Project post'), findsOneWidget);

    await tester.tap(find.text('Project post'));
    await tester.pumpAndSettle();

    expect(find.text('Project post'), findsOneWidget);
    expect(find.byKey(const Key('craftType-select-button')), findsOneWidget);
  });

  testWidgets('AT-002 regular branch opens the existing composer', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => Scaffold(
                body: Center(
                  child: ElevatedButton(
                    onPressed: () {
                      unawaited(
                        showTopLevelPostComposerChooser(
                          context,
                          position: RelativeRect.fill,
                        ),
                      );
                    },
                    child: const Text('New post'),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('New post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Regular post'));
    await tester.pumpAndSettle();

    expect(find.text('New post'), findsOneWidget);
    expect(find.text('What are you making?'), findsOneWidget);
    expect(find.text('Craft type'), findsNothing);
    expect(find.text('Project post'), findsNothing);
  });
}
