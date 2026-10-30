import 'dart:async';
import 'dart:ui' as ui;

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:file/memory.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/image_cache_fakes.dart';

void main() {
  for (final fullscreen in [false, true]) {
    testWidgets(
      '${fullscreen ? 'fullscreen' : 'inline'} preloads neighbors '
      'only after the current image decodes',
      (tester) async {
        final directory = MemoryFileSystem().systemTempDirectory.createTempSync(
          'carousel-images',
        );
        final file = directory.childFile('image.png');
        await tester.runAsync(() async {
          final recorder = ui.PictureRecorder();
          ui.Canvas(recorder)
              .drawColor(const Color(0xFF336699), ui.BlendMode.src);
          final picture = recorder.endRecording();
          final image = await picture.toImage(8, 8);
          final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
          file.writeAsBytesSync(bytes!.buffer.asUint8List());
          image.dispose();
          picture.dispose();
        });
        final requests = <String, StreamController<FileResponse>>{};
        final cache = FakeBaseCacheManager()
          ..nextStream = (url) {
            expect(requests.containsKey(url), isFalse);
            final controller = StreamController<FileResponse>();
            requests[url] = controller;
            return controller.stream;
          };
        final images = [
          for (var index = 0; index < 5; index++)
            PostImage(
              cid: 'image$index',
              mime: 'image/png',
              size: 10,
              alt: 'Image $index',
              thumb: 'https://example.com/thumb$index.png',
              fullsize: 'https://example.com/full$index.png',
            ),
        ];
        final prefix = fullscreen ? 'full' : 'thumb';
        String url(int index) => 'https://example.com/$prefix$index.png';
        final initialIndex = fullscreen ? 2 : 0;

        Future<void> finishImage(int index) async {
          requests[url(index)]!.add(
            FileInfo(file, FileSource.Cache, DateTime(2100), url(index)),
          );
          for (var attempt = 0; attempt < 10; attempt++) {
            await tester.pump();
            await tester.runAsync(() async {
              await Future<void>.delayed(const Duration(milliseconds: 10));
            });
          }
          await tester.pump();
        }

        addTearDown(() async {
          await tester.pumpWidget(const SizedBox.shrink());
          for (final controller in requests.values) {
            unawaited(controller.close());
          }
          PaintingBinding.instance.imageCache
            ..clear()
            ..clearLiveImages();
          directory.deleteSync(recursive: true);
        });

        await tester.pumpWidget(
          ProviderScope(
            overrides: [feedImageCacheManagerProvider.overrideWithValue(cache)],
            child: MaterialApp(
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Scaffold(
                body: fullscreen
                    ? PostImageGallery(
                        images: images,
                        initialIndex: initialIndex,
                      )
                    : PostImageCarousel(images: images),
              ),
            ),
          ),
        );
        await tester.pump();
        expect(requests.keys, [url(initialIndex)]);

        await finishImage(initialIndex);
        expect(
          requests.keys.toSet(),
          {
            url(initialIndex),
            if (initialIndex > 0) url(initialIndex - 1),
            url(initialIndex + 1),
          },
        );

        // Completing a prefetched image must not cascade through the album.
        await finishImage(initialIndex + 1);
        expect(requests.containsKey(url(initialIndex + 2)), isFalse);

        await tester.drag(find.byType(PageView), const Offset(-900, 0));
        await tester.pumpAndSettle();
        expect(requests.containsKey(url(initialIndex + 2)), isTrue);
        expect(requests.keys.every((value) => value.contains(prefix)), isTrue);

        // Select an unfinished page without dragging other pages into view.
        // A drag can legitimately start loading a partially exposed page.
        final pendingIndex = fullscreen ? 1 : 2;
        final beyondPendingIndex = fullscreen ? 0 : 3;
        tester
            .widget<PageView>(find.byType(PageView))
            .controller!
            .jumpToPage(pendingIndex);
        await tester.pump();
        await tester.pump();
        expect(requests.containsKey(url(beyondPendingIndex)), isFalse);
        await finishImage(pendingIndex);
        expect(requests.containsKey(url(beyondPendingIndex)), isTrue);
        expect(tester.takeException(), isNull);
      },
    );
  }
}
