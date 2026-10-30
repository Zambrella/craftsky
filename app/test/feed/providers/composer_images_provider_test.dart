import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as sessions;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/composer/draft_media_write_adapter.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../../test_support/recording_error_reporter.dart';
import '../../test_support/serialized_sentry_transport.dart';

void main() {
  test(
    'image editing UT-005: activation away and back invalidates pending pixels',
    () async {
      final registry = sessions.SessionRegistry.empty()
          .upsertAndActivate(
            token: 'test-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'test-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _pngBytes(width: 8, height: 6),
              name: 'one.png',
              mimeType: 'image/png',
            ),
          ],
        ),
        registry: registry,
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      final sub = _listenComposer(container);
      addTearDown(sub.close);
      final provider = composerImagesProvider('composer');
      final notifier = container.read(provider.notifier);
      await notifier.addImages();
      final before = await _waitForState(
        container,
        (s) => s.images.isNotEmpty && s.isSubmitReady,
      );
      final pending = Completer<BasicImageEditOutput?>();
      final operation = notifier.editImage(
        before.images.single.id,
        initiatingLease: registry.activeLease,
        editor: (_) => pending.future,
      );
      final accounts = container.read(sessionRegistryProvider.notifier);
      await accounts.activate(registry.leaseFor(AccountKey('did:plc:alice'))!);
      await accounts.activate(registry.activeLease!.session);
      pending.complete(
        BasicImageEditOutput(
          bytes: _pngBytes(width: 3, height: 4),
          mimeType: 'image/png',
        ),
      );
      await operation;
      expect(container.read(provider).images.single, before.images.single);
      expect(container.read(provider).isSubmitReady, isTrue);
    },
  );

  test('image editing UT-006: consumed failure reports '
      'once with protected serialized diagnostics', () async {
    final transport = SerializedTransport();
    await Sentry.init((options) {
      options
        ..dsn = 'https://public@example.invalid/1'
        ..transport = transport;
      configureDiagnosticOptions(options, enableLogs: false);
    });
    addTearDown(Sentry.close);
    final local = <String>[];
    final oldLevel = Logger.root.level;
    Logger.root.level = Level.ALL;
    addTearDown(() => Logger.root.level = oldLevel);
    final logs = configureRootLogForwarding(platformSink: local.add);
    addTearDown(logs.cancel);
    final container = _containerWithPicker(
      _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 8, height: 6),
            name: 'one.png',
            mimeType: 'image/png',
          ),
        ],
      ),
      reporter: const SentryErrorReporter(),
    );
    addTearDown(container.dispose);
    final sub = _listenComposer(container);
    addTearDown(sub.close);
    final provider = composerImagesProvider('composer');
    final notifier = container.read(provider.notifier);
    await notifier.addImages();
    final before = await _waitForState(
      container,
      (s) => s.images.isNotEmpty && s.isSubmitReady,
    );
    final cause = PlatformException(
      code: 'codec_failed',
      message: 'PRIVATE_IMAGE_PROSE',
      details: {
        'imageBytes': 'PRIVATE_IMAGE_BYTES',
        'path': '/private/photo/PRIVATE_IMAGE_FILE.png',
      },
    );
    final stack = StackTrace.fromString(
      '#0 editPhoto (package:craftsky_app/photo.dart:12:3)',
    );
    await expectLater(
      notifier.editImage(
        before.images.single.id,
        editor: (_) async => Error.throwWithStackTrace(cause, stack),
      ),
      throwsA(same(cause)),
    );
    await Sentry.close();
    final events = transport.payloads
        .map(jsonDecode)
        .where((e) => (e as Map).containsKey('exception'))
        .toList();
    expect(events, hasLength(1));
    for (final serialized in [local.join(), jsonEncode(events)]) {
      expect(serialized, contains('media.image.edit'));
      expect(serialized, contains('PlatformException'));
      expect(serialized, contains('editPhoto'));
      expect(serialized, isNot(contains('PRIVATE_IMAGE')));
    }
    expect(container.read(provider).images.single, before.images.single);
  });

  test('image editing UT-004: output must pass final '
      'upload validation before replacement', () async {
    final media = _EditValidationMediaService();
    final container = _containerWithPicker(
      _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 8, height: 6),
            name: 'one.png',
            mimeType: 'image/png',
          ),
        ],
      ),
      media: media,
    );
    addTearDown(container.dispose);
    final sub = _listenComposer(container);
    addTearDown(sub.close);
    final provider = composerImagesProvider('composer');
    final notifier = container.read(provider.notifier);
    await notifier.addImages();
    final before = await _waitForState(
      container,
      (s) => s.images.isNotEmpty && s.isSubmitReady,
    );
    media.reject = true;
    await expectLater(
      notifier.editImage(
        before.images.single.id,
        editor: (_) async => BasicImageEditOutput(
          bytes: _pngBytes(width: 3, height: 4),
          mimeType: 'image/png',
        ),
      ),
      throwsA(isA<FormatException>()),
    );
    expect(container.read(provider).images.single, before.images.single);
    expect(container.read(provider).canSubmitImages(), isTrue);
  });

  test('image editing UT-007/AT-007: repeated edits reopen '
      'accepted pixels and map fresh draft media', () async {
    final container = _containerWithPicker(
      _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 8, height: 6),
            name: 'one.png',
            mimeType: 'image/png',
          ),
        ],
      ),
    );
    addTearDown(container.dispose);
    final sub = _listenComposer(container);
    addTearDown(sub.close);
    final provider = composerImagesProvider('composer');
    final notifier = container.read(provider.notifier);
    await notifier.addImages();
    var before = await _waitForState(
      container,
      (s) => s.images.isNotEmpty && s.isSubmitReady,
    );
    final id = before.images.single.id;
    notifier.setAltText(id, 'Kept description');
    for (var i = 2; i <= 3; i++) {
      final accepted = (before.images.single.phase as ImageReady).bytes;
      await notifier.editImage(
        id,
        editor: (request) async {
          expect(request.bytes, accepted);
          return BasicImageEditOutput(
            bytes: _pngBytes(width: i, height: 4),
            mimeType: 'image/png',
          );
        },
      );
      before = container.read(provider);
      final ready = before.images.single.phase as ImageReady;
      final write =
          draftMediaWritesFromComposer(before.images).single
              as PreparedDraftMedia;
      expect(write.bytes, ready.bytes);
      expect(write.width, i);
      expect(write.height, 4);
      expect(write.altText, 'Kept description');
      expect(write.mediaId, id);
      expect(ready.storedOrigin, isNull);
    }
  });

  test('image editing UT-003: cancel and failure retain '
      'accepted image and permit retry', () async {
    final container = _containerWithPicker(
      _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 8, height: 6),
            name: 'one.png',
            mimeType: 'image/png',
          ),
        ],
      ),
    );
    addTearDown(container.dispose);
    final sub = _listenComposer(container);
    addTearDown(sub.close);
    final provider = composerImagesProvider('composer');
    final notifier = container.read(provider.notifier);
    await notifier.addImages();
    final before = await _waitForState(
      container,
      (s) => s.images.isNotEmpty && s.isSubmitReady,
    );
    await notifier.editImage(
      before.images.single.id,
      editor: (_) async => null,
    );
    expect(container.read(provider).images.single, before.images.single);
    await expectLater(
      notifier.editImage(
        before.images.single.id,
        editor: (_) async => throw StateError('editor failed'),
      ),
      throwsStateError,
    );
    expect(container.read(provider).images.single, before.images.single);
    expect(container.read(provider).canSubmitImages(), isTrue);
  });

  test('image editing UT-005: pending edit blocks submit '
      'and removal fences late output', () async {
    final container = _containerWithPicker(
      _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 8, height: 6),
            name: 'one.png',
            mimeType: 'image/png',
          ),
        ],
      ),
    );
    addTearDown(container.dispose);
    final sub = _listenComposer(container);
    addTearDown(sub.close);
    final provider = composerImagesProvider('composer');
    final notifier = container.read(provider.notifier);
    await notifier.addImages();
    final before = await _waitForState(
      container,
      (s) => s.images.isNotEmpty && s.isSubmitReady,
    );
    final pending = Completer<BasicImageEditOutput?>();
    final editing = notifier.editImage(
      before.images.single.id,
      editor: (_) => pending.future,
    );
    expect(container.read(provider).canSubmitImages(), isFalse);
    expect(container.read(provider).canSaveDraftMedia(), isFalse);
    expect(container.read(provider).images.single, before.images.single);
    notifier.remove(before.images.single.id);
    pending.complete(
      BasicImageEditOutput(
        bytes: _pngBytes(width: 3, height: 4),
        mimeType: 'image/png',
      ),
    );
    await editing;
    expect(container.read(provider).images, isEmpty);
    expect(container.read(provider).isSubmitReady, isTrue);
  });

  test('image editing UT-002: atomic accepted pixels '
      'preserve identity and latest alt', () async {
    final picker = _FakeImagePicker(
      () async => [
        XFile.fromData(
          _pngBytes(width: 8, height: 6),
          name: 'one.png',
          mimeType: 'image/png',
        ),
        XFile.fromData(
          _pngBytes(width: 7, height: 5),
          name: 'two.png',
          mimeType: 'image/png',
        ),
      ],
    );
    final container = _containerWithPicker(picker);
    addTearDown(container.dispose);
    final sub = _listenComposer(container);
    addTearDown(sub.close);
    final provider = composerImagesProvider('composer');
    final notifier = container.read(provider.notifier);
    await notifier.addImages();
    final before = await _waitForState(
      container,
      (s) => s.images.length == 2 && s.isSubmitReady,
    );
    final target = before.images.first;
    final output = _pngBytes(width: 3, height: 4);
    await notifier.editImage(
      target.id,
      editor: (request) async {
        expect(request.bytes, (target.phase as ImageReady).bytes);
        notifier
          ..setAltText(target.id, 'Latest description')
          ..reorder(fromIndex: 0, toIndex: 1);
        return BasicImageEditOutput(bytes: output, mimeType: 'image/png');
      },
    );
    final after = container.read(provider);
    expect(after.images.first, before.images.last);
    expect(after.images.last.id, target.id);
    expect(after.images.last.altText, 'Latest description');
    final ready = after.images.last.phase as ImageReady;
    expect(ready.width, 3);
    expect(ready.height, 4);
    expect(ready.sha256, isNot((target.phase as ImageReady).sha256));
    expect(ready.storedOrigin, isNull);
    expect(after.images.last.previewBytes, ready.bytes);
  });

  test('seeds locally saved ready and unavailable draft images', () async {
    const available = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000002',
      storageRevision: '00000000-0000-4000-8000-000000000012',
      storageFileName: 'available.png',
      displayFileName: 'available.png',
      mimeType: 'image/png',
      byteLength: 3,
      sha256:
          '039058c6f2c0cb492c533b0a4d14ef77'
          'cc0f78abccced5287d84a1a2011cfb81',
      width: 1,
      height: 1,
      altText: 'Available alt',
      order: 0,
    );
    const missing = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000003',
      storageRevision: '00000000-0000-4000-8000-000000000013',
      storageFileName: 'missing.png',
      displayFileName: 'missing.png',
      mimeType: 'image/png',
      byteLength: 3,
      sha256:
          '039058c6f2c0cb492c533b0a4d14ef77'
          'cc0f78abccced5287d84a1a2011cfb81',
      width: 1,
      height: 1,
      altText: 'Missing alt',
      order: 1,
      availability: DraftMediaAvailability.unavailable,
    );
    final draft = LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000001',
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.standard,
      createdAt: DateTime.utc(2026),
      updatedAt: DateTime.utc(2026),
      content: const StandardDraftContent(text: '', languages: ['en']),
      schedule: const DraftScheduleIntent.now(),
      media: const [available, missing],
    );
    final seed = LocalPostDraftSeed(
      draft: draft,
      media: [
        HydratedDraftMedia(
          descriptor: available,
          bytes: Uint8List.fromList([1, 2, 3]),
        ),
        const HydratedDraftMedia(descriptor: missing, bytes: null),
      ],
    );
    final container = ProviderContainer.test();
    addTearDown(container.dispose);

    // Keep the composer mounted across asynchronous edited-output preparation.
    final subscription = container.listen(
      composerImagesProvider('draft-composer'),
      (_, _) {},
    );
    addTearDown(subscription.close);
    container
        .read(composerImagesProvider('draft-composer').notifier)
        .seedLocalDraft(seed);
    final state = container.read(composerImagesProvider('draft-composer'));

    expect(state.images.first.phase, isA<ImageReady>());
    expect((state.images.first.phase as ImageReady).storedOrigin, isNotNull);
    expect(state.images.last.phase, isA<ImageUnavailable>());
    expect(state.canSubmitImages(), isFalse);
    expect(state.canSaveDraftMedia(), isFalse);
    // Image editing UT-007: an edited restored image must become fresh media.
    await container
        .read(composerImagesProvider('draft-composer').notifier)
        .editImage(
          available.mediaId,
          editor: (_) async => BasicImageEditOutput(
            bytes: _pngBytes(width: 3, height: 4),
            mimeType: 'image/png',
          ),
        );
    final edited = container
        .read(composerImagesProvider('draft-composer'))
        .images
        .first;
    expect((edited.phase as ImageReady).storedOrigin, isNull);
    expect(
      draftMediaWritesFromComposer([edited]).single,
      isA<PreparedDraftMedia>(),
    );
    expect(edited.altText, available.altText);
  });

  test(
    'replaces unavailable draft media in place and preserves alt text',
    () async {
      const missing = DraftMediaDescriptor(
        mediaId: '00000000-0000-4000-8000-000000000003',
        storageRevision: '00000000-0000-4000-8000-000000000013',
        storageFileName: 'missing.png',
        displayFileName: 'missing.png',
        mimeType: 'image/png',
        byteLength: 3,
        sha256:
            '039058c6f2c0cb492c533b0a4d14ef77'
            'cc0f78abccced5287d84a1a2011cfb81',
        width: 1,
        height: 1,
        altText: 'Preserved description',
        order: 0,
        availability: DraftMediaAvailability.unavailable,
      );
      final draft = LocalPostDraft(
        id: '00000000-0000-4000-8000-000000000001',
        owner: AccountKey('did:plc:alice'),
        kind: LocalPostDraftKind.standard,
        createdAt: DateTime.utc(2026),
        updatedAt: DateTime.utc(2026),
        content: const StandardDraftContent(text: '', languages: ['en']),
        schedule: const DraftScheduleIntent.now(),
        media: const [missing],
      );
      final picker = _FakeImagePicker(
        () async => const [],
        pickSingle: () async => XFile.fromData(
          _pngBytes(width: 2, height: 1),
          name: 'replacement.png',
          mimeType: 'image/png',
        ),
      );
      final container = _containerWithPicker(picker);
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);
      final notifier =
          container.read(composerImagesProvider('composer').notifier)
            ..seedLocalDraft(
              LocalPostDraftSeed(
                draft: draft,
                media: const [
                  HydratedDraftMedia(descriptor: missing, bytes: null),
                ],
              ),
            );

      await notifier.replaceUnavailable(missing.mediaId);
      final state = await _waitForState(
        container,
        (state) => state.images.single.phase is ImageReady,
      );

      expect(state.images, hasLength(1));
      expect(state.images.single.id, missing.mediaId);
      expect(state.images.single.altText, 'Preserved description');
      final ready = state.images.single.phase as ImageReady;
      expect(ready.width, 2);
      expect(ready.height, 1);
      expect(ready.storedOrigin, isNull);
      expect(picker.lastSingleMaxWidth, mediaConfig.maxImageWidth.toDouble());
      expect(
        picker.lastSingleMaxHeight,
        mediaConfig.maxImageHeight.toDouble(),
      );
      expect(picker.lastSource, ImageSource.gallery);
    },
  );
  group('ComposerImages', () {
    test('camera capture enters the existing preparation pipeline', () async {
      final picker = _FakeImagePicker(
        () async => const [],
        pickSingle: () async => XFile.fromData(
          _pngBytes(width: 3, height: 2),
          name: 'camera.png',
          mimeType: 'image/png',
        ),
      );
      final container = _containerWithPicker(picker);
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .takePhoto();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageReady,
      );

      expect(picker.lastSource, ImageSource.camera);
      expect(picker.lastSingleMaxWidth, mediaConfig.maxImageWidth.toDouble());
      expect(picker.lastSingleMaxHeight, mediaConfig.maxImageHeight.toDouble());
      expect(state.images, hasLength(1));
      final ready = state.images.single.phase as ImageReady;
      expect(ready.width, 3);
      expect(ready.height, 2);
    });

    test('camera cancellation leaves the composer unchanged', () async {
      final picker = _FakeImagePicker(
        () async => const [],
        pickSingle: () async => null,
      );
      final container = _containerWithPicker(picker);
      addTearDown(container.dispose);

      await container
          .read(composerImagesProvider('composer').notifier)
          .takePhoto();

      expect(
        container.read(composerImagesProvider('composer')).images,
        isEmpty,
      );
      expect(picker.lastSource, ImageSource.camera);
    });

    test(
      'IT-010 camera failure retains cause and stage without '
      'changing the draft',
      () async {
        final records = <LogRecord>[];
        final logs = Logger.root.onRecord.listen(records.add);
        addTearDown(logs.cancel);
        final failure = StateError('private camera path canary');
        final picker = _FakeImagePicker(
          () async => const [],
          pickSingle: () async => throw failure,
        );
        final reporter = RecordingErrorReporter();
        final container = _containerWithPicker(picker, reporter: reporter);
        addTearDown(container.dispose);

        await container
            .read(composerImagesProvider('composer').notifier)
            .takePhoto();

        final state = container.read(composerImagesProvider('composer'));
        expect(state.images, isEmpty);
        expect(state.notice, isA<ImagePickerFailedNotice>());
        final record = records.singleWhere((r) => r.error == failure);
        final diagnostic = selectDiagnosticRecord(record);
        expect(reporter.errors, [failure]);
        expect(diagnostic['failureStage'], 'camera');
        expect(diagnostic['operation'], 'media.image.camera');
        expect(diagnostic['cause'].toString(), contains('StateError'));
        expect(diagnostic['stack'], isNotEmpty);
        expect(
          diagnostic.toString(),
          isNot(contains('private camera path canary')),
        );
      },
    );

    test('IT-010 denied camera permission is an expected diagnostic', () async {
      final records = <LogRecord>[];
      final logs = Logger.root.onRecord.listen(records.add);
      addTearDown(logs.cancel);
      final failure = PlatformException(
        code: 'camera_access_denied',
        message: 'private native prose',
      );
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => const [],
          pickSingle: () async => throw failure,
        ),
      );
      addTearDown(container.dispose);
      await container
          .read(composerImagesProvider('composer').notifier)
          .takePhoto();
      final record = records.singleWhere((r) => r.error == failure);
      expect(record.level, Level.WARNING);
      expect(selectDiagnosticRecord(record)['outcome'], 'expected');
      expect(
        selectDiagnosticRecord(record).toString(),
        isNot(contains('private native prose')),
      );
      expect(
        container.read(composerImagesProvider('composer')).images,
        isEmpty,
      );
    });

    test('surfaces picker failures without changing the draft', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(() async => throw Exception('permission denied')),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      final state = container.read(composerImagesProvider('composer'));
      expect(state.images, isEmpty);
      expect(state.notice, isA<ImagePickerFailedNotice>());
    });

    test('requests bounded images from the multi-image picker', () async {
      final picker = _FakeImagePicker(() async => const []);
      final container = _containerWithPicker(picker);
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      expect(picker.lastMultiMaxWidth, mediaConfig.maxImageWidth.toDouble());
      expect(picker.lastMultiMaxHeight, mediaConfig.maxImageHeight.toDouble());
    });

    test('rejects WebP selections before adding draft images', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              Uint8List.fromList([
                0x52,
                0x49,
                0x46,
                0x46,
                0x00,
                0x00,
                0x00,
                0x00,
                0x57,
                0x45,
                0x42,
                0x50,
              ]),
              name: 'project.webp',
              mimeType: 'image/webp',
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      final state = container.read(composerImagesProvider('composer'));
      expect(state.images, isEmpty);
      expect(state.notice, isA<UnsupportedImagesNotice>());
    });

    test('fails oversized originals before reading full bytes', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              Uint8List.fromList(List<int>.filled(16, 0xff)),
              name: 'large.jpg',
              mimeType: 'image/jpeg',
              length: mediaConfig.maxSourceImageBytes + 1,
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) =>
            state.images.length == 1 &&
            state.images.single.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect(phase, isA<ImageFailed>());
      final failure = (phase as ImageFailed).failure;
      expect(failure, isA<ImageTooLarge>());
      expect((failure as ImageTooLarge).source, isTrue);
      expect(failure.message, contains('too large to process'));
    });

    test(
      'prepares accepted images locally without uploading',
      () async {
        final originalBytes = _jpegBytes(width: 3, height: 2);
        final api = _FakePostApiClient();
        final container = _containerWithPicker(
          _FakeImagePicker(
            () async => [
              XFile.fromData(
                originalBytes,
                path: '/tmp/PROJECT.JPG',
              ),
            ],
          ),
          api: api,
        );
        addTearDown(container.dispose);
        final sub = _listenComposer(container);
        addTearDown(sub.close);

        await container
            .read(composerImagesProvider('composer').notifier)
            .addImages();
        final state = await _waitForState(
          container,
          (state) => state.images.singleOrNull?.phase is ImageReady,
        );

        final image = state.images.single;
        expect(image.fileName, 'PROJECT.JPG');
        expect(image.mimeType, 'image/jpeg');
        expect(image.previewBytes, isNotEmpty);
        expect(image.previewAspectRatio?.width, 3);
        expect(image.previewAspectRatio?.height, 2);
        expect(api.uploadCount, 0);

        final ready = image.phase as ImageReady;
        expect(ready.bytes, image.previewBytes);
        expect(identical(ready.bytes, image.previewBytes), isTrue);
        expect(ready.mimeType, 'image/jpeg');
        expect(ready.width, 3);
        expect(ready.height, 2);
        expect(ready.sha256, isNotEmpty);
      },
    );

    test('prepares at most one selected image at a time', () async {
      final bytes = _pngBytes(width: 256, height: 256);
      var batch = 0;
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async {
            final offset = batch * (mediaConfig.maxImages ~/ 2);
            batch += 1;
            return List.generate(
              mediaConfig.maxImages ~/ 2,
              (index) => XFile.fromData(
                bytes,
                name: 'project-${offset + index}.png',
                mimeType: 'image/png',
              ),
            );
          },
        ),
      );
      addTearDown(container.dispose);
      var maxPreparing = 0;
      final sub = container.listen(
        composerImagesProvider('composer'),
        (_, state) {
          final preparing = state.images
              .where((image) => image.phase is ImagePreparing)
              .length;
          if (preparing > maxPreparing) maxPreparing = preparing;
        },
        fireImmediately: true,
      );
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      await _waitForState(
        container,
        (state) => state.images.any((image) => image.phase is ImagePreparing),
      );
      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) =>
            state.images.length == mediaConfig.maxImages &&
            state.images.every((image) => image.phase is ImageReady),
      );

      expect(state.images, hasLength(mediaConfig.maxImages));
      expect(maxPreparing, 1);
    });

    test('maps unsupported original headers to image failure', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _jpegBytes(width: 1, height: 1),
              name: 'spoofed.jpg',
              mimeType: 'image/jpeg',
            ),
          ],
        ),
        media: const _UnsupportedOriginalMediaService(),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect((phase as ImageFailed).failure, isA<UnsupportedImageType>());
    });

    test('maps corrupt accepted bytes to preparation failure', () async {
      final logRecords = <LogRecord>[];
      final logSubscription = Logger.root.onRecord.listen(logRecords.add);
      addTearDown(logSubscription.cancel);
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              Uint8List.fromList([1, 2, 3, 4]),
              name: 'corrupt.jpg',
              mimeType: 'image/jpeg',
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect((phase as ImageFailed).failure, isA<ImagePreparationFailed>());
      expect(phase.failure.canRetry, isFalse);
      final messages = logRecords.map((record) => record.message).join('\n');
      final diagnostic = selectDiagnosticRecord(logRecords.last);
      expect(diagnostic['failureStage'], 'read');
      expect(diagnostic['cause'], isNotNull);
      expect(diagnostic['stack'], isNotEmpty);
      expect(diagnostic.toString(), isNot(contains('corrupt.jpg')));
      expect(messages, contains('composer image pipeline failed'));
      expect(messages, isNot(contains('corrupt.jpg')));
    });

    test(
      'allows retry when preparation fails after bytes are retained',
      () async {
        final container = _containerWithPicker(
          _FakeImagePicker(
            () async => [
              XFile.fromData(
                _jpegHeader(width: 1, height: 1),
                name: 'incomplete.jpg',
                mimeType: 'image/jpeg',
              ),
            ],
          ),
        );
        addTearDown(container.dispose);
        final sub = _listenComposer(container);
        addTearDown(sub.close);

        await container
            .read(composerImagesProvider('composer').notifier)
            .addImages();
        final state = await _waitForState(
          container,
          (state) => state.images.singleOrNull?.phase is ImageFailed,
        );
        final phase = state.images.single.phase as ImageFailed;

        expect(phase.failure, isA<ImagePreparationFailed>());
        expect(phase.failure.canRetry, isTrue);
        expect(state.images.single.previewBytes, isNull);
      },
    );

    test('maps oversized source geometry to a non-retryable failure', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _jpegHeader(width: 8000, height: 6001),
              name: 'large-dimensions.jpg',
              mimeType: 'image/jpeg',
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);
      final notifier = container.read(
        composerImagesProvider('composer').notifier,
      );

      await notifier.addImages();
      final failed = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );
      final phase = failed.images.single.phase as ImageFailed;

      expect(phase.failure, isA<ImageSourceDimensionsTooLarge>());
      expect(phase.failure.canRetry, isFalse);
      notifier.retry(failed.images.single.id);
      await Future<void>.delayed(const Duration(milliseconds: 20));
      expect(container.read(composerImagesProvider('composer')), failed);
    });

    test('maps prepared size failures before upload starts', () async {
      final api = _FakePostApiClient();
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _pngBytes(width: 1, height: 1),
              name: 'project.png',
              mimeType: 'image/png',
            ),
          ],
        ),
        api: api,
        media: const _PreparedTooLargeMediaService(),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect((phase as ImageFailed).failure, isA<ImageTooLarge>());
      expect(api.uploadCount, 0);
    });

    test('emits limit notice without opening picker when full', () async {
      final picker = _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 1, height: 1),
            name: 'project.png',
            mimeType: 'image/png',
          ),
        ],
      );
      final container = _containerWithPicker(
        picker,
        media: const _NoSlotsMediaService(),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      final notice = container.read(composerImagesProvider('composer')).notice;
      expect(
        notice,
        isA<ImageSelectionLimitNotice>()
            .having((notice) => notice.maxImages, 'maxImages', 0)
            .having((notice) => notice.acceptedCount, 'acceptedCount', 0),
      );
      expect(picker.pickCount, 0);
    });
  });
}

