import 'dart:async';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/service_status/widgets/status_message_image.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart' show FakeStatusRepository;
import 'support/status_test_fixtures.dart';

class Images extends HttpOverrides {
  final result = Completer<Uint8List>();
  late final String url;
  late final HttpClient client;
  void close() {
    debugNetworkImageHttpClientProvider = null;
    client.close(force: true);
  }

  Future<void> start(WidgetTester tester) async {
    final server = (await tester.runAsync(
      () => HttpServer.bind(InternetAddress.loopbackIPv4, 0),
    ))!;
    client = createHttpClient(null);
    debugNetworkImageHttpClientProvider = () => client;
    url = 'http://127.0.0.1:${server.port}/image';
    server.listen((request) async {
      final bytes = await result.future;
      request.response.add(bytes);
      await request.response.close();
    });
    addTearDown(() async {
      if (!result.isCompleted) result.complete(Uint8List(0));
      client.close(force: true);
      await server.close(force: true);
      debugNetworkImageHttpClientProvider = null;
    });
  }
}

Future<Uint8List> imageBytes(int width, int height) async {
  final recorder = ui.PictureRecorder();
  ui.Canvas(recorder).drawColor(const Color(0xff993366), ui.BlendMode.src);
  final picture = recorder.endRecording();
  final image = await picture.toImage(width, height);
  final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
  image.dispose();
  picture.dispose();
  return bytes!.buffer.asUint8List();
}

void main() {
  testWidgets('trusted host image uses standard network loading', (
    tester,
  ) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: StatusMessageImage(url: 'https://trusted.example/image.png'),
        ),
      ),
    );
    final image = tester.widget<Image>(find.byType(Image));
    final network = image.image as NetworkImage;
    expect(network.url, 'https://trusted.example/image.png');
    expect(network.headers, isNull);
    expect(image.height, 200);
    expect(image.fit, BoxFit.contain);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  for (final (imageState, width, dark) in [
    ('absent', 320.0, false),
    ('pending', 320.0, true),
    ('valid', 1280.0, false),
    ('broken', 1280.0, true),
    ('extreme', 320.0, false),
  ]) {
    for (final scale in [1.0, 1.5]) {
      testWidgets(
        'optional $imageState width=$width dark=$dark '
        'scale=$scale remains usable',
        (tester) async {
          await tester.binding.setSurfaceSize(Size(width, 640));
          addTearDown(() => tester.binding.setSurfaceSize(null));
          final repository = FakeStatusRepository();
          final images = Images();
          await images.start(tester);
          final reporter = ImageReporter();
          if (imageState == 'valid' || imageState == 'extreme') {
            images.result.complete(
              await tester.runAsync(
                () => imageBytes(imageState == 'extreme' ? 2000 : 16, 16),
              ),
            );
          } else if (imageState == 'broken') {
            images.result.complete(Uint8List.fromList([0, 1, 2]));
          }
          final container = ProviderContainer(
            overrides: [
              ...statusRuntimeOverrides,
              serviceStatusReporterProvider.overrideWithValue(reporter),
              serviceStatusRepositoryProvider.overrideWithValue(repository),
              announcementDismissalStoreProvider.overrideWithValue(
                FakeDismissalStore(),
              ),
            ],
          );
          final controller = container.read(
            serviceStatusControllerProvider.notifier,
          );
          await tester.pumpWidget(
            UncontrolledProviderScope(
              container: container,
              child: MaterialApp(
                theme: dark ? AppTheme.darkThemeData : AppTheme.lightThemeData,
                localizationsDelegates: appLocalizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(context)
                      .copyWith(textScaler: TextScaler.linear(scale)),
                  child: ServiceStatusHost(child: child!),
                ),
                home: const Scaffold(body: Text('Covered content')),
              ),
            ),
          );
          await tester.pump();
          final refresh = controller.refresh();
          repository.requests.last.complete(
            ServiceStatusDocument(
              revision: 'D',
              message: StatusMessage(
                revision: 'A',
                minVersion: '1.8.0',
                title: 'Notice ${'x' * 153}',
                description: 'Public ${'long text ' * 239}',
                imageUrl: imageState == 'absent' ? null : images.url,
                cta: const StatusMessageCta(
                  text: 'More',
                  destination: 'settings',
                ),
              ),
            ),
          );
          await tester.pump();
          await refresh;
          for (var attempt = 0; attempt < 20; attempt++) {
            await tester.runAsync(
              () => Future<void>.delayed(const Duration(milliseconds: 20)),
            );
            await tester.pump();
            final raw = find.byType(RawImage);
            if (imageState == 'broken' && raw.evaluate().isEmpty) break;
            if ((imageState == 'valid' || imageState == 'extreme') &&
                raw.evaluate().isNotEmpty &&
                tester.widget<RawImage>(raw).image != null) {
              break;
            }
            if (imageState == 'absent' || imageState == 'pending') break;
          }
          await tester.pumpAndSettle();
          expect(tester.takeException(), isNull);
          if (imageState == 'valid' || imageState == 'extreme') {
            final rendered = tester.widget<RawImage>(find.byType(RawImage));
            expect(rendered.image, isNotNull);
            expect(
              rendered.image!.width / rendered.image!.height,
              closeTo(imageState == 'extreme' ? 125 : 1, 0.1),
            );
          }
          if (imageState == 'broken') {
            expect(find.byType(RawImage), findsNothing);
          }
          expect(
            reporter.errors,
            isEmpty,
            reason: 'Invalid optional media is expected content failure',
          );
          await tester.ensureVisible(find.text('More'));
          await tester.pump();
          expect(find.text('More').hitTestable(), findsOneWidget);
          await tester.ensureVisible(find.text('Dismiss'));
          await tester.pump();
          await tester.tap(find.text('Dismiss'));
          await tester.pump();
          expect(controller.announcement, isNull);
          await tester.pumpWidget(const SizedBox.shrink());
          container.dispose();
          images.close();
        },
      );
    }
  }

  // AT-009 / FR-011, NFR-002 / AC-015, AC-019
  testWidgets(
    'pending optional image never blocks text/actions and is removed when covered',
    (tester) async {
      final repository = FakeStatusRepository();
      final images = Images();
      await images.start(tester);
      final container = ProviderContainer(
        overrides: [
          ...statusRuntimeOverrides,
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.darkThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context)
                  .copyWith(textScaler: const TextScaler.linear(1.5)),
              child: ServiceStatusHost(child: child!),
            ),
            home: const Scaffold(body: Text('Covered content')),
          ),
        ),
      );
      await tester.pump();
      final refresh = controller.refresh();
      repository.requests.last.complete(
        ServiceStatusDocument(
          revision: 'D',
          message: StatusMessage(
            revision: 'A',
            minVersion: '1.8.0',
            title: 'News',
            description: 'Readable text',
            imageUrl: images.url,
            cta: const StatusMessageCta(text: 'More', destination: 'settings'),
          ),
        ),
      );
      await tester.pump();
      await refresh;
      expect(find.text('Readable text'), findsOneWidget);
      expect(find.text('Dismiss'), findsOneWidget);
      expect(find.text('More'), findsOneWidget);
      expect(find.byType(StatusMessageImage), findsOneWidget);
      final cover = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          revision: 'D2',
          maintenance: StatusMaintenance(
            title: 'Break',
            description: 'Back soon',
          ),
        ),
      );
      await tester.pump();
      await tester.pump();
      await cover;
      expect(find.byType(StatusMessageImage), findsNothing);
      expect(find.text('Break'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
      images.close();
    },
  );
}

class ImageReporter implements ErrorReporter {
  final errors = <Object>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    return null;
  }
}
