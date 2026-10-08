import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

const _serviceToken = 'service.jwt.secret';
const _jobId = 'job-one';

void main() {
  testWidgets(
    'IT-009 standard interruption cancels and cannot resume '
    'after reconstruction',
    (tester) async {
      final adapter = _BlockingVideoAdapter();
      final storage = _RegistryStorage(_aliceRegistry());
      final repository = FakePostRepository();

      await _pumpStandardComposer(
        tester,
        adapter: adapter,
        storage: storage,
        repository: repository,
      );
      await tester.enterText(find.byType(TextField).first, 'Interrupted post');
      await _pumpUntilPostEnabled(tester);
      await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
      await _pumpUntil(tester, () => adapter.started == 1);

      tester.binding.handleAppLifecycleStateChanged(
        AppLifecycleState.paused,
      );
      await _pumpUntil(tester, () => adapter.canceled == 1);

      expect(repository.lastCreateVideo, isNull);
      _expectNoEphemeralPersistence(storage);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await _pumpStandardComposer(
        tester,
        adapter: adapter,
        storage: storage,
        repository: repository,
      );
      await tester.pump(const Duration(milliseconds: 100));

      expect(adapter.started, 1);
      expect(repository.lastCreateVideo, isNull);
      _expectNoEphemeralPersistence(storage);
    },
  );

  testWidgets(
    'IT-009 project account replacement and disposal cannot resume '
    'after reconstruction',
    (tester) async {
      final adapter = _BlockingVideoAdapter();
      final initial = _twoAccountRegistry();
      final storage = _RegistryStorage(initial);
      final repository = FakePostRepository();

      await _pumpProjectComposer(
        tester,
        adapter: adapter,
        storage: storage,
        repository: repository,
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(MaterialApp)),
      );
      await container.read(sessionRegistryProvider.future);
      await _selectCraft(tester, 'Embroidery');
      for (var i = 0; i < 3 && _projectBodyField().evaluate().isEmpty; i++) {
        await _goNext(tester);
      }
      await tester.enterText(_projectBodyField(), 'Interrupted project');
      await _pumpUntilPostEnabled(tester);
      await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
      await _pumpUntil(tester, () => adapter.started == 1);

      final bob = initial.leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bob);
      await _pumpUntil(tester, () => adapter.canceled == 1);
      await tester.pumpWidget(const SizedBox.shrink());

      expect(repository.lastCreateVideo, isNull);
      _expectNoEphemeralPersistence(storage);

      await _pumpProjectComposer(
        tester,
        adapter: adapter,
        storage: storage,
        repository: repository,
      );
      await tester.pump(const Duration(milliseconds: 100));

      expect(adapter.started, 1);
      expect(repository.lastCreateVideo, isNull);
      _expectNoEphemeralPersistence(storage);
    },
  );

  testWidgets(
    'IT-009 standard lifecycle interruption during authorization stops '
    'before upload',
    (tester) async {
      final api = _BlockingAuthorizationPostApiClient();
      final adapter = _NoRequestVideoAdapter();
      final storage = _RegistryStorage(_aliceRegistry());
      final repository = FakePostRepository();

      await _pumpStandardComposer(
        tester,
        adapter: adapter,
        storage: storage,
        repository: repository,
        api: api,
      );
      await tester.enterText(find.byType(TextField).first, 'Interrupted post');
      await _pumpUntilPostEnabled(tester);
      await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
      await _pumpUntil(tester, () => api.authorizationStarted);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      api.releaseAuthorization();
      await tester.pumpAndSettle();

      expect(api.authorizationCalls, 1);
      expect(adapter.uploadRequests, 0);
      expect(adapter.pollRequests, 0);
      expect(repository.lastCreateLangs, isNull);
      _expectNoEphemeralPersistence(storage);
      _resumeLifecycle(tester);
    },
  );

  testWidgets(
    'IT-009 project lifecycle interruption during authorization stops '
    'before upload',
    (tester) async {
      final api = _BlockingAuthorizationPostApiClient();
      final adapter = _NoRequestVideoAdapter();
      final storage = _RegistryStorage(_aliceRegistry());
      final repository = FakePostRepository();

      await _pumpProjectComposer(
        tester,
        adapter: adapter,
        storage: storage,
        repository: repository,
        api: api,
      );
      await _selectCraft(tester, 'Embroidery');
      for (var i = 0; i < 3 && _projectBodyField().evaluate().isEmpty; i++) {
        await _goNext(tester);
      }
      await tester.enterText(_projectBodyField(), 'Interrupted project');
      await _pumpUntilPostEnabled(tester);
      await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
      await _pumpUntil(tester, () => api.authorizationStarted);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      api.releaseAuthorization();
      await tester.pumpAndSettle();

      expect(api.authorizationCalls, 1);
      expect(adapter.uploadRequests, 0);
      expect(adapter.pollRequests, 0);
      expect(repository.lastCreateLangs, isNull);
      _expectNoEphemeralPersistence(storage);
      _resumeLifecycle(tester);
    },
  );
}