ProviderSubscription<ComposerImagesState> _listenComposer(
  ProviderContainer container,
) {
  return container.listen(
    composerImagesProvider('composer'),
    (_, _) {},
    fireImmediately: true,
  );
}

ProviderContainer _containerWithPicker(
  ImagePicker picker, {
  PostApiClient? api,
  ComposerImageMediaService? media,
  ErrorReporter reporter = const NoopErrorReporter(),
  sessions.SessionRegistry? registry,
}) {
  return ProviderContainer.test(
    overrides: [
      if (registry != null)
        secureSessionRegistryStorageProvider.overrideWithValue(
          _EditRegistryStorage(registry),
        ),
      imagePickerProvider.overrideWithValue(picker),
      errorReporterProvider.overrideWithValue(reporter),
      composerImageMediaServiceProvider.overrideWithValue(
        media ?? const ComposerImageMediaService(),
      ),
      postApiClientProvider.overrideWith(
        (ref) =>
            api ??
            PostApiClient(Dio(BaseOptions(baseUrl: 'https://example.com'))),
      ),
    ],
  );
}

Future<ComposerImagesState> _waitForState(
  ProviderContainer container,
  bool Function(ComposerImagesState state) predicate,
) async {
  final provider = composerImagesProvider('composer');
  final completed = Completer<ComposerImagesState>();
  final subscription = container.listen(provider, (_, state) {
    if (predicate(state) && !completed.isCompleted) completed.complete(state);
  }, fireImmediately: true);
  try {
    return await completed.future.timeout(
      const Duration(seconds: 10),
      onTimeout: () => throw StateError(
        'Timed out waiting for composer image state: '
        '${container.read(provider)}',
      ),
    );
  } finally {
    subscription.close();
  }
}

