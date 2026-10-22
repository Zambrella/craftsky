import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';

import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/recording_error_reporter.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('repeated avatar selections save the latest uploaded picture', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'test-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = registry.activeLease!;
    final profile = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const [],
    );
    final picker = _RepeatingPicker();
    UploadedBlob? savedAvatar;
    final repository = FakeProfileRepository(
      onFetchMe: () async => profile,
      onUpdateMe:
          ({
            displayName,
            pronouns,
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async {
            savedAvatar = avatar;
            return profile;
          },
    );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => repository,
        ),
        accountProfileImagePickerProvider.overrideWith(
          (ref, lease) async => picker,
        ),
      ],
    );
    final flow = onboardingFlowProvider(lease);
    final subscription = container.listen(flow, (_, _) {});
    addTearDown(subscription.close);
    await container.read(flow.future);
    final notifier = container.read(flow.notifier);
    for (var i = 1; i <= 3; i++) {
      await notifier.pickAvatar(ImageSource.gallery);
      await container.pump();
      final state = container.read(flow).requireValue;
      expect(state.avatarPreview, [i]);
      expect(state.avatarBlob?.ref.link, 'photo-$i');
      expect(state.uploadingAvatar, isFalse);
      expect(state.avatarUploadFailed, isFalse);
    }
    await notifier.saveAndNext();
    expect(savedAvatar?.ref.link, 'photo-3');
  });

  test('onboarding reports an unknown photo error once '
      'while keeping failure state', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'test-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = registry.activeLease!;
    final reporter = RecordingErrorReporter();
    final error = StateError('PRIVATE_IMAGE_ERROR');
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountDioProvider.overrideWith((ref, account) async => Dio()),
        errorReporterProvider.overrideWithValue(reporter),
        imagePickerProvider.overrideWithValue(_FailingImagePicker(error)),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => FakeProfileRepository(
            onFetchMe: () async => Profile(
              did: 'did:plc:alice',
              handle: 'alice.test',
              crafts: const [],
            ),
          ),
        ),
      ],
    );
    final flow = onboardingFlowProvider(lease);
    final subscription = container.listen(flow, (_, _) {});
    addTearDown(subscription.close);
    await container.read(flow.future);
    await container.read(flow.notifier).pickAvatar(ImageSource.gallery);
    expect(container.read(flow).requireValue.avatarUploadFailed, isTrue);
    expect(reporter.errors, [same(error)]);
    expect(reporter.contexts.single.feature, 'Onboarding');
    expect(
      reporter.contexts.single.workflow!.selectedFields['operationAccountDid'],
      'did:plc:alice',
    );
  });

  for (final leaveOnboarding in [false, true]) {
    test(
      leaveOnboarding
          ? 'leaving onboarding disposes the picker and prevents upload'
          : 'onboarding retains the picker until upload authorization finishes',
      () async {
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'test-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
        final lease = registry.activeLease!;
        final picker = _PendingImagePicker();
        final authorization = Completer<void>();
        final requestStarted = Completer<void>();
        var requests = 0;
        final dio = Dio(BaseOptions(baseUrl: 'https://appview.example'));
        dio.interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) async {
              expect(options.path, '/v1/blobs/images/authorization');
              requests++;
              requestStarted.complete();
              await authorization.future;
              handler.reject(DioException(requestOptions: options));
            },
          ),
        );
        final container = ProviderContainer.test(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            accountDioProvider.overrideWith((ref, account) async => dio),
            imagePickerProvider.overrideWithValue(picker),
            accountProfileRepositoryProvider.overrideWith(
              (ref, lease) async => FakeProfileRepository(
                onFetchMe: () async => Profile(
                  did: 'did:plc:alice',
                  handle: 'alice.test',
                  displayName: 'Alice',
                  crafts: const [],
                ),
              ),
            ),
          ],
        );
        final flow = onboardingFlowProvider(lease);
        final subscription = container.listen(flow, (_, _) {});
        addTearDown(subscription.close);
        await container.read(flow.future);
        final operation = container
            .read(flow.notifier)
            .pickAvatar(
              ImageSource.gallery,
            );
        await picker.opened.future;
        await container.pump();
        if (leaveOnboarding) {
          subscription.close();
          await container.pump();
        }
        picker.selected.complete(
          XFile.fromData(
            Uint8List.fromList(img.encodePng(img.Image(width: 2, height: 1))),
            name: 'photo.png',
            mimeType: 'image/png',
          ),
        );
        if (leaveOnboarding) {
          await operation;
          expect(requests, 0);
        } else {
          // Let image preparation and the authorization request run in real
          // async time, while holding the request open across provider cleanup.
          await Future.any([requestStarted.future, operation]);
          expect(requests, 1);
          await container.pump();
          expect(container.read(flow).requireValue.uploadingAvatar, isTrue);
          expect(container.read(flow).requireValue.avatarPreview, isNotEmpty);
          authorization.complete();
          await operation;
          await container.pump();
          expect(container.read(flow).requireValue.uploadingAvatar, isFalse);
        }
        expect(
          container.exists(accountProfileImagePickerProvider(lease)),
          isFalse,
        );
      },
    );
  }
}

class _RepeatingPicker extends Fake implements ProfileImagePicker {
  int picks = 0;

  @override
  Future<ProfileImagePickResult?> pickAndUpload({
    required ImageSource source,
    required void Function(Uint8List bytes) onPreviewReady,
  }) async {
    final bytes = Uint8List.fromList([++picks]);
    onPreviewReady(bytes);
    return ProfileImagePickResult(
      previewBytes: bytes,
      uploaded: UploadedImageBlob(
        blob: UploadedBlob(
          type: 'blob',
          ref: UploadedBlobRef(link: 'photo-$picks'),
          mimeType: 'image/png',
          size: bytes.length,
        ),
        cid: 'photo-$picks',
        mime: 'image/png',
        size: bytes.length,
      ),
    );
  }
}

class _PendingImagePicker extends ImagePicker {
  final opened = Completer<void>();
  final selected = Completer<XFile?>();

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) {
    opened.complete();
    return selected.future;
  }
}

class _Storage implements SessionRegistryStorage {
  new(this.registry);
  final SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {}
}

class _FailingImagePicker extends ImagePicker {
  new(this.error);
  final Object error;

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) async => Error.throwWithStackTrace(error, StackTrace.current);
}
