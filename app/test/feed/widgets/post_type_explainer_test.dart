import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../fakes/auth_session_fakes.dart';

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  for (final label in ['Regular post', 'Project post']) {
    testWidgets('$label selection persists for only that account', (
      tester,
    ) async {
      final preferences = await SharedPreferences.getInstance();
      final launches = <String>[];
      final first = await _container(preferences, 'did:plc:alice');
      addTearDown(first.dispose);
      await _pump(tester, first, launches);
      await _open(tester);
      expect(find.byType(CraftskyDialog), findsOneWidget);
      expect(
        find.text(
          'Share a quick update, ask a question, or show what’s inspiring you.',
        ),
        findsOneWidget,
      );
      expect(
        find.text(
          'Document something you’re making, with details like the craft, '
          'pattern, materials, and progress.',
        ),
        findsOneWidget,
      );
      await tester.tap(find.text(label));
      await tester.pumpAndSettle();
      expect(launches, [label]);
      expect(find.byType(CraftskyDialog), findsNothing);

      // Recreate the container and reload local preferences as on restart.
      await tester.pumpWidget(const SizedBox());
      await preferences.reload();
      final reopened = await _container(preferences, 'did:plc:alice');
      addTearDown(reopened.dispose);
      await _pump(tester, reopened, launches);
      await _open(tester);
      expect(find.byType(CraftskyDialog), findsNothing);
      expect(find.text('Regular post'), findsOneWidget);
      expect(find.text('Project post'), findsOneWidget);
      await tester.tap(find.text(label));
      await tester.pumpAndSettle();
      expect(launches, [label, label]);

      await tester.pumpWidget(const SizedBox());
      final second = await _container(preferences, 'did:plc:bob');
      addTearDown(second.dispose);
      await _pump(tester, second, launches);
      await _open(tester);
      expect(find.byType(CraftskyDialog), findsOneWidget);
    });
  }

  for (final dismissal in ['Cancel', 'barrier', 'back']) {
    testWidgets('$dismissal leaves the explainer available next time', (
      tester,
    ) async {
      final preferences = await SharedPreferences.getInstance();
      final container = await _container(preferences, 'did:plc:alice');
      addTearDown(container.dispose);
      final launches = <String>[];
      await _pump(tester, container, launches);
      await _open(tester);
      switch (dismissal) {
        case 'Cancel':
          await tester.tap(find.text('Cancel'));
        case 'barrier':
          await tester.tapAt(const Offset(5, 5));
        case 'back':
          await tester.binding.handlePopRoute();
      }
      await tester.pumpAndSettle();
      expect(find.byType(CraftskyDialog), findsNothing);
      expect(launches, isEmpty);
      expect(preferences.getKeys(), isEmpty);
      await _open(tester);
      expect(find.byType(CraftskyDialog), findsOneWidget);
    });
  }

  testWidgets('account switch while explaining does not launch a composer', (
    tester,
  ) async {
    final preferences = await SharedPreferences.getInstance();
    final container = await _container(preferences, 'did:plc:alice');
    addTearDown(container.dispose);
    final launches = <String>[];
    await _pump(tester, container, launches);
    await _open(tester);
    (container.read(authSessionProvider.notifier) as _SwitchableAuthSession)
        .switchAccount('did:plc:bob');
    await tester.tap(find.text('Project post'));
    await tester.pumpAndSettle();
    expect(launches, isEmpty);
    expect(preferences.getKeys(), isEmpty);
    await _open(tester);
    expect(find.byType(CraftskyDialog), findsOneWidget);
  });

  testWidgets('options remain usable on a narrow screen with large text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 568);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    tester.platformDispatcher.textScaleFactorTestValue = 2;
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
    final container = await _container(
      await SharedPreferences.getInstance(),
      'did:plc:alice',
    );
    addTearDown(container.dispose);
    final launches = <String>[];
    await _pump(tester, container, launches);
    await _open(tester);
    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.text('Project post'));
    await tester.tap(find.text('Project post'));
    await tester.pumpAndSettle();
    expect(launches, ['Project post']);
    expect(tester.takeException(), isNull);
  });
}

Future<ProviderContainer> _container(
  SharedPreferences preferences,
  String did,
) async {
  final container = ProviderContainer(
    overrides: [
      sharedPreferencesProvider.overrideWithValue(preferences),
      authSessionProvider.overrideWith(() => _SwitchableAuthSession(did: did)),
    ],
  );
  await container.read(authSessionProvider.future);
  return container;
}

Future<void> _pump(
  WidgetTester tester,
  ProviderContainer container,
  List<String> launches,
) => tester.pumpWidget(
  UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: appLocalizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Builder(
        builder: (context) => Scaffold(
          body: Center(
            child: ElevatedButton(
              onPressed: () async {
                await showTopLevelPostComposerChooser(
                  context,
                  position: RelativeRect.fill,
                  showRegularComposer: (_) async {
                    launches.add('Regular post');
                    return null;
                  },
                  showProjectComposer: (_) async {
                    launches.add('Project post');
                    return null;
                  },
                );
              },
              child: const Text('New post'),
            ),
          ),
        ),
      ),
    ),
  ),
);

Future<void> _open(WidgetTester tester) async {
  await tester.tap(find.text('New post'));
  await tester.pumpAndSettle();
}

class _SwitchableAuthSession extends SignedInAuthSession {
  new({required super.did});

  void switchAccount(String did) {
    state = AsyncData(SignedIn(did: did, handle: 'test.bsky.social'));
  }
}