class _FakeImagePicker extends ImagePicker {
  new(this._pick, {this.pickSingle});

  final Future<List<XFile>> Function() _pick;
  final Future<XFile?> Function()? pickSingle;
  int pickCount = 0;
  double? lastMultiMaxWidth;
  double? lastMultiMaxHeight;
  double? lastSingleMaxWidth;
  double? lastSingleMaxHeight;
  ImageSource? lastSource;

  @override
  Future<List<XFile>> pickMultiImage({
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    int? limit,
    bool requestFullMetadata = true,
  }) {
    pickCount += 1;
    lastMultiMaxWidth = maxWidth;
    lastMultiMaxHeight = maxHeight;
    return _pick();
  }

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) {
    lastSource = source;
    lastSingleMaxWidth = maxWidth;
    lastSingleMaxHeight = maxHeight;
    return pickSingle?.call() ?? Future<XFile?>.value();
  }
}

class _FakePostApiClient extends PostApiClient {
  new()
    : _uploadHandler = null,
      super(Dio(BaseOptions(baseUrl: 'https://example.com')));

  final Future<UploadedImageBlob> Function({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  })?
  _uploadHandler;
  int uploadCount = 0;
  List<int>? lastBytes;
  String? lastMimeType;
  CancelToken? lastCancelToken;

