import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_card.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pump(
  WidgetTester tester,
  Widget child, {
  TextScaler? textScaler,
}) {
  return tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, routeChild) {
          if (textScaler == null) return routeChild!;
          final mediaQuery = MediaQuery.of(context);
          return MediaQuery(
            data: mediaQuery.copyWith(textScaler: textScaler),
            child: routeChild!,
          );
        },
        home: Scaffold(
          body: SingleChildScrollView(
            child: Padding(padding: const EdgeInsets.all(16), child: child),
          ),
        ),
      ),
    ),
  );
}

void main() {
  group('ProjectCard', () {
    testWidgets('summary keeps feed project details compact', (tester) async {
      await _pump(
        tester,
        const ProjectCard(
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.sewingCraftToken,
              status: ProjectOptionCatalogs.finishedStatusToken,
              title: 'Indigo jacket',
              duration: '3 weekends',
              materials: ['linen'],
            ),
            details: SewingProjectDetails(
              projectType: '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
              fitNotes: 'Shortened sleeves',
            ),
          ),
        ),
      );

      expect(find.text('Indigo jacket'), findsOneWidget);
      expect(find.text('Finished'), findsOneWidget);
      expect(find.text('Sewing'), findsOneWidget);
      expect(find.text('DURATION'), findsNothing);
      expect(find.text('3 weekends'), findsNothing);
      expect(find.text('MATERIALS'), findsNothing);
      expect(find.text('Shortened sleeves'), findsNothing);
    });

    testWidgets('detail renders common, pattern, and sewing fields', (
      tester,
    ) async {
      await _pump(
        tester,
        const ProjectCard(
          variant: ProjectCardVariant.detail,
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.sewingCraftToken,
              status: ProjectOptionCatalogs.finishedStatusToken,
              title: 'Wiksten Haori in indigo linen',
              duration: '3 weekends',
              pattern: ProjectPattern(
                name: 'Wiksten Haori',
                designer: 'Jenny Gordy',
                difficulty:
                    '${ProjectOptionCatalogs.feedDefsPrefix}#intermediate',
                url: 'https://example.com/pattern',
              ),
              materials: ['linen', 'cotton thread'],
              colors: ['blue', 'natural'],
              designTags: [
                '${ProjectOptionCatalogs.projectDefsPrefix}#minimalist',
              ],
              tags: ['jacket', 'stashbuster'],
            ),
            details: SewingProjectDetails(
              projectType: '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
              projectSubtype:
                  '${ProjectOptionCatalogs.sewingDefsPrefix}#coatJacket',
              sizeMade: 'Medium',
              fitNotes: 'Shortened sleeves',
            ),
          ),
        ),
      );

      expect(find.byKey(const ValueKey('project-detail-card')), findsOneWidget);
      expect(
        tester.widget<Column>(
          find.byKey(const ValueKey('project-detail-card')),
        ),
        isA<Column>(),
      );
      expect(find.text('Wiksten Haori in indigo linen'), findsOneWidget);
      expect(find.text('DURATION'), findsOneWidget);
      expect(find.text('3 weekends'), findsOneWidget);
      expect(find.text('PATTERN'), findsOneWidget);
      expect(_richText('Wiksten Haori'), findsOneWidget);
      expect(_richText('Jenny Gordy'), findsOneWidget);
      expect(find.text('DIFFICULTY'), findsOneWidget);
      expect(find.text('Intermediate'), findsOneWidget);
      expect(find.text('LINK'), findsOneWidget);
      expect(find.text('example.com/pattern'), findsOneWidget);
      expect(find.text('PROJECT TYPE'), findsOneWidget);
      expect(find.text('Garment > Coat or jacket'), findsOneWidget);
      expect(find.text('PROJECT SUBTYPE'), findsNothing);
      expect(find.text('SIZE MADE'), findsOneWidget);
      expect(find.text('Medium'), findsOneWidget);
      expect(find.text('FIT NOTES'), findsOneWidget);
      expect(find.text('Shortened sleeves'), findsOneWidget);
      expect(find.text('MATERIALS'), findsOneWidget);
      expect(find.text('linen'), findsOneWidget);
      expect(find.text('cotton thread'), findsOneWidget);
      expect(find.text('COLOURS'), findsOneWidget);
      expect(find.text('Blue'), findsOneWidget);
      expect(find.text('Natural'), findsOneWidget);
      expect(find.text('DESIGN TAGS'), findsOneWidget);
      expect(find.text('Minimalist'), findsOneWidget);
      expect(find.text('TAGS'), findsOneWidget);
      expect(find.text('jacket'), findsOneWidget);
      expect(find.text('stashbuster'), findsOneWidget);
    });

    testWidgets('detail formats knitting gauge and tool fields', (
      tester,
    ) async {
      await _pump(
        tester,
        const ProjectCard(
          variant: ProjectCardVariant.detail,
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.knittingCraftToken,
            ),
            details: KnittingProjectDetails(
              yarnWeight:
                  '${ProjectOptionCatalogs.projectDefsPrefix}#fingering',
              needleSizeMm: '3.5mm',
              gauge: ProjectGauge(
                stitches: 24,
                rows: 32,
                measurement: 10,
                unit: 'cm',
              ),
              finishedSize: '40 in bust',
            ),
          ),
        ),
      );

      expect(find.text('YARN WEIGHT'), findsOneWidget);
      expect(find.text('Fingering'), findsOneWidget);
      expect(find.text('NEEDLE SIZE'), findsOneWidget);
      expect(find.text('3.5mm'), findsOneWidget);
      expect(find.text('GAUGE'), findsOneWidget);
      expect(find.text('24 sts / 32 rows per 10 cm'), findsOneWidget);
      expect(find.text('FINISHED SIZE'), findsOneWidget);
      expect(find.text('40 in bust'), findsOneWidget);
    });

    testWidgets('detail formats crochet gauge without rows', (tester) async {
      await _pump(
        tester,
        const ProjectCard(
          variant: ProjectCardVariant.detail,
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.crochetCraftToken,
            ),
            details: CrochetProjectDetails(
              hookSizeMm: '4.0mm',
              gauge: ProjectGauge(stitches: 18, measurement: 4, unit: 'in'),
              finishedSize: 'Baby blanket',
            ),
          ),
        ),
      );

      expect(find.text('HOOK SIZE'), findsOneWidget);
      expect(find.text('4.0mm'), findsOneWidget);
      expect(find.text('18 sts per 4 in'), findsOneWidget);
      expect(find.text('Baby blanket'), findsOneWidget);
    });

    testWidgets('detail formats quilting fields', (tester) async {
      await _pump(
        tester,
        const ProjectCard(
          variant: ProjectCardVariant.detail,
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.quiltingCraftToken,
            ),
            details: QuiltingProjectDetails(
              projectType: '${ProjectOptionCatalogs.projectDefsPrefix}#quilt',
              projectSubtype:
                  '${ProjectOptionCatalogs.quiltingDefsPrefix}#throwQuilt',
              size: '60 x 72 in',
              piecingTechnique:
                  '${ProjectOptionCatalogs.quiltingDefsPrefix}#improv',
              quiltingMethod:
                  '${ProjectOptionCatalogs.quiltingDefsPrefix}#handQuilted',
            ),
          ),
        ),
      );

      expect(find.text('Quilt > Throw quilt'), findsOneWidget);
      expect(find.text('60 x 72 in'), findsOneWidget);
      expect(find.text('PIECING TECHNIQUE'), findsOneWidget);
      expect(find.text('Improv'), findsOneWidget);
      expect(find.text('QUILTING METHOD'), findsOneWidget);
      expect(find.text('Hand quilted'), findsOneWidget);
    });

    testWidgets('metadata label column scales with text scaler', (
      tester,
    ) async {
      await _pump(
        tester,
        const ProjectCard(
          variant: ProjectCardVariant.detail,
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.sewingCraftToken,
              duration: 'Scaled value',
            ),
          ),
        ),
        textScaler: const TextScaler.linear(2),
      );

      expect(
        tester.getTopLeft(find.text('Scaled value')).dx,
        greaterThan(160),
      );
    });

    testWidgets('link display strips noise and confirms before launching', (
      tester,
    ) async {
      final launched = <Uri>[];
      await _pump(
        tester,
        ProjectCard(
          variant: ProjectCardVariant.detail,
          launchUrl: (uri) async {
            launched.add(uri);
            return true;
          },
          project: const Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.sewingCraftToken,
              pattern: ProjectPattern(
                url: 'https://example.com/patterns/top?utm_source=feed#details',
              ),
            ),
          ),
        ),
      );

      final link = find.text('example.com/patterns/top');
      expect(link, findsOneWidget);
      expect(find.textContaining('utm_source'), findsNothing);

      final text = tester.widget<Text>(link);
      final primary = Theme.of(tester.element(link)).colorScheme.primary;
      expect(text.style?.color, primary);

      await tester.tap(link);
      await tester.pumpAndSettle();

      expect(find.text('Open link?'), findsOneWidget);
      expect(find.text('This will open outside Craftsky.'), findsOneWidget);
      expect(
        find.text('https://example.com/patterns/top?utm_source=feed#details'),
        findsOneWidget,
      );
      expect(launched, isEmpty);

      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();

      expect(launched, [
        Uri.parse('https://example.com/patterns/top?utm_source=feed#details'),
      ]);
      expect(find.text('Open link?'), findsNothing);
    });
  });
}

Finder _richText(String text) {
  return find.byWidgetPredicate(
    (widget) => widget is Text && widget.textSpan?.toPlainText() == text,
  );
}