Future<void> _pumpStandardComposer(
  WidgetTester tester, {
  required HttpClientAdapter adapter,
  required _RegistryStorage storage,
  required FakePostRepository repository,
  PostApiClient? api,
}) => _pumpComposer(
  tester,
  adapter: adapter,
  storage: storage,
  repository: repository,
  api: api,
  child: PostComposerSheet(
    composerId: 'credential-standard',
    videoController: _selectedVideoController(),
  ),
);

Future<void> _pumpProjectComposer(
  WidgetTester tester, {
  required HttpClientAdapter adapter,
  required _RegistryStorage storage,
  required FakePostRepository repository,
  PostApiClient? api,
}) => _pumpComposer(
  tester,
  adapter: adapter,
  storage: storage,
  repository: repository,
  api: api,
  project: true,
  child: ProjectComposerSheet(
    composerId: 'credential-project',
    videoController: _selectedVideoController(),
  ),
);

Future<void> _pumpComposer(
  WidgetTester tester, {
  required HttpClientAdapter adapter,
  required _RegistryStorage storage,
  required FakePostRepository repository,
  required Widget child,
  PostApiClient? api,
  bool project = false,
}) async {
  final dio = Dio()..httpClientAdapter = adapter;
  final service = VideoServiceClient.forTesting(
    uploadEndpoint: Uri.parse(
      'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
    ),
    dio: dio,
  );
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postApiClientProvider.overrideWithValue(api ?? _VideoPostApiClient()),
        videoServiceClientProvider.overrideWithValue(service),
        postRepositoryProvider.overrideWithValue(repository),
        if (project)
          composerImagesProvider('credential-project').overrideWithValue(
            const ComposerImagesState(images: []),
          ),
      ],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: child,
        ),
      ),
    ),
  );
  final container = ProviderScope.containerOf(
    tester.element(find.byType(MaterialApp)),
  );
  await container.read(sessionRegistryProvider.future);
  await tester.pumpAndSettle();
}

ComposerVideoController _selectedVideoController() =>
    ComposerVideoController(picker: _NoopPicker())
      ..restoredSelection = LocalVideoSelection(
        displayName: 'private.mp4',
        mimeType: 'video/mp4',
        byteLength: 12,
        duration: null,
        width: 1080,
        height: 1920,
        headerBytes: Uint8List(0),
        openRead: () => Stream.value(List.filled(12, 1)),
        posterBytes: Uint8List.fromList([1]),
        altText: 'A private video',
      );

SessionRegistry _aliceRegistry() => SessionRegistry.empty().upsertAndActivate(
  token: 'craftsky-session-alice',
  did: 'did:plc:alice',
  handle: 'alice.example',
);

SessionRegistry _twoAccountRegistry() {
  final alice = _aliceRegistry();
  final aliceLease = alice.activeLease!.session;
  return alice
      .upsertAndActivate(
        token: 'craftsky-session-bob',
        did: 'did:plc:bob',
        handle: 'bob.example',
      )
      .activate(aliceLease);
}