  @override
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    uploadCount += 1;
    lastBytes = bytes;
    lastMimeType = mimeType;
    lastCancelToken = cancelToken;

    final handler = _uploadHandler;
    if (handler != null) {
      return await handler(
        bytes: bytes,
        mimeType: mimeType,
        onSendProgress: onSendProgress,
        onReceiveProgress: onReceiveProgress,
        cancelToken: cancelToken,
      );
    }

    onSendProgress?.call(bytes.length, bytes.length);
    return _uploadedBlob(mimeType: mimeType, size: bytes.length);
  }
}

class _PreparedTooLargeMediaService extends ComposerImageMediaService {
  const new();

  @override
  PreparedUploadValidationResult validatePreparedUploadBytes({
    required int originalBytes,
    required int preparedBytes,
    required int width,
    required int height,
  }) {
    return const PreparedUploadValidationResult(
      canUpload: false,
      rejectedReason: PreparedUploadRejection.tooLarge,
    );
  }
}

class _UnsupportedOriginalMediaService extends ComposerImageMediaService {
  const new();

  @override
  OriginalImageValidationResult validateOriginalImage({
    required int sizeBytes,
    required String fileName,
    required String mimeType,
    required Uint8List headerBytes,
  }) {
    return const OriginalImageValidationResult(
      canPrepare: false,
      rejectedReason: OriginalImageRejection.unsupportedType,
    );
  }
}

