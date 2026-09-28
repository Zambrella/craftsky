import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:skeletonizer/skeletonizer.dart';

void main() {
  Widget app({
    required Widget home,
    bool disableAnimations = false,
  }) {
    return MediaQuery(
      data: MediaQueryData(disableAnimations: disableAnimations),
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: home,
      ),
    );
  }

  testWidgets('box skeleton exposes one loading label and hides bones', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      app(
        home: const Scaffold(
          body: CraftskySkeletonList(
            itemCount: 2,
            itemBuilder: _accountRow,
          ),
        ),
      ),
    );

    expect(find.byType(AccountRowSkeleton), findsNWidgets(2));
    expect(find.bySemanticsLabel('Loading'), findsOneWidget);
    semantics.dispose();
  });

  testWidgets('sliver skeleton renders the requested rows', (tester) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      app(
        home: const Scaffold(
          body: CustomScrollView(
            slivers: [
              CraftskySkeletonSliverList(
                itemCount: 3,
                itemBuilder: _postRow,
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.byType(PostCardSkeleton), findsWidgets);
    expect(find.byType(SliverSkeletonizer), findsOneWidget);
    final list = tester.widget<SliverList>(find.byType(SliverList));
    expect(list.delegate.estimatedChildCount, 3);
    expect(find.bySemanticsLabel('Loading'), findsOneWidget);
    semantics.dispose();
  });

  testWidgets('reduced motion uses a static effect', (tester) async {
    late PaintingEffect effect;
    await tester.pumpWidget(
      app(
        disableAnimations: true,
        home: Builder(
          builder: (context) {
            effect = craftskySkeletonEffect(context);
            return const SizedBox.shrink();
          },
        ),
      ),
    );

    expect(effect, isA<SolidColorEffect>());
    expect(effect.duration, Duration.zero);
  });

  testWidgets('skeleton blocks pointer input', (tester) async {
    var tapped = false;
    await tester.pumpWidget(
      app(
        home: Scaffold(
          body: CraftskySkeleton(
            child: GestureDetector(
              onTap: () => tapped = true,
              child: const SizedBox(width: 100, height: 100),
            ),
          ),
        ),
      ),
    );

    await tester.tapAt(tester.getCenter(find.byType(GestureDetector)));
    expect(tapped, isFalse);
  });

  testWidgets('post skeleton does not render a divider', (tester) async {
    await tester.pumpWidget(
      app(home: const CraftskySkeleton(child: PostCardSkeleton())),
    );

    expect(find.byType(Divider), findsNothing);
  });

  testWidgets('shimmer follows inherited RTL direction', (tester) async {
    await tester.pumpWidget(
      app(
        home: const Directionality(
          textDirection: TextDirection.rtl,
          child: Scaffold(
            body: CraftskySkeleton(child: AccountRowSkeleton()),
          ),
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    final skeleton = tester.widget<Skeletonizer>(
      find.byWidgetPredicate((widget) => widget is Skeletonizer),
    );
    expect(skeleton.effect, isA<ShimmerEffect>());
  });

  testWidgets('dark theme resolves a theme-specific shimmer', (tester) async {
    late ShimmerEffect lightEffect;
    late ShimmerEffect darkEffect;

    await tester.pumpWidget(
      app(
        home: Builder(
          builder: (context) {
            lightEffect = craftskySkeletonEffect(context) as ShimmerEffect;
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    await tester.pumpWidget(
      app(
        home: Theme(
          data: AppTheme.darkThemeData,
          child: Builder(
            builder: (context) {
              darkEffect = craftskySkeletonEffect(context) as ShimmerEffect;
              return const SizedBox.shrink();
            },
          ),
        ),
      ),
    );

    expect(darkEffect.colors, isNot(lightEffect.colors));
  });
}

Widget _accountRow(BuildContext context, int index) {
  return const AccountRowSkeleton();
}

Widget _postRow(BuildContext context, int index) {
  return PostCardSkeleton(showMedia: index == 0);
}
