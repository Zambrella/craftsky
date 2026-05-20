import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pump(WidgetTester tester, Widget child) {
  return tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        home: Scaffold(body: SizedBox(height: 500, child: child)),
      ),
    ),
  );
}

void main() {
  testWidgets('shows current alt text and updates when swiping pages', (
    tester,
  ) async {
    await _pump(
      tester,
      PostImageGallery(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl laid flat',
            thumb: 'https://cdn.example.com/thumb1.jpg',
            fullsize: 'https://cdn.example.com/full1.jpg',
          ),
          PostImage(
            cid: 'bafkimage2',
            mime: 'image/png',
            size: 11,
            alt: 'Close-up stitch detail',
            thumb: 'https://cdn.example.com/thumb2.jpg',
            fullsize: 'https://cdn.example.com/full2.jpg',
          ),
        ],
      ),
    );

    expect(find.text('Blue shawl laid flat'), findsOneWidget);
    expect(find.bySemanticsLabel('Blue shawl laid flat'), findsWidgets);
    expect(find.byType(InteractiveViewer), findsWidgets);

    final pageView = tester.widget<PageView>(
      find.byKey(const Key('post-image-gallery-page-view')),
    );
    pageView.onPageChanged?.call(1);
    await tester.pump();

    expect(find.text('Close-up stitch detail'), findsOneWidget);
    expect(find.bySemanticsLabel('Close-up stitch detail'), findsWidgets);
  });

  testWidgets('gallery images are zoom-enabled via InteractiveViewer', (
    tester,
  ) async {
    await _pump(
      tester,
      PostImageGallery(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl laid flat',
            thumb: 'https://cdn.example.com/thumb1.jpg',
            fullsize: 'https://cdn.example.com/full1.jpg',
          ),
        ],
      ),
    );

    final viewer = tester.widget<InteractiveViewer>(
      find.byType(InteractiveViewer).first,
    );
    expect(viewer.minScale, 1);
    expect(viewer.maxScale, greaterThan(1));
  });
}
