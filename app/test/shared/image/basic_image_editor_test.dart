import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/image/basic_image_editor_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:material_ui/material_ui.dart';
import 'package:pro_image_editor/pro_image_editor.dart';

void main() {
  testWidgets(
    'image editing AT-004: export failure closes editor and '
    'preserves original cause',
    (tester) async {
      final request = BasicImageEditRequest(
        bytes: Uint8List.fromList(
          img.encodePng(img.Image(width: 40, height: 30)),
        ),
        mimeType: 'image/png',
        mode: BasicImageEditMode.composer,
      );
      final cause = StateError('export failure');
      final originalStack = StackTrace.current;
      Object? failure;
      StackTrace? failureStack;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(disableAnimations: true),
            child: child!,
          ),
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () async {
                  try {
                    await showBasicImageEditor(context, request);
                  } on Object catch (e, s) {
                    failure = e;
                    failureStack = s;
                  }
                },
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      tester
          .widget<BasicImageEditorPage>(find.byType(BasicImageEditorPage))
          .onExportError!(cause, originalStack);
      await tester.pumpAndSettle();
      expect(failure, same(cause));
      expect(failureStack, same(originalStack));
      expect(find.text('Open'), findsOneWidget);
    },
  );

  testWidgets(
    'image editing AT-010: reduced motion, labeled controls '
    'and keyboard Cancel',
    (tester) async {
      final semantics = tester.ensureSemantics();
      final request = BasicImageEditRequest(
        bytes: Uint8List.fromList(
          img.encodePng(img.Image(width: 40, height: 30)),
        ),
        mimeType: 'image/png',
        mode: BasicImageEditMode.avatar,
      );
      var returned = false;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              disableAnimations: true,
              textScaler: const TextScaler.linear(2),
            ),
            child: child!,
          ),
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () async {
                  expect(await showBasicImageEditor(context, request), isNull);
                  returned = true;
                },
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      for (var i = 0; i < 10; i++) {
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        });
        await tester.pump(const Duration(milliseconds: 50));
      }
      final editor = tester.widget<CropRotateEditor>(
        find.byType(CropRotateEditor),
      );
      expect(
        editor.initConfigs.configs.cropRotateEditor.animationDuration,
        Duration.zero,
      );
      final rotate = find.byKey(
        const ValueKey('crop-rotate-editor-rotate-btn'),
      );
      expect(tester.getSize(rotate).height, greaterThanOrEqualTo(48));
      expect(
        tester.getSemantics(rotate),
        matchesSemantics(
          label: 'Rotate 90°',
          isButton: true,
          hasEnabledState: true,
          isEnabled: true,
          hasTapAction: true,
          hasFocusAction: true,
          isFocusable: true,
        ),
      );
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pumpAndSettle();
      expect(returned, isTrue);
      semantics.dispose();
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );

  testWidgets(
    'image editing AT-009: duplicate completion returns the '
    'first accepted output once',
    (tester) async {
      BasicImageEditOutput? result;
      final request = BasicImageEditRequest(
        bytes: Uint8List.fromList(
          img.encodePng(img.Image(width: 40, height: 30)),
        ),
        mimeType: 'image/png',
        mode: BasicImageEditMode.composer,
      );
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(disableAnimations: true),
            child: child!,
          ),
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () async {
                  result = await showBasicImageEditor(context, request);
                },
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      for (var i = 0; i < 10; i++) {
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        });
        await tester.pump(const Duration(milliseconds: 50));
      }
      final callbacks = tester
          .widget<CropRotateEditor>(find.byType(CropRotateEditor))
          .initConfigs
          .callbacks;
      final first = Uint8List.fromList([1, 2, 3]);
      await callbacks.onImageEditingComplete!(first);
      await callbacks.onImageEditingComplete!(Uint8List.fromList([4, 5]));
      callbacks.onCloseEditor!(EditorMode.cropRotate);
      callbacks.onCloseEditor!(EditorMode.cropRotate);
      await tester.pumpAndSettle();
      expect(result!.bytes, first);
      expect(find.text('Open'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );

  for (final mode in BasicImageEditMode.values) {
    testWidgets(
      'image editing AT-002/005/010 ($mode): restricted '
      'composer tools and cancellation',
      (tester) async {
        await tester.pumpWidget(
          MaterialApp(
            theme: AppTheme.lightThemeData,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context).copyWith(disableAnimations: true),
              child: child!,
            ),
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: BasicImageEditorPage(
              request: BasicImageEditRequest(
                bytes: Uint8List.fromList(
                  img.encodePng(img.Image(width: 40, height: 30)),
                ),
                mimeType: 'image/png',
                mode: mode,
              ),
            ),
          ),
        );
        for (var i = 0; i < 10; i++) {
          await tester.runAsync(() async {
            await Future<void>.delayed(const Duration(milliseconds: 20));
          });
          await tester.pump(const Duration(milliseconds: 50));
        }
        final editor = tester.widget<CropRotateEditor>(
          find.byType(CropRotateEditor),
        );
        expect(editor.initConfigs.convertToUint8List, isTrue);
        // The surrounding app is light; both standalone controls and package
        // overlays must still receive CraftSky's dark photo-editing palette.
        expect(editor.initConfigs.theme, AppTheme.darkThemeData);
        expect(editor.initConfigs.configs.theme, AppTheme.darkThemeData);
        final toolbarContext = tester.element(find.byType(AppBar));
        expect(Theme.of(toolbarContext).brightness, Brightness.dark);
        final config = editor.initConfigs.configs.cropRotateEditor;
        expect(
          config.style.background,
          AppTheme.darkThemeData.scaffoldBackgroundColor,
        );
        expect(config.style.bottomBarBackground, config.style.background);
        expect(
          config.style.bottomBarColor,
          AppTheme.darkThemeData.colorScheme.onSurface,
        );
        expect(
          config.style.cropCornerColor,
          AppTheme.darkThemeData.colorScheme.primary,
        );
        expect(config.tools, [CropRotateTool.rotate]);
        expect(config.enableKeyboardShortcuts, isFalse);
        expect(config.exportOvalMask, isFalse);
        expect(
          config.initAspectRatio,
          mode == BasicImageEditMode.avatar ? 1 : -1,
        );
        expect(
          config.initialCropMode,
          mode == BasicImageEditMode.avatar
              ? CropMode.oval
              : CropMode.rectangular,
        );
        expect(find.text('Rotate 90°'), findsOneWidget);
        expect(find.byTooltip('Cancel'), findsOneWidget);
        expect(find.byTooltip('Done'), findsOneWidget);
        expect(find.text('Filter'), findsNothing);
        await tester.pumpWidget(const SizedBox());
        await tester.pump(const Duration(seconds: 1));
      },
    );
  }
}
