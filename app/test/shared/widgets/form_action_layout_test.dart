import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/shared/widgets/form_action_layout.dart';
import 'package:craftsky_app/shared/widgets/form_keyboard_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  Widget harness(Widget child) => MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  );

  testWidgets(
    'Done remains enabled for an invalid form and only dismisses editing',
    (tester) async {
      final focus = FocusNode();
      addTearDown(focus.dispose);
      await tester.pumpWidget(
        harness(
          Scaffold(
            body: FormActionLayout(
              keyboardVisible: true,
              label: 'Submit',
              onPressed: null,
              actionKey: const Key('submit'),
              contentBuilder: (_, _) => TextField(focusNode: focus),
            ),
          ),
        ),
      );
      await tester.tap(find.byType(TextField));
      await tester.pump();
      expect(focus.hasFocus, isTrue);
      await tester.tap(find.byKey(const Key('form-keyboard-done')));
      await tester.pumpAndSettle();
      expect(focus.hasFocus, isFalse);
      expect(find.byKey(const Key('submit')), findsNothing);
    },
  );

  testWidgets('Done requests a keyboard hide without a focused input', (
    tester,
  ) async {
    var submitted = false;
    await tester.pumpWidget(
      harness(
        Scaffold(
          body: FormActionLayout(
            keyboardVisible: true,
            label: 'Save',
            onPressed: () => submitted = true,
            actionKey: const Key('save'),
            contentBuilder: (_, _) => const Text('No focused input'),
          ),
        ),
      ),
    );
    tester.testTextInput.log.clear();
    await tester.tap(find.byKey(const Key('form-keyboard-done')));
    await tester.pump();

    expect(
      tester.testTextInput.log.map((call) => call.method),
      contains('TextInput.hide'),
    );
    expect(submitted, isFalse);
  });

  testWidgets('the form scrolls behind Done and clears it at the end', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 700);
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    addTearDown(tester.view.reset);
    final controller = ScrollController();
    addTearDown(controller.dispose);
    const fieldKey = Key('last-field');
    await tester.pumpWidget(
      harness(
        Scaffold(
          body: FormActionLayout(
            keyboardVisible: true,
            label: 'Save',
            onPressed: () {},
            actionKey: const Key('save'),
            contentBuilder: (_, clearance) => SingleChildScrollView(
              controller: controller,
              padding: EdgeInsets.only(bottom: clearance),
              child: const Column(
                children: [
                  SizedBox(height: 600),
                  TextField(key: fieldKey),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final done = find.byKey(const Key('form-keyboard-done'));
    final viewport = find.byType(SingleChildScrollView);
    expect(tester.getRect(viewport).bottom, 400);
    expect(tester.getRect(done).bottom, lessThan(400));
    final buttonPosition = tester.getRect(done);

    controller.jumpTo(controller.position.maxScrollExtent);
    await tester.pumpAndSettle();
    expect(tester.getRect(done), buttonPosition);
    expect(
      tester.getRect(find.byKey(fieldKey)).bottom,
      lessThanOrEqualTo(buttonPosition.top),
    );
  });

  for (final canPop in [true, false]) {
    testWidgets(
      'keyboard cleanup respects a '
      '${canPop ? 'completed' : 'rejected'} composer pop',
      (tester) async {
        final focus = FocusNode();
        addTearDown(focus.dispose);
        final navigator = GlobalKey<NavigatorState>();
        await tester.pumpWidget(
          MaterialApp(
            navigatorKey: navigator,
            theme: AppTheme.lightThemeData,
            home: const Scaffold(body: Text('Home')),
          ),
        );
        navigator.currentState!.push<void>(
          FullscreenModalRoute<void>(
            builder: (_) => PopScope(
              canPop: canPop,
              child: Scaffold(body: TextField(focusNode: focus)),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.tap(find.byType(TextField));
        await tester.pump();
        expect(tester.testTextInput.isVisible, isTrue);
        await navigator.currentState!.maybePop();
        await tester.pumpAndSettle();
        expect(focus.hasFocus, !canPop);
        expect(tester.testTextInput.isVisible, !canPop);
      },
    );
  }

  testWidgets(
    'closing a form leaves the destination focus under its own control',
    (tester) async {
      final sourceFocus = FocusNode();
      final destinationFocus = FocusNode();
      addTearDown(sourceFocus.dispose);
      addTearDown(destinationFocus.dispose);
      final navigator = GlobalKey<NavigatorState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: navigator,
          home: Scaffold(body: TextField(focusNode: destinationFocus)),
        ),
      );
      navigator.currentState!.push<void>(
        MaterialPageRoute<void>(
          builder: (_) => FormKeyboardScope(
            child: Scaffold(body: TextField(focusNode: sourceFocus)),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(TextField).last);
      await tester.pump();
      navigator.currentState!.pop();
      destinationFocus.requestFocus();
      await tester.pumpAndSettle();
      expect(sourceFocus.hasFocus, isFalse);
      expect(destinationFocus.hasFocus, isTrue);
    },
  );

  testWidgets('dialog Done stays visible above the keyboard with large text', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 700);
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      harness(
        MediaQuery(
          data: const MediaQueryData(
            size: Size(390, 700),
            viewInsets: EdgeInsets.only(bottom: 300),
            textScaler: TextScaler.linear(2),
          ),
          child: CraftskyDialog(
            title: 'Edit link',
            body: Column(
              children: List.generate(
                12,
                (i) => Text('Long helper description $i'),
              ),
            ),
            actions: const [],
            primaryAction: const CraftskyDialogPrimaryAction(
              label: 'Save',
              onPressed: null,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final done = find.byKey(const Key('dialog-keyboard-done'));
    expect(done, findsOneWidget);
    expect(tester.getBottomRight(done).dy, lessThanOrEqualTo(400));
    expect(tester.getTopLeft(done).dy, greaterThanOrEqualTo(0));
    await tester.tap(done);
    expect(tester.takeException(), isNull);
  });
}