void _expectNoEphemeralPersistence(_RegistryStorage storage) {
  final snapshots = [storage.registry.toJson(), ...storage.writes];
  expect(snapshots, everyElement(isNot(contains(_serviceToken))));
  expect(snapshots, everyElement(isNot(contains(_jobId))));
}

void _resumeLifecycle(WidgetTester tester) {
  tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
  tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
  tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
}

Future<void> _pumpUntil(
  WidgetTester tester,
  bool Function() condition,
) async {
  for (var attempt = 0; attempt < 100; attempt++) {
    await tester.pump(const Duration(milliseconds: 20));
    if (condition()) return;
  }
  fail('Condition did not become true');
}

Future<void> _pumpUntilPostEnabled(WidgetTester tester) => _pumpUntil(
  tester,
  () {
    final finder = find.widgetWithText(ChunkyButton, 'Post');
    return finder.evaluate().isNotEmpty &&
        tester.widget<ChunkyButton>(finder).onPressed != null;
  },
);

Finder _projectBodyField() => find.descendant(
  of: find.byKey(const Key('project-composer-body-editor')),
  matching: find.byType(TextField),
);

Future<void> _goNext(WidgetTester tester) async {
  await tester.tap(find.byKey(const Key('project-composer-primary-action')));
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  await tester.tap(find.byKey(const Key('craftType-select-button')));
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

final class _VideoPostApiClient extends PostApiClient {
  _VideoPostApiClient() : super(Dio());

  @override
  Future<VideoUploadLimits> getVideoUploadLimits() async =>
      const VideoUploadLimits(canUpload: true);

  @override
  Future<VideoUploadAuthorization> authorizeVideoUpload() async =>
      VideoUploadAuthorization.fromMap({
        'token': _serviceToken,
        'expiresAt': '2030-01-01T00:00:00Z',
      });
}

final class _BlockingAuthorizationPostApiClient extends PostApiClient {
  _BlockingAuthorizationPostApiClient() : super(Dio());

  final _authorizationGate = Completer<void>();
  bool authorizationStarted = false;
  int authorizationCalls = 0;

  void releaseAuthorization() => _authorizationGate.complete();

  @override
  Future<VideoUploadLimits> getVideoUploadLimits() async =>
      const VideoUploadLimits(canUpload: true);

  @override
  Future<VideoUploadAuthorization> authorizeVideoUpload() async {
    authorizationCalls++;
    authorizationStarted = true;
    await _authorizationGate.future;
    return VideoUploadAuthorization.fromMap({
      'token': _serviceToken,
      'expiresAt': '2030-01-01T00:00:00Z',
    });
  }
}

final class _BlockingVideoAdapter implements HttpClientAdapter {
  int started = 0;
  int canceled = 0;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) {
    started++;
    final response = Completer<ResponseBody>();
    final subscription = requestStream?.listen((_) {});
    if (cancelFuture case final cancellation?) {
      unawaited(
        cancellation.then((_) async {
          canceled++;
          await subscription?.cancel();
          if (!response.isCompleted) {
            response.completeError(
              DioException.requestCancelled(
                requestOptions: options,
                reason: 'interrupted',
              ),
            );
          }
        }),
      );
    }
    return response.future;
  }

  @override
  void close({bool force = false}) {}
}

final class _NoRequestVideoAdapter implements HttpClientAdapter {
  int uploadRequests = 0;
  int pollRequests = 0;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) {
    if (options.path.endsWith('uploadVideo')) uploadRequests++;
    if (options.path.endsWith('getJobStatus')) pollRequests++;
    return Future.error(
      StateError('Video transport must not run after interruption'),
    );
  }

  @override
  void close({bool force = false}) {}
}

final class _NoopPicker implements ExistingVideoPicker {
  @override
  Future<LocalVideoSelection?> pickExisting() async => null;
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  SessionRegistry registry;
  final List<String> writes = [];

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
    writes.add(registry.toJson());
  }
}
