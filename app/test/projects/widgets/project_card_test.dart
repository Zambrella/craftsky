import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_card.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

Future<void> _pump(
  WidgetTester tester,
  Widget child, {
  TextScaler? textScaler,
}) {
  return tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
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
  testWidgets('mixed-craft project shows primary and additional craft chips', (
    tester,
  ) async {
    await _pump(
      tester,
      const ProjectCard(
        project: Project(
          common: ProjectCommon(
            craftType: ProjectOptionCatalogs.sewingCraftToken,
            additionalCrafts: [
              ProjectCraft(
                craftType: ProjectOptionCatalogs.embroideryCraftToken,
              ),
            ],
          ),
        ),
      ),
    );
    expect(find.text('Sewing'), findsOneWidget);
    expect(find.text('Embroidery'), findsOneWidget);
  });

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
              materials: [ProjectMaterial(text: 'linen')],
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
      expect(find.byType(CraftIcon), findsOneWidget);
      expect(find.text('DURATION'), findsNothing);
      expect(find.text('3 weekends'), findsNothing);
      expect(find.text('MATERIALS'), findsNothing);
      expect(find.text('Shortened sleeves'), findsNothing);
    });

    testWidgets('detail keeps active common, pattern, and sewing fields', (
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
                selfDrafted: true,
              ),
              materials: [
                ProjectMaterial(text: 'linen'),
                ProjectMaterial(text: 'cotton thread'),
              ],
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
      expect(find.text('DURATION'), findsNothing);
      expect(find.text('3 weekends'), findsNothing);
      expect(find.text('PATTERN'), findsOneWidget);
      expect(_richText('Wiksten Haori'), findsOneWidget);
      expect(_richText('Jenny Gordy'), findsOneWidget);
      expect(find.text('DIFFICULTY'), findsOneWidget);
      expect(find.text('Intermediate'), findsOneWidget);
      expect(find.text('Self-drafted'), findsOneWidget);
      expect(find.text('LINK'), findsOneWidget);
      expect(find.text('example.com/pattern'), findsOneWidget);
      expect(find.text('PROJECT TYPE'), findsOneWidget);
      expect(find.text('Garment > Coat or jacket'), findsOneWidget);
      expect(find.text('PROJECT SUBTYPE'), findsNothing);
      expect(find.text('SIZE MADE'), findsNothing);
      expect(find.text('Medium'), findsNothing);
      expect(find.text('FIT NOTES'), findsNothing);
      expect(find.text('Shortened sleeves'), findsNothing);
      expect(find.text('MATERIALS'), findsNothing);
      expect(find.text('linen'), findsNothing);
      expect(find.text('cotton thread'), findsNothing);
      expect(find.text('COLOURS'), findsNothing);
      expect(find.text('Blue'), findsNothing);
      expect(find.text('Natural'), findsNothing);
      expect(find.text('DESIGN'), findsNothing);
      expect(find.text('Minimalist'), findsNothing);
      expect(find.text('TAGS'), findsOneWidget);
      expect(find.text('jacket'), findsOneWidget);
      expect(find.text('stashbuster'), findsOneWidget);
    });

    testWidgets(
      'detail shows knitting methods and hides retired measurements',
      (
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
                extraFields: {
                  'methods': [ProjectOptionCatalogs.knittingMachineMethodToken],
                },
              ),
            ),
          ),
        );

        expect(find.text('KNITTING METHODS'), findsOneWidget);
        expect(find.text('Machine'), findsOneWidget);
        expect(find.text('YARN WEIGHT'), findsNothing);
        expect(find.text('Fingering'), findsNothing);
        expect(find.text('NEEDLE SIZE'), findsNothing);
        expect(find.text('3.5mm'), findsNothing);
        expect(find.text('GAUGE'), findsNothing);
        expect(find.text('24 sts / 32 rows per 10 cm'), findsNothing);
        expect(find.text('FINISHED SIZE'), findsNothing);
        expect(find.text('40 in bust'), findsNothing);
      },
    );

    testWidgets(
      'detail shows the Tunisian method without retired fields',
      (tester) async {
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
                extraFields: {
                  'techniques': [
                    {'technique': 'tunisianCrochet'},
                    {'technique': 'grannySquare'},
                  ],
                },
              ),
            ),
          ),
        );

        expect(find.text('CROCHET METHODS'), findsOneWidget);
        expect(find.text('Tunisian'), findsOneWidget);
        expect(find.text('TECHNIQUES'), findsNothing);
        expect(find.text('Granny square'), findsNothing);
        expect(find.text('HOOK SIZE'), findsNothing);
        expect(find.text('4.0mm'), findsNothing);
        expect(find.text('18 sts per 4 in'), findsNothing);
        expect(find.text('Baby blanket'), findsNothing);
      },
    );

    testWidgets('detail shows the quilt type without retired quilting fields', (
      tester,
    ) async {
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

      expect(find.text('Quilt > Throw or lap quilt'), findsOneWidget);
      expect(find.text('60 x 72 in'), findsNothing);
      expect(find.text('PIECING TECHNIQUE'), findsNothing);
      expect(find.text('Improv'), findsNothing);
      expect(find.text('QUILTING METHOD'), findsNothing);
      expect(find.text('Hand quilted'), findsNothing);
    });

    testWidgets(
      'detail uses active felting choices with a safe legacy fallback',
      (
        tester,
      ) async {
        const oldDetails = {
          'method': 'needle',
          'form': 'retired form',
          'core': 'retired core',
        };
        for (final (raw, needleVisible) in [
          (oldDetails, true),
          ({...oldDetails, 'methods': <String>[]}, false),
        ]) {
          await _pump(
            tester,
            ProjectCard(
              variant: ProjectCardVariant.detail,
              project: Project(
                common: const ProjectCommon(
                  craftType: ProjectOptionCatalogs.feltingCraftToken,
                ),
                details: UnknownProjectDetails(
                  type: 'social.craftsky.project.felting#details',
                  raw: raw,
                ),
              ),
            ),
          );
          expect(
            find.text('FELTING METHODS'),
            needleVisible ? findsOneWidget : findsNothing,
          );
          expect(
            find.text('Needle'),
            needleVisible ? findsOneWidget : findsNothing,
          );
          expect(find.text('FELTING METHOD'), findsNothing);
          expect(find.text('retired form'), findsNothing);
          expect(find.text('retired core'), findsNothing);
        }
      },
    );

    testWidgets('detail shows current embroidery choices without old groups', (
      tester,
    ) async {
      await _pump(
        tester,
        const ProjectCard(
          variant: ProjectCardVariant.detail,
          project: Project(
            common: ProjectCommon(
              craftType: ProjectOptionCatalogs.embroideryCraftToken,
            ),
            details: UnknownProjectDetails(
              type: 'social.craftsky.project.embroidery#details',
              raw: {
                'method': 'social.craftsky.project.embroidery.defs#machine',
                'style': 'social.craftsky.project.embroidery.defs#crossStitch',
                'methods': [
                  {
                    'method': 'social.craftsky.project.embroidery.defs#hand',
                    'styles': [
                      {
                        'style':
                            'social.craftsky.project.embroidery.defs#sashiko',
                      },
                    ],
                  },
                ],
              },
            ),
          ),
        ),
      );

      expect(find.text('EMBROIDERY METHOD'), findsOneWidget);
      expect(find.text('Machine'), findsOneWidget);
      expect(find.text('EMBROIDERY STYLE'), findsOneWidget);
      expect(find.text('Cross-stitch'), findsOneWidget);
      expect(find.text('EMBROIDERY METHODS'), findsNothing);
      expect(find.text('Hand (Sashiko)'), findsNothing);
    });

    testWidgets('detail does not infer a new type from retired dyeing fields', (
      tester,
    ) async {
      const oldDetails = {
        'materialForm': 'social.craftsky.project.dyeing#roving',
        'recipeNotes': 'Former recipe notes',
      };
      for (final (raw, typeVisible) in [
        (oldDetails, false),
        (
          {
            ...oldDetails,
            'projectType': 'social.craftsky.project.dyeing#fabric',
          },
          true,
        ),
      ]) {
        await _pump(
          tester,
          ProjectCard(
            variant: ProjectCardVariant.detail,
            project: Project(
              common: const ProjectCommon(
                craftType: ProjectOptionCatalogs.dyeingCraftToken,
              ),
              details: UnknownProjectDetails(
                type: 'social.craftsky.project.dyeing#details',
                raw: raw,
              ),
            ),
          ),
        );
        expect(
          find.text('DYED ITEM'),
          typeVisible ? findsOneWidget : findsNothing,
        );
        expect(
          find.text('Fabric'),
          typeVisible ? findsOneWidget : findsNothing,
        );
        expect(find.text('Material form'), findsNothing);
        expect(find.text('Roving'), findsNothing);
        expect(find.text('Former recipe notes'), findsNothing);
      }
    });

    testWidgets('detail allows active fields in untyped historical details', (
      tester,
    ) async {
      for (final (craft, detailsType, itemVisible) in [
        (ProjectOptionCatalogs.sewingCraftToken, null, true),
        (
          ProjectOptionCatalogs.sewingCraftToken,
          knittingProjectDetailsType,
          false,
        ),
        (ProjectOptionCatalogs.spinningCraftToken, null, false),
      ]) {
        await _pump(
          tester,
          ProjectCard(
            variant: ProjectCardVariant.detail,
            project: Project(
              common: ProjectCommon(craftType: craft),
              details: UnknownProjectDetails(
                type: detailsType,
                raw: const {
                  'projectType':
                      '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
                  'projectSubtype':
                      '${ProjectOptionCatalogs.sewingDefsPrefix}#coatJacket',
                  'fitNotes': 'Old fit details',
                },
              ),
            ),
          ),
        );
        expect(
          find.text('Garment > Coat or jacket'),
          itemVisible ? findsOneWidget : findsNothing,
        );
        expect(find.text('Old fit details'), findsNothing);
      }
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
              pattern: ProjectPattern(difficulty: 'Scaled value'),
            ),
          ),
        ),
        textScaler: const TextScaler.linear(2),
      );

      expect(
        tester.getTopLeft(find.text('Scaled Value')).dx,
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
      expect(find.text('This will open outside CraftSky.'), findsOneWidget);
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
