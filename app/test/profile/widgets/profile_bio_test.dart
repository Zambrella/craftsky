import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_action_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  group('ProfileBio', () {
    for (final handle in ['stashhubapp.com', 'craftsky.social']) {
      testWidgets('bio mention @$handle resolves and opens its profile', (
        tester,
      ) async {
        final requested = <String>[];
        final repo = FakeProfileRepository(
          onFetch: (identifier) async {
            requested.add(identifier);
            return Profile(
              did: 'did:plc:mentioned',
              handle: handle,
              crafts: const [],
            );
          },
        );
        final router = GoRouter(
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) =>
                  Scaffold(body: ProfileBio(description: '@$handle')),
            ),
            GoRoute(
              path: '/profiles/@:handle',
              redirect: (context, state) => ProfileAliasRoute(
                handle: Handle.parse(state.pathParameters['handle']!),
              ).redirect(context, state),
            ),
            GoRoute(
              path: '/profiles/:did',
              builder: (_, state) => Scaffold(
                body: Text('profile:${state.pathParameters['did']}'),
              ),
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [profileRepositoryProvider.overrideWithValue(repo)],
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData,
              routerConfig: router,
            ),
          ),
        );
        expect(requested, isEmpty);
        await tester.tap(find.text('@$handle'));
        await tester.pumpAndSettle();
        expect(requested, [handle]);
        expect(find.text('profile:did:plc:mentioned'), findsOneWidget);
        expect(tester.takeException(), isNull);
      });
    }

    testWidgets(
      'AT-005 detects and styles plain bio tokens with theme primary color',
      (
        tester,
      ) async {
        await tester.pumpWidget(
          ProviderScope(
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const Scaffold(
                body: ProfileBio(
                  description:
                      'Visit craftsky.social @alice.craftsky.social #Lace',
                ),
              ),
            ),
          ),
        );

        final body = tester.widget<Text>(
          find.byWidgetPredicate(
            (widget) =>
                widget is Text &&
                widget.textSpan?.toPlainText() ==
                    'Visit craftsky.social @alice.craftsky.social #Lace',
          ),
        );
        final spans = _leafTextSpans(body.textSpan! as TextSpan);

        expect(spans.map((span) => span.text), [
          'Visit ',
          'craftsky.social',
          ' ',
          '@alice.craftsky.social',
          ' ',
          '#Lace',
        ]);
        expect(spans[1].style?.color, BrandColors.cobalt);
        expect(spans[3].style?.color, BrandColors.cobalt);
        expect(spans[5].style?.color, BrandColors.cobalt);
        expect(spans[1].style?.decoration, TextDecoration.underline);
        expect(spans[3].style?.decoration, TextDecoration.none);
        expect(spans[5].style?.decoration, TextDecoration.none);
      },
    );

    testWidgets('Ink hyperlinks remain identifiable by their underline', (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: ProfileCustomisationTheme(
                customisation: ProfileCustomisation(colour: 'ink'),
                child: ProfileBio(
                  description: 'Read craftsky.social',
                ),
              ),
            ),
          ),
        ),
      );

      final body = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text &&
              widget.textSpan?.toPlainText() == 'Read craftsky.social',
        ),
      );
      final spans = _leafTextSpans(body.textSpan! as TextSpan);

      expect(spans[0].style?.color, BrandColors.ink);
      expect(spans[1].style?.color, BrandColors.ink);
      expect(spans[0].style?.decoration, TextDecoration.none);
      expect(spans[1].style?.decoration, TextDecoration.underline);
    });

    testWidgets('UT-009 leaves unsupported schemes and URL fragments safe', (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: ProfileBio(
                description:
                    'mailto:x@y.example https://craftsky.social/#lace #lace',
              ),
            ),
          ),
        ),
      );

      final body = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text &&
              widget.textSpan?.toPlainText() ==
                  'mailto:x@y.example craftsky.social #lace',
        ),
      );
      final spans = _leafTextSpans(body.textSpan! as TextSpan);
      expect(spans.map((span) => span.text), [
        'mailto:x@y.example ',
        'craftsky.social',
        ' ',
        '#lace',
      ]);
      expect(tester.takeException(), isNull);
    });

    testWidgets('confirms before opening bio links', (tester) async {
      Uri? launched;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            facetUrlLauncherProvider.overrideWithValue((uri) async {
              launched = uri;
              return true;
            }),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: ProfileBio(
                description: 'https://craftsky.social/#lace',
              ),
            ),
          ),
        ),
      );

      await tester.tap(find.text('craftsky.social'));
      await tester.pumpAndSettle();

      expect(find.text('Open link?'), findsOneWidget);
      expect(find.text('https://craftsky.social/#lace'), findsOneWidget);
      expect(launched, isNull);

      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();

      expect(launched, Uri.parse('https://craftsky.social/#lace'));
    });
  });
}

List<TextSpan> _leafTextSpans(TextSpan root) {
  final leaves = <TextSpan>[];

  void visit(TextSpan span) {
    final children = span.children;
    if (children == null || children.isEmpty) {
      leaves.add(span);
      return;
    }
    for (final child in children) {
      if (child is TextSpan) visit(child);
    }
  }

  visit(root);
  return leaves;
}