class _NoSlotsMediaService extends ComposerImageMediaService {
  const new();

  @override
  int get maxImages => 0;
}

const _testCid = 'bafkreicomposerimagetest';

UploadedImageBlob _uploadedBlob({
  required String mimeType,
  required int size,
  String cid = _testCid,
}) {
  return UploadedImageBlob(
    blob: UploadedBlob(
      type: 'blob',
      ref: UploadedBlobRef(link: cid),
      mimeType: mimeType,
      size: size,
    ),
    cid: cid,
    mime: mimeType,
    size: size,
  );
}

Uint8List _jpegBytes({required int width, required int height}) {
  return Uint8List.fromList(
    img.encodeJpg(img.Image(width: width, height: height)),
  );
}

Uint8List _jpegHeader({required int width, required int height}) {
  return Uint8List.fromList([
    0xff,
    0xd8,
    0xff,
    0xc0,
    0x00,
    0x11,
    0x08,
    height >> 8,
    height & 0xff,
    width >> 8,
    width & 0xff,
    0x03,
    0x01,
    0x11,
    0x00,
    0x02,
    0x11,
    0x00,
    0x03,
    0x11,
    0x00,
    0xff,
    0xd9,
  ]);
}

Uint8List _pngBytes({required int width, required int height}) {
  return Uint8List.fromList(
    img.encodePng(img.Image(width: width, height: height)),
  );
}

class _EditValidationMediaService extends ComposerImageMediaService {
  bool reject = false;
  @override
  PreparedUploadValidationResult validatePreparedUploadBytes({
    required int originalBytes,
    required int preparedBytes,
    required int width,
    required int height,
  }) => reject
      ? const PreparedUploadValidationResult(
          canUpload: false,
          rejectedReason: PreparedUploadRejection.tooLarge,
        )
      : super.validatePreparedUploadBytes(
          originalBytes: originalBytes,
          preparedBytes: preparedBytes,
          width: width,
          height: height,
        );
}

class _EditRegistryStorage implements SessionRegistryStorage {
  new(this.registry);
  final sessions.SessionRegistry registry;
  @override
  Future<sessions.SessionRegistry> read() async => registry;
  @override
  Future<void> write(sessions.SessionRegistry value) async {}
}
