import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('Android fullscreen modal slides up and back down', (
    tester,
  ) async {
    final navigatorKey = GlobalKey<NavigatorState>();
    await tester.pumpWidget(
      MaterialApp(
        navigatorKey: navigatorKey,
        theme: ThemeData(platform: TargetPlatform.android),
        home: const Scaffold(body: Text('Underneath')),
      ),
    );

    final route = FullscreenModalRoute<void>(
      builder: (_) => const Scaffold(body: Text('Modal')),
    );
    navigatorKey.currentState!.push(route).ignore();
    await tester.pump();

    expect(route.fullscreenDialog, isTrue);
    final slideFinder = find.ancestor(
      of: find.text('Modal', skipOffstage: false),
      matching: find.byType(SlideTransition, skipOffstage: false),
    );
    Offset slideOffset() =>
        tester.widget<SlideTransition>(slideFinder.first).position.value;
    expect(slideOffset(), const Offset(0, 1));

    await tester.pump(const Duration(milliseconds: 150));
    expect(slideOffset().dy, inExclusiveRange(0, 1));
    await tester.pumpAndSettle();
    expect(slideOffset(), Offset.zero);

    navigatorKey.currentState!.pop();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
    expect(slideOffset().dy, inExclusiveRange(0, 1));
    await tester.pumpAndSettle();
    expect(find.text('Modal'), findsNothing);
  });

  for (final presentation in [
    (size: const Size(1200, 800), keepsNavigation: true),
    (size: const Size(500, 800), keepsNavigation: false),
  ]) {
    final description = presentation.keepsNavigation
        ? 'stays beside the large rail'
        : 'covers compact navigation';
    testWidgets(
      'fullscreen modal $description',
      (tester) async {
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = presentation.size;

        await tester.pumpWidget(
          MaterialApp(
            builder: (context, child) => FormFactorWidget(child: child!),
            home: const _ModalNavigationHarness(),
          ),
        );

        await tester.tap(find.text('Open modal'));
        await tester.pumpAndSettle();

        expect(find.byKey(const Key('modal')), findsOneWidget);
        expect(
          find.byKey(const Key('navigation')),
          presentation.keepsNavigation ? findsOneWidget : findsNothing,
        );
      },
    );
  }
}

class _ModalNavigationHarness extends StatelessWidget {
  const _ModalNavigationHarness();

  @override
  Widget build(BuildContext context) {
    final isLarge = FormFactorWidget.of(context).isLarge;
    final content = Navigator(
      onGenerateRoute: (_) => MaterialPageRoute<void>(
        builder: (contentContext) => Scaffold(
          body: Center(
            child: ElevatedButton(
              onPressed: () {
                responsiveModalNavigator(contentContext)
                    .push<void>(
                      MaterialPageRoute<void>(
                        builder: (_) => const Scaffold(key: Key('modal')),
                      ),
                    )
                    .ignore();
              },
              child: const Text('Open modal'),
            ),
          ),
        ),
      ),
    );
    if (isLarge) {
      return Row(
        children: [
          const SizedBox(key: Key('navigation'), width: 240),
          Expanded(child: content),
        ],
      );
    }
    return Scaffold(
      body: content,
      bottomNavigationBar: const SizedBox(
        key: Key('navigation'),
        height: 64,
      ),
    );
  }
}
