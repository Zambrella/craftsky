import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_sliver_app_bar.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:visibility_detector/visibility_detector.dart';

void main() {
  setUp(
    () => VisibilityDetectorController.instance.updateInterval = Duration.zero,
  );
  tearDown(
    () => VisibilityDetectorController.instance.updateInterval = const Duration(
      milliseconds: 500,
    ),
  );

  testWidgets(
    'bright profile backgrounds keep navigation legible in both themes',
    (tester) async {
      for (final mode in [ThemeMode.light, ThemeMode.dark]) {
        for (final background in ['animated-ribbons', 'animated-flow']) {
          await tester.pumpWidget(
            ProviderScope(
              child: MaterialApp(
                theme: AppTheme.lightThemeData,
                darkTheme: AppTheme.darkThemeData,
                themeMode: mode,
                localizationsDelegates: appLocalizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(context)
                      .copyWith(disableAnimations: true),
                  child: child!,
                ),
                home: Builder(
                  builder: (context) => Scaffold(
                    body: TextButton(
                      onPressed: () => Navigator.of(context).push(
                        MaterialPageRoute<void>(
                          builder: (_) => Scaffold(
                            body: CustomScrollView(
                              slivers: [
                                ProfileSliverAppBar(
                                  handle: 'alice.example',
                                  customisation: ProfileCustomisation(
                                    background: background,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                      child: const Text('Open profile'),
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.tap(find.text('Open profile'));
          await tester.pumpAndSettle();
          final back = tester.widget<BackButton>(find.byType(BackButton));
          final context = tester.element(find.byType(BackButton));
          final theme = Theme.of(context);
          final fill = back.style!.backgroundColor!.resolve({})!;
          if (background == 'animated-ribbons') {
            final ink = back.style!.foregroundColor!.resolve({})!;
            expect(fill, theme.extension<BrandSwatchTheme>()!.paper);
            final luminances = [fill.computeLuminance(), ink.computeLuminance()]
              ..sort();
            expect(
              (luminances.last + 0.05) / (luminances.first + 0.05),
              greaterThanOrEqualTo(4.5),
            );
          } else {
            expect(fill, Colors.transparent);
          }
          await tester.pageBack();
          await tester.pumpAndSettle();
        }
      }
      await tester.pumpWidget(const SizedBox());
    },
  );
}
