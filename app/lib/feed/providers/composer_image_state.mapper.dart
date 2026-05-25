// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'composer_image_state.dart';

class ComposerImagesStateMapper extends ClassMapperBase<ComposerImagesState> {
  ComposerImagesStateMapper._();

  static ComposerImagesStateMapper? _instance;
  static ComposerImagesStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ComposerImagesStateMapper._());
      ComposerImageDraftMapper.ensureInitialized();
      ComposerImageNoticeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ComposerImagesState';

  static List<ComposerImageDraft> _$images(ComposerImagesState v) => v.images;
  static const Field<ComposerImagesState, List<ComposerImageDraft>> _f$images =
      Field('images', _$images);
  static ComposerImageNotice? _$notice(ComposerImagesState v) => v.notice;
  static const Field<ComposerImagesState, ComposerImageNotice> _f$notice =
      Field('notice', _$notice, opt: true);

  @override
  final MappableFields<ComposerImagesState> fields = const {
    #images: _f$images,
    #notice: _f$notice,
  };

  static ComposerImagesState _instantiate(DecodingData data) {
    return ComposerImagesState(
      images: data.dec(_f$images),
      notice: data.dec(_f$notice),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ComposerImagesState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ComposerImagesState>(map);
  }

  static ComposerImagesState fromJson(String json) {
    return ensureInitialized().decodeJson<ComposerImagesState>(json);
  }
}
mixin ComposerImagesStateMappable {
  String toJson() {
    return ComposerImagesStateMapper.ensureInitialized()
        .encodeJson<ComposerImagesState>(this as ComposerImagesState);
  }

  Map<String, dynamic> toMap() {
    return ComposerImagesStateMapper.ensureInitialized()
        .encodeMap<ComposerImagesState>(this as ComposerImagesState);
  }

  ComposerImagesStateCopyWith<
    ComposerImagesState,
    ComposerImagesState,
    ComposerImagesState
  >
  get copyWith =>
      _ComposerImagesStateCopyWithImpl<
        ComposerImagesState,
        ComposerImagesState
      >(this as ComposerImagesState, $identity, $identity);
  @override
  String toString() {
    return ComposerImagesStateMapper.ensureInitialized().stringifyValue(
      this as ComposerImagesState,
    );
  }

  @override
  bool operator ==(Object other) {
    return ComposerImagesStateMapper.ensureInitialized().equalsValue(
      this as ComposerImagesState,
      other,
    );
  }

  @override
  int get hashCode {
    return ComposerImagesStateMapper.ensureInitialized().hashValue(
      this as ComposerImagesState,
    );
  }
}

extension ComposerImagesStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ComposerImagesState, $Out> {
  ComposerImagesStateCopyWith<$R, ComposerImagesState, $Out>
  get $asComposerImagesState => $base.as(
    (v, t, t2) => _ComposerImagesStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ComposerImagesStateCopyWith<
  $R,
  $In extends ComposerImagesState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ComposerImageDraft,
    ComposerImageDraftCopyWith<$R, ComposerImageDraft, ComposerImageDraft>
  >
  get images;
  ComposerImageNoticeCopyWith<$R, ComposerImageNotice, ComposerImageNotice>?
  get notice;
  $R call({List<ComposerImageDraft>? images, ComposerImageNotice? notice});
  ComposerImagesStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ComposerImagesStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ComposerImagesState, $Out>
    implements ComposerImagesStateCopyWith<$R, ComposerImagesState, $Out> {
  _ComposerImagesStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ComposerImagesState> $mapper =
      ComposerImagesStateMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ComposerImageDraft,
    ComposerImageDraftCopyWith<$R, ComposerImageDraft, ComposerImageDraft>
  >
  get images => ListCopyWith(
    $value.images,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(images: v),
  );
  @override
  ComposerImageNoticeCopyWith<$R, ComposerImageNotice, ComposerImageNotice>?
  get notice => $value.notice?.copyWith.$chain((v) => call(notice: v));
  @override
  $R call({List<ComposerImageDraft>? images, Object? notice = $none}) => $apply(
    FieldCopyWithData({
      if (images != null) #images: images,
      if (notice != $none) #notice: notice,
    }),
  );
  @override
  ComposerImagesState $make(CopyWithData data) => ComposerImagesState(
    images: data.get(#images, or: $value.images),
    notice: data.get(#notice, or: $value.notice),
  );

  @override
  ComposerImagesStateCopyWith<$R2, ComposerImagesState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ComposerImagesStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ComposerImageDraftMapper extends ClassMapperBase<ComposerImageDraft> {
  ComposerImageDraftMapper._();

  static ComposerImageDraftMapper? _instance;
  static ComposerImageDraftMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ComposerImageDraftMapper._());
      ComposerImagePhaseMapper.ensureInitialized();
      CreatePostImageAspectRatioMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ComposerImageDraft';

  static String _$id(ComposerImageDraft v) => v.id;
  static const Field<ComposerImageDraft, String> _f$id = Field('id', _$id);
  static String _$fileName(ComposerImageDraft v) => v.fileName;
  static const Field<ComposerImageDraft, String> _f$fileName = Field(
    'fileName',
    _$fileName,
  );
  static String _$mimeType(ComposerImageDraft v) => v.mimeType;
  static const Field<ComposerImageDraft, String> _f$mimeType = Field(
    'mimeType',
    _$mimeType,
  );
  static String _$altText(ComposerImageDraft v) => v.altText;
  static const Field<ComposerImageDraft, String> _f$altText = Field(
    'altText',
    _$altText,
  );
  static ComposerImagePhase _$phase(ComposerImageDraft v) => v.phase;
  static const Field<ComposerImageDraft, ComposerImagePhase> _f$phase = Field(
    'phase',
    _$phase,
  );
  static Uint8List? _$previewBytes(ComposerImageDraft v) => v.previewBytes;
  static const Field<ComposerImageDraft, Uint8List> _f$previewBytes = Field(
    'previewBytes',
    _$previewBytes,
    opt: true,
  );
  static CreatePostImageAspectRatio? _$previewAspectRatio(
    ComposerImageDraft v,
  ) => v.previewAspectRatio;
  static const Field<ComposerImageDraft, CreatePostImageAspectRatio>
  _f$previewAspectRatio = Field(
    'previewAspectRatio',
    _$previewAspectRatio,
    opt: true,
  );

  @override
  final MappableFields<ComposerImageDraft> fields = const {
    #id: _f$id,
    #fileName: _f$fileName,
    #mimeType: _f$mimeType,
    #altText: _f$altText,
    #phase: _f$phase,
    #previewBytes: _f$previewBytes,
    #previewAspectRatio: _f$previewAspectRatio,
  };

  static ComposerImageDraft _instantiate(DecodingData data) {
    return ComposerImageDraft(
      id: data.dec(_f$id),
      fileName: data.dec(_f$fileName),
      mimeType: data.dec(_f$mimeType),
      altText: data.dec(_f$altText),
      phase: data.dec(_f$phase),
      previewBytes: data.dec(_f$previewBytes),
      previewAspectRatio: data.dec(_f$previewAspectRatio),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ComposerImageDraft fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ComposerImageDraft>(map);
  }

  static ComposerImageDraft fromJson(String json) {
    return ensureInitialized().decodeJson<ComposerImageDraft>(json);
  }
}

mixin ComposerImageDraftMappable {
  String toJson() {
    return ComposerImageDraftMapper.ensureInitialized()
        .encodeJson<ComposerImageDraft>(this as ComposerImageDraft);
  }

  Map<String, dynamic> toMap() {
    return ComposerImageDraftMapper.ensureInitialized()
        .encodeMap<ComposerImageDraft>(this as ComposerImageDraft);
  }

  ComposerImageDraftCopyWith<
    ComposerImageDraft,
    ComposerImageDraft,
    ComposerImageDraft
  >
  get copyWith =>
      _ComposerImageDraftCopyWithImpl<ComposerImageDraft, ComposerImageDraft>(
        this as ComposerImageDraft,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ComposerImageDraftMapper.ensureInitialized().stringifyValue(
      this as ComposerImageDraft,
    );
  }

  @override
  bool operator ==(Object other) {
    return ComposerImageDraftMapper.ensureInitialized().equalsValue(
      this as ComposerImageDraft,
      other,
    );
  }

  @override
  int get hashCode {
    return ComposerImageDraftMapper.ensureInitialized().hashValue(
      this as ComposerImageDraft,
    );
  }
}

extension ComposerImageDraftValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ComposerImageDraft, $Out> {
  ComposerImageDraftCopyWith<$R, ComposerImageDraft, $Out>
  get $asComposerImageDraft => $base.as(
    (v, t, t2) => _ComposerImageDraftCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ComposerImageDraftCopyWith<
  $R,
  $In extends ComposerImageDraft,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ComposerImagePhaseCopyWith<$R, ComposerImagePhase, ComposerImagePhase>
  get phase;
  CreatePostImageAspectRatioCopyWith<
    $R,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >?
  get previewAspectRatio;
  $R call({
    String? id,
    String? fileName,
    String? mimeType,
    String? altText,
    ComposerImagePhase? phase,
    Uint8List? previewBytes,
    CreatePostImageAspectRatio? previewAspectRatio,
  });
  ComposerImageDraftCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ComposerImageDraftCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ComposerImageDraft, $Out>
    implements ComposerImageDraftCopyWith<$R, ComposerImageDraft, $Out> {
  _ComposerImageDraftCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ComposerImageDraft> $mapper =
      ComposerImageDraftMapper.ensureInitialized();
  @override
  ComposerImagePhaseCopyWith<$R, ComposerImagePhase, ComposerImagePhase>
  get phase => $value.phase.copyWith.$chain((v) => call(phase: v));
  @override
  CreatePostImageAspectRatioCopyWith<
    $R,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >?
  get previewAspectRatio => $value.previewAspectRatio?.copyWith.$chain(
    (v) => call(previewAspectRatio: v),
  );
  @override
  $R call({
    String? id,
    String? fileName,
    String? mimeType,
    String? altText,
    ComposerImagePhase? phase,
    Object? previewBytes = $none,
    Object? previewAspectRatio = $none,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (fileName != null) #fileName: fileName,
      if (mimeType != null) #mimeType: mimeType,
      if (altText != null) #altText: altText,
      if (phase != null) #phase: phase,
      if (previewBytes != $none) #previewBytes: previewBytes,
      if (previewAspectRatio != $none) #previewAspectRatio: previewAspectRatio,
    }),
  );
  @override
  ComposerImageDraft $make(CopyWithData data) => ComposerImageDraft(
    id: data.get(#id, or: $value.id),
    fileName: data.get(#fileName, or: $value.fileName),
    mimeType: data.get(#mimeType, or: $value.mimeType),
    altText: data.get(#altText, or: $value.altText),
    phase: data.get(#phase, or: $value.phase),
    previewBytes: data.get(#previewBytes, or: $value.previewBytes),
    previewAspectRatio: data.get(
      #previewAspectRatio,
      or: $value.previewAspectRatio,
    ),
  );

  @override
  ComposerImageDraftCopyWith<$R2, ComposerImageDraft, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ComposerImageDraftCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ComposerImagePhaseMapper extends ClassMapperBase<ComposerImagePhase> {
  ComposerImagePhaseMapper._();

  static ComposerImagePhaseMapper? _instance;
  static ComposerImagePhaseMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ComposerImagePhaseMapper._());
      ImageQueuedMapper.ensureInitialized();
      ImageReadingMapper.ensureInitialized();
      ImagePreparingMapper.ensureInitialized();
      ImageUploadingMapper.ensureInitialized();
      ImageUploadedMapper.ensureInitialized();
      ImageFailedMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ComposerImagePhase';

  @override
  final MappableFields<ComposerImagePhase> fields = const {};

  static ComposerImagePhase _instantiate(DecodingData data) {
    throw MapperException.missingSubclass(
      'ComposerImagePhase',
      'type',
      '${data.value['type']}',
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ComposerImagePhase fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ComposerImagePhase>(map);
  }

  static ComposerImagePhase fromJson(String json) {
    return ensureInitialized().decodeJson<ComposerImagePhase>(json);
  }
}

mixin ComposerImagePhaseMappable {
  String toJson();
  Map<String, dynamic> toMap();
  ComposerImagePhaseCopyWith<
    ComposerImagePhase,
    ComposerImagePhase,
    ComposerImagePhase
  >
  get copyWith;
}

abstract class ComposerImagePhaseCopyWith<
  $R,
  $In extends ComposerImagePhase,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call();
  ComposerImagePhaseCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class ComposerImageNoticeMapper extends ClassMapperBase<ComposerImageNotice> {
  ComposerImageNoticeMapper._();

  static ComposerImageNoticeMapper? _instance;
  static ComposerImageNoticeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ComposerImageNoticeMapper._());
      ImageSelectionLimitNoticeMapper.ensureInitialized();
      UnsupportedImagesNoticeMapper.ensureInitialized();
      ImagePickerFailedNoticeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ComposerImageNotice';

  static int _$id(ComposerImageNotice v) => v.id;
  static const Field<ComposerImageNotice, int> _f$id = Field('id', _$id);

  @override
  final MappableFields<ComposerImageNotice> fields = const {#id: _f$id};

  static ComposerImageNotice _instantiate(DecodingData data) {
    throw MapperException.missingSubclass(
      'ComposerImageNotice',
      'type',
      '${data.value['type']}',
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ComposerImageNotice fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ComposerImageNotice>(map);
  }

  static ComposerImageNotice fromJson(String json) {
    return ensureInitialized().decodeJson<ComposerImageNotice>(json);
  }
}

mixin ComposerImageNoticeMappable {
  String toJson();
  Map<String, dynamic> toMap();
  ComposerImageNoticeCopyWith<
    ComposerImageNotice,
    ComposerImageNotice,
    ComposerImageNotice
  >
  get copyWith;
}

abstract class ComposerImageNoticeCopyWith<
  $R,
  $In extends ComposerImageNotice,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({int? id});
  ComposerImageNoticeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class ImageQueuedMapper extends SubClassMapperBase<ImageQueued> {
  ImageQueuedMapper._();

  static ImageQueuedMapper? _instance;
  static ImageQueuedMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageQueuedMapper._());
      ComposerImagePhaseMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImageQueued';

  @override
  final MappableFields<ImageQueued> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageQueued';
  @override
  late final ClassMapperBase superMapper =
      ComposerImagePhaseMapper.ensureInitialized();

  static ImageQueued _instantiate(DecodingData data) {
    return ImageQueued();
  }

  @override
  final Function instantiate = _instantiate;

  static ImageQueued fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageQueued>(map);
  }

  static ImageQueued fromJson(String json) {
    return ensureInitialized().decodeJson<ImageQueued>(json);
  }
}

mixin ImageQueuedMappable {
  String toJson() {
    return ImageQueuedMapper.ensureInitialized().encodeJson<ImageQueued>(
      this as ImageQueued,
    );
  }

  Map<String, dynamic> toMap() {
    return ImageQueuedMapper.ensureInitialized().encodeMap<ImageQueued>(
      this as ImageQueued,
    );
  }

  ImageQueuedCopyWith<ImageQueued, ImageQueued, ImageQueued> get copyWith =>
      _ImageQueuedCopyWithImpl<ImageQueued, ImageQueued>(
        this as ImageQueued,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ImageQueuedMapper.ensureInitialized().stringifyValue(
      this as ImageQueued,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageQueuedMapper.ensureInitialized().equalsValue(
      this as ImageQueued,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageQueuedMapper.ensureInitialized().hashValue(this as ImageQueued);
  }
}

extension ImageQueuedValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageQueued, $Out> {
  ImageQueuedCopyWith<$R, ImageQueued, $Out> get $asImageQueued =>
      $base.as((v, t, t2) => _ImageQueuedCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImageQueuedCopyWith<$R, $In extends ImageQueued, $Out>
    implements ComposerImagePhaseCopyWith<$R, $In, $Out> {
  @override
  $R call();
  ImageQueuedCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ImageQueuedCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageQueued, $Out>
    implements ImageQueuedCopyWith<$R, ImageQueued, $Out> {
  _ImageQueuedCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageQueued> $mapper =
      ImageQueuedMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  ImageQueued $make(CopyWithData data) => ImageQueued();

  @override
  ImageQueuedCopyWith<$R2, ImageQueued, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageQueuedCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageReadingMapper extends SubClassMapperBase<ImageReading> {
  ImageReadingMapper._();

  static ImageReadingMapper? _instance;
  static ImageReadingMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageReadingMapper._());
      ComposerImagePhaseMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImageReading';

  @override
  final MappableFields<ImageReading> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageReading';
  @override
  late final ClassMapperBase superMapper =
      ComposerImagePhaseMapper.ensureInitialized();

  static ImageReading _instantiate(DecodingData data) {
    return ImageReading();
  }

  @override
  final Function instantiate = _instantiate;

  static ImageReading fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageReading>(map);
  }

  static ImageReading fromJson(String json) {
    return ensureInitialized().decodeJson<ImageReading>(json);
  }
}

mixin ImageReadingMappable {
  String toJson() {
    return ImageReadingMapper.ensureInitialized().encodeJson<ImageReading>(
      this as ImageReading,
    );
  }

  Map<String, dynamic> toMap() {
    return ImageReadingMapper.ensureInitialized().encodeMap<ImageReading>(
      this as ImageReading,
    );
  }

  ImageReadingCopyWith<ImageReading, ImageReading, ImageReading> get copyWith =>
      _ImageReadingCopyWithImpl<ImageReading, ImageReading>(
        this as ImageReading,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ImageReadingMapper.ensureInitialized().stringifyValue(
      this as ImageReading,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageReadingMapper.ensureInitialized().equalsValue(
      this as ImageReading,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageReadingMapper.ensureInitialized().hashValue(
      this as ImageReading,
    );
  }
}

extension ImageReadingValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageReading, $Out> {
  ImageReadingCopyWith<$R, ImageReading, $Out> get $asImageReading =>
      $base.as((v, t, t2) => _ImageReadingCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImageReadingCopyWith<$R, $In extends ImageReading, $Out>
    implements ComposerImagePhaseCopyWith<$R, $In, $Out> {
  @override
  $R call();
  ImageReadingCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ImageReadingCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageReading, $Out>
    implements ImageReadingCopyWith<$R, ImageReading, $Out> {
  _ImageReadingCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageReading> $mapper =
      ImageReadingMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  ImageReading $make(CopyWithData data) => ImageReading();

  @override
  ImageReadingCopyWith<$R2, ImageReading, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageReadingCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImagePreparingMapper extends SubClassMapperBase<ImagePreparing> {
  ImagePreparingMapper._();

  static ImagePreparingMapper? _instance;
  static ImagePreparingMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImagePreparingMapper._());
      ComposerImagePhaseMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImagePreparing';

  @override
  final MappableFields<ImagePreparing> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImagePreparing';
  @override
  late final ClassMapperBase superMapper =
      ComposerImagePhaseMapper.ensureInitialized();

  static ImagePreparing _instantiate(DecodingData data) {
    return ImagePreparing();
  }

  @override
  final Function instantiate = _instantiate;

  static ImagePreparing fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImagePreparing>(map);
  }

  static ImagePreparing fromJson(String json) {
    return ensureInitialized().decodeJson<ImagePreparing>(json);
  }
}

mixin ImagePreparingMappable {
  String toJson() {
    return ImagePreparingMapper.ensureInitialized().encodeJson<ImagePreparing>(
      this as ImagePreparing,
    );
  }

  Map<String, dynamic> toMap() {
    return ImagePreparingMapper.ensureInitialized().encodeMap<ImagePreparing>(
      this as ImagePreparing,
    );
  }

  ImagePreparingCopyWith<ImagePreparing, ImagePreparing, ImagePreparing>
  get copyWith => _ImagePreparingCopyWithImpl<ImagePreparing, ImagePreparing>(
    this as ImagePreparing,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ImagePreparingMapper.ensureInitialized().stringifyValue(
      this as ImagePreparing,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImagePreparingMapper.ensureInitialized().equalsValue(
      this as ImagePreparing,
      other,
    );
  }

  @override
  int get hashCode {
    return ImagePreparingMapper.ensureInitialized().hashValue(
      this as ImagePreparing,
    );
  }
}

extension ImagePreparingValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImagePreparing, $Out> {
  ImagePreparingCopyWith<$R, ImagePreparing, $Out> get $asImagePreparing =>
      $base.as((v, t, t2) => _ImagePreparingCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImagePreparingCopyWith<$R, $In extends ImagePreparing, $Out>
    implements ComposerImagePhaseCopyWith<$R, $In, $Out> {
  @override
  $R call();
  ImagePreparingCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ImagePreparingCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImagePreparing, $Out>
    implements ImagePreparingCopyWith<$R, ImagePreparing, $Out> {
  _ImagePreparingCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImagePreparing> $mapper =
      ImagePreparingMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  ImagePreparing $make(CopyWithData data) => ImagePreparing();

  @override
  ImagePreparingCopyWith<$R2, ImagePreparing, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImagePreparingCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageUploadingMapper extends SubClassMapperBase<ImageUploading> {
  ImageUploadingMapper._();

  static ImageUploadingMapper? _instance;
  static ImageUploadingMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageUploadingMapper._());
      ComposerImagePhaseMapper.ensureInitialized().addSubMapper(_instance!);
      ImageTransferProgressMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ImageUploading';

  static ImageTransferProgress _$progress(ImageUploading v) => v.progress;
  static const Field<ImageUploading, ImageTransferProgress> _f$progress = Field(
    'progress',
    _$progress,
  );

  @override
  final MappableFields<ImageUploading> fields = const {#progress: _f$progress};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageUploading';
  @override
  late final ClassMapperBase superMapper =
      ComposerImagePhaseMapper.ensureInitialized();

  static ImageUploading _instantiate(DecodingData data) {
    return ImageUploading(data.dec(_f$progress));
  }

  @override
  final Function instantiate = _instantiate;

  static ImageUploading fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageUploading>(map);
  }

  static ImageUploading fromJson(String json) {
    return ensureInitialized().decodeJson<ImageUploading>(json);
  }
}

mixin ImageUploadingMappable {
  String toJson() {
    return ImageUploadingMapper.ensureInitialized().encodeJson<ImageUploading>(
      this as ImageUploading,
    );
  }

  Map<String, dynamic> toMap() {
    return ImageUploadingMapper.ensureInitialized().encodeMap<ImageUploading>(
      this as ImageUploading,
    );
  }

  ImageUploadingCopyWith<ImageUploading, ImageUploading, ImageUploading>
  get copyWith => _ImageUploadingCopyWithImpl<ImageUploading, ImageUploading>(
    this as ImageUploading,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ImageUploadingMapper.ensureInitialized().stringifyValue(
      this as ImageUploading,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageUploadingMapper.ensureInitialized().equalsValue(
      this as ImageUploading,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageUploadingMapper.ensureInitialized().hashValue(
      this as ImageUploading,
    );
  }
}

extension ImageUploadingValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageUploading, $Out> {
  ImageUploadingCopyWith<$R, ImageUploading, $Out> get $asImageUploading =>
      $base.as((v, t, t2) => _ImageUploadingCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImageUploadingCopyWith<$R, $In extends ImageUploading, $Out>
    implements ComposerImagePhaseCopyWith<$R, $In, $Out> {
  ImageTransferProgressCopyWith<
    $R,
    ImageTransferProgress,
    ImageTransferProgress
  >
  get progress;
  @override
  $R call({ImageTransferProgress? progress});
  ImageUploadingCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ImageUploadingCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageUploading, $Out>
    implements ImageUploadingCopyWith<$R, ImageUploading, $Out> {
  _ImageUploadingCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageUploading> $mapper =
      ImageUploadingMapper.ensureInitialized();
  @override
  ImageTransferProgressCopyWith<
    $R,
    ImageTransferProgress,
    ImageTransferProgress
  >
  get progress => $value.progress.copyWith.$chain((v) => call(progress: v));
  @override
  $R call({ImageTransferProgress? progress}) =>
      $apply(FieldCopyWithData({if (progress != null) #progress: progress}));
  @override
  ImageUploading $make(CopyWithData data) =>
      ImageUploading(data.get(#progress, or: $value.progress));

  @override
  ImageUploadingCopyWith<$R2, ImageUploading, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageUploadingCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageTransferProgressMapper
    extends ClassMapperBase<ImageTransferProgress> {
  ImageTransferProgressMapper._();

  static ImageTransferProgressMapper? _instance;
  static ImageTransferProgressMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageTransferProgressMapper._());
      TransferStartingMapper.ensureInitialized();
      TransferBytesMapper.ensureInitialized();
      TransferFinalizingMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ImageTransferProgress';

  @override
  final MappableFields<ImageTransferProgress> fields = const {};

  static ImageTransferProgress _instantiate(DecodingData data) {
    throw MapperException.missingSubclass(
      'ImageTransferProgress',
      'type',
      '${data.value['type']}',
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ImageTransferProgress fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageTransferProgress>(map);
  }

  static ImageTransferProgress fromJson(String json) {
    return ensureInitialized().decodeJson<ImageTransferProgress>(json);
  }
}

mixin ImageTransferProgressMappable {
  String toJson();
  Map<String, dynamic> toMap();
  ImageTransferProgressCopyWith<
    ImageTransferProgress,
    ImageTransferProgress,
    ImageTransferProgress
  >
  get copyWith;
}

abstract class ImageTransferProgressCopyWith<
  $R,
  $In extends ImageTransferProgress,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call();
  ImageTransferProgressCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class ImageUploadedMapper extends SubClassMapperBase<ImageUploaded> {
  ImageUploadedMapper._();

  static ImageUploadedMapper? _instance;
  static ImageUploadedMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageUploadedMapper._());
      ComposerImagePhaseMapper.ensureInitialized().addSubMapper(_instance!);
      UploadedDraftImageMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ImageUploaded';

  static UploadedDraftImage _$uploaded(ImageUploaded v) => v.uploaded;
  static const Field<ImageUploaded, UploadedDraftImage> _f$uploaded = Field(
    'uploaded',
    _$uploaded,
  );

  @override
  final MappableFields<ImageUploaded> fields = const {#uploaded: _f$uploaded};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageUploaded';
  @override
  late final ClassMapperBase superMapper =
      ComposerImagePhaseMapper.ensureInitialized();

  static ImageUploaded _instantiate(DecodingData data) {
    return ImageUploaded(data.dec(_f$uploaded));
  }

  @override
  final Function instantiate = _instantiate;

  static ImageUploaded fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageUploaded>(map);
  }

  static ImageUploaded fromJson(String json) {
    return ensureInitialized().decodeJson<ImageUploaded>(json);
  }
}

mixin ImageUploadedMappable {
  String toJson() {
    return ImageUploadedMapper.ensureInitialized().encodeJson<ImageUploaded>(
      this as ImageUploaded,
    );
  }

  Map<String, dynamic> toMap() {
    return ImageUploadedMapper.ensureInitialized().encodeMap<ImageUploaded>(
      this as ImageUploaded,
    );
  }

  ImageUploadedCopyWith<ImageUploaded, ImageUploaded, ImageUploaded>
  get copyWith => _ImageUploadedCopyWithImpl<ImageUploaded, ImageUploaded>(
    this as ImageUploaded,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ImageUploadedMapper.ensureInitialized().stringifyValue(
      this as ImageUploaded,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageUploadedMapper.ensureInitialized().equalsValue(
      this as ImageUploaded,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageUploadedMapper.ensureInitialized().hashValue(
      this as ImageUploaded,
    );
  }
}

extension ImageUploadedValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageUploaded, $Out> {
  ImageUploadedCopyWith<$R, ImageUploaded, $Out> get $asImageUploaded =>
      $base.as((v, t, t2) => _ImageUploadedCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImageUploadedCopyWith<$R, $In extends ImageUploaded, $Out>
    implements ComposerImagePhaseCopyWith<$R, $In, $Out> {
  UploadedDraftImageCopyWith<$R, UploadedDraftImage, UploadedDraftImage>
  get uploaded;
  @override
  $R call({UploadedDraftImage? uploaded});
  ImageUploadedCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ImageUploadedCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageUploaded, $Out>
    implements ImageUploadedCopyWith<$R, ImageUploaded, $Out> {
  _ImageUploadedCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageUploaded> $mapper =
      ImageUploadedMapper.ensureInitialized();
  @override
  UploadedDraftImageCopyWith<$R, UploadedDraftImage, UploadedDraftImage>
  get uploaded => $value.uploaded.copyWith.$chain((v) => call(uploaded: v));
  @override
  $R call({UploadedDraftImage? uploaded}) =>
      $apply(FieldCopyWithData({if (uploaded != null) #uploaded: uploaded}));
  @override
  ImageUploaded $make(CopyWithData data) =>
      ImageUploaded(data.get(#uploaded, or: $value.uploaded));

  @override
  ImageUploadedCopyWith<$R2, ImageUploaded, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageUploadedCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class UploadedDraftImageMapper extends ClassMapperBase<UploadedDraftImage> {
  UploadedDraftImageMapper._();

  static UploadedDraftImageMapper? _instance;
  static UploadedDraftImageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UploadedDraftImageMapper._());
      CreatePostImageAspectRatioMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'UploadedDraftImage';

  static String _$cid(UploadedDraftImage v) => v.cid;
  static const Field<UploadedDraftImage, String> _f$cid = Field('cid', _$cid);
  static String _$mime(UploadedDraftImage v) => v.mime;
  static const Field<UploadedDraftImage, String> _f$mime = Field(
    'mime',
    _$mime,
  );
  static int _$size(UploadedDraftImage v) => v.size;
  static const Field<UploadedDraftImage, int> _f$size = Field('size', _$size);
  static CreatePostImageAspectRatio? _$aspectRatio(UploadedDraftImage v) =>
      v.aspectRatio;
  static const Field<UploadedDraftImage, CreatePostImageAspectRatio>
  _f$aspectRatio = Field('aspectRatio', _$aspectRatio, opt: true);

  @override
  final MappableFields<UploadedDraftImage> fields = const {
    #cid: _f$cid,
    #mime: _f$mime,
    #size: _f$size,
    #aspectRatio: _f$aspectRatio,
  };

  static UploadedDraftImage _instantiate(DecodingData data) {
    return UploadedDraftImage(
      cid: data.dec(_f$cid),
      mime: data.dec(_f$mime),
      size: data.dec(_f$size),
      aspectRatio: data.dec(_f$aspectRatio),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UploadedDraftImage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UploadedDraftImage>(map);
  }

  static UploadedDraftImage fromJson(String json) {
    return ensureInitialized().decodeJson<UploadedDraftImage>(json);
  }
}

mixin UploadedDraftImageMappable {
  String toJson() {
    return UploadedDraftImageMapper.ensureInitialized()
        .encodeJson<UploadedDraftImage>(this as UploadedDraftImage);
  }

  Map<String, dynamic> toMap() {
    return UploadedDraftImageMapper.ensureInitialized()
        .encodeMap<UploadedDraftImage>(this as UploadedDraftImage);
  }

  UploadedDraftImageCopyWith<
    UploadedDraftImage,
    UploadedDraftImage,
    UploadedDraftImage
  >
  get copyWith =>
      _UploadedDraftImageCopyWithImpl<UploadedDraftImage, UploadedDraftImage>(
        this as UploadedDraftImage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return UploadedDraftImageMapper.ensureInitialized().stringifyValue(
      this as UploadedDraftImage,
    );
  }

  @override
  bool operator ==(Object other) {
    return UploadedDraftImageMapper.ensureInitialized().equalsValue(
      this as UploadedDraftImage,
      other,
    );
  }

  @override
  int get hashCode {
    return UploadedDraftImageMapper.ensureInitialized().hashValue(
      this as UploadedDraftImage,
    );
  }
}

extension UploadedDraftImageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UploadedDraftImage, $Out> {
  UploadedDraftImageCopyWith<$R, UploadedDraftImage, $Out>
  get $asUploadedDraftImage => $base.as(
    (v, t, t2) => _UploadedDraftImageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class UploadedDraftImageCopyWith<
  $R,
  $In extends UploadedDraftImage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  CreatePostImageAspectRatioCopyWith<
    $R,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >?
  get aspectRatio;
  $R call({
    String? cid,
    String? mime,
    int? size,
    CreatePostImageAspectRatio? aspectRatio,
  });
  UploadedDraftImageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UploadedDraftImageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UploadedDraftImage, $Out>
    implements UploadedDraftImageCopyWith<$R, UploadedDraftImage, $Out> {
  _UploadedDraftImageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UploadedDraftImage> $mapper =
      UploadedDraftImageMapper.ensureInitialized();
  @override
  CreatePostImageAspectRatioCopyWith<
    $R,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >?
  get aspectRatio =>
      $value.aspectRatio?.copyWith.$chain((v) => call(aspectRatio: v));
  @override
  $R call({
    String? cid,
    String? mime,
    int? size,
    Object? aspectRatio = $none,
  }) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (mime != null) #mime: mime,
      if (size != null) #size: size,
      if (aspectRatio != $none) #aspectRatio: aspectRatio,
    }),
  );
  @override
  UploadedDraftImage $make(CopyWithData data) => UploadedDraftImage(
    cid: data.get(#cid, or: $value.cid),
    mime: data.get(#mime, or: $value.mime),
    size: data.get(#size, or: $value.size),
    aspectRatio: data.get(#aspectRatio, or: $value.aspectRatio),
  );

  @override
  UploadedDraftImageCopyWith<$R2, UploadedDraftImage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _UploadedDraftImageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageFailedMapper extends SubClassMapperBase<ImageFailed> {
  ImageFailedMapper._();

  static ImageFailedMapper? _instance;
  static ImageFailedMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageFailedMapper._());
      ComposerImagePhaseMapper.ensureInitialized().addSubMapper(_instance!);
      ComposerImageFailureMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ImageFailed';

  static ComposerImageFailure _$failure(ImageFailed v) => v.failure;
  static const Field<ImageFailed, ComposerImageFailure> _f$failure = Field(
    'failure',
    _$failure,
  );

  @override
  final MappableFields<ImageFailed> fields = const {#failure: _f$failure};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageFailed';
  @override
  late final ClassMapperBase superMapper =
      ComposerImagePhaseMapper.ensureInitialized();

  static ImageFailed _instantiate(DecodingData data) {
    return ImageFailed(data.dec(_f$failure));
  }

  @override
  final Function instantiate = _instantiate;

  static ImageFailed fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageFailed>(map);
  }

  static ImageFailed fromJson(String json) {
    return ensureInitialized().decodeJson<ImageFailed>(json);
  }
}

mixin ImageFailedMappable {
  String toJson() {
    return ImageFailedMapper.ensureInitialized().encodeJson<ImageFailed>(
      this as ImageFailed,
    );
  }

  Map<String, dynamic> toMap() {
    return ImageFailedMapper.ensureInitialized().encodeMap<ImageFailed>(
      this as ImageFailed,
    );
  }

  ImageFailedCopyWith<ImageFailed, ImageFailed, ImageFailed> get copyWith =>
      _ImageFailedCopyWithImpl<ImageFailed, ImageFailed>(
        this as ImageFailed,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ImageFailedMapper.ensureInitialized().stringifyValue(
      this as ImageFailed,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageFailedMapper.ensureInitialized().equalsValue(
      this as ImageFailed,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageFailedMapper.ensureInitialized().hashValue(this as ImageFailed);
  }
}

extension ImageFailedValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageFailed, $Out> {
  ImageFailedCopyWith<$R, ImageFailed, $Out> get $asImageFailed =>
      $base.as((v, t, t2) => _ImageFailedCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImageFailedCopyWith<$R, $In extends ImageFailed, $Out>
    implements ComposerImagePhaseCopyWith<$R, $In, $Out> {
  ComposerImageFailureCopyWith<$R, ComposerImageFailure, ComposerImageFailure>
  get failure;
  @override
  $R call({ComposerImageFailure? failure});
  ImageFailedCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ImageFailedCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageFailed, $Out>
    implements ImageFailedCopyWith<$R, ImageFailed, $Out> {
  _ImageFailedCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageFailed> $mapper =
      ImageFailedMapper.ensureInitialized();
  @override
  ComposerImageFailureCopyWith<$R, ComposerImageFailure, ComposerImageFailure>
  get failure => $value.failure.copyWith.$chain((v) => call(failure: v));
  @override
  $R call({ComposerImageFailure? failure}) =>
      $apply(FieldCopyWithData({if (failure != null) #failure: failure}));
  @override
  ImageFailed $make(CopyWithData data) =>
      ImageFailed(data.get(#failure, or: $value.failure));

  @override
  ImageFailedCopyWith<$R2, ImageFailed, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageFailedCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ComposerImageFailureMapper extends ClassMapperBase<ComposerImageFailure> {
  ComposerImageFailureMapper._();

  static ComposerImageFailureMapper? _instance;
  static ComposerImageFailureMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ComposerImageFailureMapper._());
      UnsupportedImageTypeMapper.ensureInitialized();
      ImagePreparationFailedMapper.ensureInitialized();
      ImageTooLargeMapper.ensureInitialized();
      ImageUploadFailedMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ComposerImageFailure';

  @override
  final MappableFields<ComposerImageFailure> fields = const {};

  static ComposerImageFailure _instantiate(DecodingData data) {
    throw MapperException.missingSubclass(
      'ComposerImageFailure',
      'type',
      '${data.value['type']}',
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ComposerImageFailure fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ComposerImageFailure>(map);
  }

  static ComposerImageFailure fromJson(String json) {
    return ensureInitialized().decodeJson<ComposerImageFailure>(json);
  }
}

mixin ComposerImageFailureMappable {
  String toJson();
  Map<String, dynamic> toMap();
  ComposerImageFailureCopyWith<
    ComposerImageFailure,
    ComposerImageFailure,
    ComposerImageFailure
  >
  get copyWith;
}

abstract class ComposerImageFailureCopyWith<
  $R,
  $In extends ComposerImageFailure,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call();
  ComposerImageFailureCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class UnsupportedImageTypeMapper
    extends SubClassMapperBase<UnsupportedImageType> {
  UnsupportedImageTypeMapper._();

  static UnsupportedImageTypeMapper? _instance;
  static UnsupportedImageTypeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UnsupportedImageTypeMapper._());
      ComposerImageFailureMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'UnsupportedImageType';

  @override
  final MappableFields<UnsupportedImageType> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'UnsupportedImageType';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageFailureMapper.ensureInitialized();

  static UnsupportedImageType _instantiate(DecodingData data) {
    return UnsupportedImageType();
  }

  @override
  final Function instantiate = _instantiate;

  static UnsupportedImageType fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UnsupportedImageType>(map);
  }

  static UnsupportedImageType fromJson(String json) {
    return ensureInitialized().decodeJson<UnsupportedImageType>(json);
  }
}

mixin UnsupportedImageTypeMappable {
  String toJson() {
    return UnsupportedImageTypeMapper.ensureInitialized()
        .encodeJson<UnsupportedImageType>(this as UnsupportedImageType);
  }

  Map<String, dynamic> toMap() {
    return UnsupportedImageTypeMapper.ensureInitialized()
        .encodeMap<UnsupportedImageType>(this as UnsupportedImageType);
  }

  UnsupportedImageTypeCopyWith<
    UnsupportedImageType,
    UnsupportedImageType,
    UnsupportedImageType
  >
  get copyWith =>
      _UnsupportedImageTypeCopyWithImpl<
        UnsupportedImageType,
        UnsupportedImageType
      >(this as UnsupportedImageType, $identity, $identity);
  @override
  String toString() {
    return UnsupportedImageTypeMapper.ensureInitialized().stringifyValue(
      this as UnsupportedImageType,
    );
  }

  @override
  bool operator ==(Object other) {
    return UnsupportedImageTypeMapper.ensureInitialized().equalsValue(
      this as UnsupportedImageType,
      other,
    );
  }

  @override
  int get hashCode {
    return UnsupportedImageTypeMapper.ensureInitialized().hashValue(
      this as UnsupportedImageType,
    );
  }
}

extension UnsupportedImageTypeValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UnsupportedImageType, $Out> {
  UnsupportedImageTypeCopyWith<$R, UnsupportedImageType, $Out>
  get $asUnsupportedImageType => $base.as(
    (v, t, t2) => _UnsupportedImageTypeCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class UnsupportedImageTypeCopyWith<
  $R,
  $In extends UnsupportedImageType,
  $Out
>
    implements ComposerImageFailureCopyWith<$R, $In, $Out> {
  @override
  $R call();
  UnsupportedImageTypeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UnsupportedImageTypeCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UnsupportedImageType, $Out>
    implements UnsupportedImageTypeCopyWith<$R, UnsupportedImageType, $Out> {
  _UnsupportedImageTypeCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UnsupportedImageType> $mapper =
      UnsupportedImageTypeMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  UnsupportedImageType $make(CopyWithData data) => UnsupportedImageType();

  @override
  UnsupportedImageTypeCopyWith<$R2, UnsupportedImageType, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _UnsupportedImageTypeCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImagePreparationFailedMapper
    extends SubClassMapperBase<ImagePreparationFailed> {
  ImagePreparationFailedMapper._();

  static ImagePreparationFailedMapper? _instance;
  static ImagePreparationFailedMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImagePreparationFailedMapper._());
      ComposerImageFailureMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImagePreparationFailed';

  @override
  final MappableFields<ImagePreparationFailed> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImagePreparationFailed';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageFailureMapper.ensureInitialized();

  static ImagePreparationFailed _instantiate(DecodingData data) {
    return ImagePreparationFailed();
  }

  @override
  final Function instantiate = _instantiate;

  static ImagePreparationFailed fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImagePreparationFailed>(map);
  }

  static ImagePreparationFailed fromJson(String json) {
    return ensureInitialized().decodeJson<ImagePreparationFailed>(json);
  }
}

mixin ImagePreparationFailedMappable {
  String toJson() {
    return ImagePreparationFailedMapper.ensureInitialized()
        .encodeJson<ImagePreparationFailed>(this as ImagePreparationFailed);
  }

  Map<String, dynamic> toMap() {
    return ImagePreparationFailedMapper.ensureInitialized()
        .encodeMap<ImagePreparationFailed>(this as ImagePreparationFailed);
  }

  ImagePreparationFailedCopyWith<
    ImagePreparationFailed,
    ImagePreparationFailed,
    ImagePreparationFailed
  >
  get copyWith =>
      _ImagePreparationFailedCopyWithImpl<
        ImagePreparationFailed,
        ImagePreparationFailed
      >(this as ImagePreparationFailed, $identity, $identity);
  @override
  String toString() {
    return ImagePreparationFailedMapper.ensureInitialized().stringifyValue(
      this as ImagePreparationFailed,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImagePreparationFailedMapper.ensureInitialized().equalsValue(
      this as ImagePreparationFailed,
      other,
    );
  }

  @override
  int get hashCode {
    return ImagePreparationFailedMapper.ensureInitialized().hashValue(
      this as ImagePreparationFailed,
    );
  }
}

extension ImagePreparationFailedValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImagePreparationFailed, $Out> {
  ImagePreparationFailedCopyWith<$R, ImagePreparationFailed, $Out>
  get $asImagePreparationFailed => $base.as(
    (v, t, t2) => _ImagePreparationFailedCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ImagePreparationFailedCopyWith<
  $R,
  $In extends ImagePreparationFailed,
  $Out
>
    implements ComposerImageFailureCopyWith<$R, $In, $Out> {
  @override
  $R call();
  ImagePreparationFailedCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ImagePreparationFailedCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImagePreparationFailed, $Out>
    implements
        ImagePreparationFailedCopyWith<$R, ImagePreparationFailed, $Out> {
  _ImagePreparationFailedCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImagePreparationFailed> $mapper =
      ImagePreparationFailedMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  ImagePreparationFailed $make(CopyWithData data) => ImagePreparationFailed();

  @override
  ImagePreparationFailedCopyWith<$R2, ImagePreparationFailed, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ImagePreparationFailedCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageTooLargeMapper extends SubClassMapperBase<ImageTooLarge> {
  ImageTooLargeMapper._();

  static ImageTooLargeMapper? _instance;
  static ImageTooLargeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageTooLargeMapper._());
      ComposerImageFailureMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImageTooLarge';

  static int _$maxBytes(ImageTooLarge v) => v.maxBytes;
  static const Field<ImageTooLarge, int> _f$maxBytes = Field(
    'maxBytes',
    _$maxBytes,
  );

  @override
  final MappableFields<ImageTooLarge> fields = const {#maxBytes: _f$maxBytes};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageTooLarge';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageFailureMapper.ensureInitialized();

  static ImageTooLarge _instantiate(DecodingData data) {
    return ImageTooLarge(data.dec(_f$maxBytes));
  }

  @override
  final Function instantiate = _instantiate;

  static ImageTooLarge fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageTooLarge>(map);
  }

  static ImageTooLarge fromJson(String json) {
    return ensureInitialized().decodeJson<ImageTooLarge>(json);
  }
}

mixin ImageTooLargeMappable {
  String toJson() {
    return ImageTooLargeMapper.ensureInitialized().encodeJson<ImageTooLarge>(
      this as ImageTooLarge,
    );
  }

  Map<String, dynamic> toMap() {
    return ImageTooLargeMapper.ensureInitialized().encodeMap<ImageTooLarge>(
      this as ImageTooLarge,
    );
  }

  ImageTooLargeCopyWith<ImageTooLarge, ImageTooLarge, ImageTooLarge>
  get copyWith => _ImageTooLargeCopyWithImpl<ImageTooLarge, ImageTooLarge>(
    this as ImageTooLarge,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ImageTooLargeMapper.ensureInitialized().stringifyValue(
      this as ImageTooLarge,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageTooLargeMapper.ensureInitialized().equalsValue(
      this as ImageTooLarge,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageTooLargeMapper.ensureInitialized().hashValue(
      this as ImageTooLarge,
    );
  }
}

extension ImageTooLargeValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageTooLarge, $Out> {
  ImageTooLargeCopyWith<$R, ImageTooLarge, $Out> get $asImageTooLarge =>
      $base.as((v, t, t2) => _ImageTooLargeCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ImageTooLargeCopyWith<$R, $In extends ImageTooLarge, $Out>
    implements ComposerImageFailureCopyWith<$R, $In, $Out> {
  @override
  $R call({int? maxBytes});
  ImageTooLargeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ImageTooLargeCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageTooLarge, $Out>
    implements ImageTooLargeCopyWith<$R, ImageTooLarge, $Out> {
  _ImageTooLargeCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageTooLarge> $mapper =
      ImageTooLargeMapper.ensureInitialized();
  @override
  $R call({int? maxBytes}) =>
      $apply(FieldCopyWithData({if (maxBytes != null) #maxBytes: maxBytes}));
  @override
  ImageTooLarge $make(CopyWithData data) =>
      ImageTooLarge(data.get(#maxBytes, or: $value.maxBytes));

  @override
  ImageTooLargeCopyWith<$R2, ImageTooLarge, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageTooLargeCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageUploadFailedMapper extends SubClassMapperBase<ImageUploadFailed> {
  ImageUploadFailedMapper._();

  static ImageUploadFailedMapper? _instance;
  static ImageUploadFailedMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ImageUploadFailedMapper._());
      ComposerImageFailureMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImageUploadFailed';

  @override
  final MappableFields<ImageUploadFailed> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageUploadFailed';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageFailureMapper.ensureInitialized();

  static ImageUploadFailed _instantiate(DecodingData data) {
    return ImageUploadFailed();
  }

  @override
  final Function instantiate = _instantiate;

  static ImageUploadFailed fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageUploadFailed>(map);
  }

  static ImageUploadFailed fromJson(String json) {
    return ensureInitialized().decodeJson<ImageUploadFailed>(json);
  }
}

mixin ImageUploadFailedMappable {
  String toJson() {
    return ImageUploadFailedMapper.ensureInitialized()
        .encodeJson<ImageUploadFailed>(this as ImageUploadFailed);
  }

  Map<String, dynamic> toMap() {
    return ImageUploadFailedMapper.ensureInitialized()
        .encodeMap<ImageUploadFailed>(this as ImageUploadFailed);
  }

  ImageUploadFailedCopyWith<
    ImageUploadFailed,
    ImageUploadFailed,
    ImageUploadFailed
  >
  get copyWith =>
      _ImageUploadFailedCopyWithImpl<ImageUploadFailed, ImageUploadFailed>(
        this as ImageUploadFailed,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ImageUploadFailedMapper.ensureInitialized().stringifyValue(
      this as ImageUploadFailed,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageUploadFailedMapper.ensureInitialized().equalsValue(
      this as ImageUploadFailed,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageUploadFailedMapper.ensureInitialized().hashValue(
      this as ImageUploadFailed,
    );
  }
}

extension ImageUploadFailedValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageUploadFailed, $Out> {
  ImageUploadFailedCopyWith<$R, ImageUploadFailed, $Out>
  get $asImageUploadFailed => $base.as(
    (v, t, t2) => _ImageUploadFailedCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ImageUploadFailedCopyWith<
  $R,
  $In extends ImageUploadFailed,
  $Out
>
    implements ComposerImageFailureCopyWith<$R, $In, $Out> {
  @override
  $R call();
  ImageUploadFailedCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ImageUploadFailedCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageUploadFailed, $Out>
    implements ImageUploadFailedCopyWith<$R, ImageUploadFailed, $Out> {
  _ImageUploadFailedCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageUploadFailed> $mapper =
      ImageUploadFailedMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  ImageUploadFailed $make(CopyWithData data) => ImageUploadFailed();

  @override
  ImageUploadFailedCopyWith<$R2, ImageUploadFailed, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ImageUploadFailedCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TransferStartingMapper extends SubClassMapperBase<TransferStarting> {
  TransferStartingMapper._();

  static TransferStartingMapper? _instance;
  static TransferStartingMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TransferStartingMapper._());
      ImageTransferProgressMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'TransferStarting';

  @override
  final MappableFields<TransferStarting> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'TransferStarting';
  @override
  late final ClassMapperBase superMapper =
      ImageTransferProgressMapper.ensureInitialized();

  static TransferStarting _instantiate(DecodingData data) {
    return TransferStarting();
  }

  @override
  final Function instantiate = _instantiate;

  static TransferStarting fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TransferStarting>(map);
  }

  static TransferStarting fromJson(String json) {
    return ensureInitialized().decodeJson<TransferStarting>(json);
  }
}

mixin TransferStartingMappable {
  String toJson() {
    return TransferStartingMapper.ensureInitialized()
        .encodeJson<TransferStarting>(this as TransferStarting);
  }

  Map<String, dynamic> toMap() {
    return TransferStartingMapper.ensureInitialized()
        .encodeMap<TransferStarting>(this as TransferStarting);
  }

  TransferStartingCopyWith<TransferStarting, TransferStarting, TransferStarting>
  get copyWith =>
      _TransferStartingCopyWithImpl<TransferStarting, TransferStarting>(
        this as TransferStarting,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return TransferStartingMapper.ensureInitialized().stringifyValue(
      this as TransferStarting,
    );
  }

  @override
  bool operator ==(Object other) {
    return TransferStartingMapper.ensureInitialized().equalsValue(
      this as TransferStarting,
      other,
    );
  }

  @override
  int get hashCode {
    return TransferStartingMapper.ensureInitialized().hashValue(
      this as TransferStarting,
    );
  }
}

extension TransferStartingValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TransferStarting, $Out> {
  TransferStartingCopyWith<$R, TransferStarting, $Out>
  get $asTransferStarting =>
      $base.as((v, t, t2) => _TransferStartingCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TransferStartingCopyWith<$R, $In extends TransferStarting, $Out>
    implements ImageTransferProgressCopyWith<$R, $In, $Out> {
  @override
  $R call();
  TransferStartingCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _TransferStartingCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TransferStarting, $Out>
    implements TransferStartingCopyWith<$R, TransferStarting, $Out> {
  _TransferStartingCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TransferStarting> $mapper =
      TransferStartingMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  TransferStarting $make(CopyWithData data) => TransferStarting();

  @override
  TransferStartingCopyWith<$R2, TransferStarting, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TransferStartingCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TransferBytesMapper extends SubClassMapperBase<TransferBytes> {
  TransferBytesMapper._();

  static TransferBytesMapper? _instance;
  static TransferBytesMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TransferBytesMapper._());
      ImageTransferProgressMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'TransferBytes';

  static int _$sent(TransferBytes v) => v.sent;
  static const Field<TransferBytes, int> _f$sent = Field('sent', _$sent);
  static int _$sendTotal(TransferBytes v) => v.sendTotal;
  static const Field<TransferBytes, int> _f$sendTotal = Field(
    'sendTotal',
    _$sendTotal,
  );
  static int _$received(TransferBytes v) => v.received;
  static const Field<TransferBytes, int> _f$received = Field(
    'received',
    _$received,
  );
  static int _$receiveTotal(TransferBytes v) => v.receiveTotal;
  static const Field<TransferBytes, int> _f$receiveTotal = Field(
    'receiveTotal',
    _$receiveTotal,
  );

  @override
  final MappableFields<TransferBytes> fields = const {
    #sent: _f$sent,
    #sendTotal: _f$sendTotal,
    #received: _f$received,
    #receiveTotal: _f$receiveTotal,
  };

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'TransferBytes';
  @override
  late final ClassMapperBase superMapper =
      ImageTransferProgressMapper.ensureInitialized();

  static TransferBytes _instantiate(DecodingData data) {
    return TransferBytes(
      sent: data.dec(_f$sent),
      sendTotal: data.dec(_f$sendTotal),
      received: data.dec(_f$received),
      receiveTotal: data.dec(_f$receiveTotal),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static TransferBytes fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TransferBytes>(map);
  }

  static TransferBytes fromJson(String json) {
    return ensureInitialized().decodeJson<TransferBytes>(json);
  }
}

mixin TransferBytesMappable {
  String toJson() {
    return TransferBytesMapper.ensureInitialized().encodeJson<TransferBytes>(
      this as TransferBytes,
    );
  }

  Map<String, dynamic> toMap() {
    return TransferBytesMapper.ensureInitialized().encodeMap<TransferBytes>(
      this as TransferBytes,
    );
  }

  TransferBytesCopyWith<TransferBytes, TransferBytes, TransferBytes>
  get copyWith => _TransferBytesCopyWithImpl<TransferBytes, TransferBytes>(
    this as TransferBytes,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return TransferBytesMapper.ensureInitialized().stringifyValue(
      this as TransferBytes,
    );
  }

  @override
  bool operator ==(Object other) {
    return TransferBytesMapper.ensureInitialized().equalsValue(
      this as TransferBytes,
      other,
    );
  }

  @override
  int get hashCode {
    return TransferBytesMapper.ensureInitialized().hashValue(
      this as TransferBytes,
    );
  }
}

extension TransferBytesValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TransferBytes, $Out> {
  TransferBytesCopyWith<$R, TransferBytes, $Out> get $asTransferBytes =>
      $base.as((v, t, t2) => _TransferBytesCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TransferBytesCopyWith<$R, $In extends TransferBytes, $Out>
    implements ImageTransferProgressCopyWith<$R, $In, $Out> {
  @override
  $R call({int? sent, int? sendTotal, int? received, int? receiveTotal});
  TransferBytesCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _TransferBytesCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TransferBytes, $Out>
    implements TransferBytesCopyWith<$R, TransferBytes, $Out> {
  _TransferBytesCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TransferBytes> $mapper =
      TransferBytesMapper.ensureInitialized();
  @override
  $R call({int? sent, int? sendTotal, int? received, int? receiveTotal}) =>
      $apply(
        FieldCopyWithData({
          if (sent != null) #sent: sent,
          if (sendTotal != null) #sendTotal: sendTotal,
          if (received != null) #received: received,
          if (receiveTotal != null) #receiveTotal: receiveTotal,
        }),
      );
  @override
  TransferBytes $make(CopyWithData data) => TransferBytes(
    sent: data.get(#sent, or: $value.sent),
    sendTotal: data.get(#sendTotal, or: $value.sendTotal),
    received: data.get(#received, or: $value.received),
    receiveTotal: data.get(#receiveTotal, or: $value.receiveTotal),
  );

  @override
  TransferBytesCopyWith<$R2, TransferBytes, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TransferBytesCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TransferFinalizingMapper extends SubClassMapperBase<TransferFinalizing> {
  TransferFinalizingMapper._();

  static TransferFinalizingMapper? _instance;
  static TransferFinalizingMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TransferFinalizingMapper._());
      ImageTransferProgressMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'TransferFinalizing';

  @override
  final MappableFields<TransferFinalizing> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'TransferFinalizing';
  @override
  late final ClassMapperBase superMapper =
      ImageTransferProgressMapper.ensureInitialized();

  static TransferFinalizing _instantiate(DecodingData data) {
    return TransferFinalizing();
  }

  @override
  final Function instantiate = _instantiate;

  static TransferFinalizing fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TransferFinalizing>(map);
  }

  static TransferFinalizing fromJson(String json) {
    return ensureInitialized().decodeJson<TransferFinalizing>(json);
  }
}

mixin TransferFinalizingMappable {
  String toJson() {
    return TransferFinalizingMapper.ensureInitialized()
        .encodeJson<TransferFinalizing>(this as TransferFinalizing);
  }

  Map<String, dynamic> toMap() {
    return TransferFinalizingMapper.ensureInitialized()
        .encodeMap<TransferFinalizing>(this as TransferFinalizing);
  }

  TransferFinalizingCopyWith<
    TransferFinalizing,
    TransferFinalizing,
    TransferFinalizing
  >
  get copyWith =>
      _TransferFinalizingCopyWithImpl<TransferFinalizing, TransferFinalizing>(
        this as TransferFinalizing,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return TransferFinalizingMapper.ensureInitialized().stringifyValue(
      this as TransferFinalizing,
    );
  }

  @override
  bool operator ==(Object other) {
    return TransferFinalizingMapper.ensureInitialized().equalsValue(
      this as TransferFinalizing,
      other,
    );
  }

  @override
  int get hashCode {
    return TransferFinalizingMapper.ensureInitialized().hashValue(
      this as TransferFinalizing,
    );
  }
}

extension TransferFinalizingValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TransferFinalizing, $Out> {
  TransferFinalizingCopyWith<$R, TransferFinalizing, $Out>
  get $asTransferFinalizing => $base.as(
    (v, t, t2) => _TransferFinalizingCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class TransferFinalizingCopyWith<
  $R,
  $In extends TransferFinalizing,
  $Out
>
    implements ImageTransferProgressCopyWith<$R, $In, $Out> {
  @override
  $R call();
  TransferFinalizingCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _TransferFinalizingCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TransferFinalizing, $Out>
    implements TransferFinalizingCopyWith<$R, TransferFinalizing, $Out> {
  _TransferFinalizingCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TransferFinalizing> $mapper =
      TransferFinalizingMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  TransferFinalizing $make(CopyWithData data) => TransferFinalizing();

  @override
  TransferFinalizingCopyWith<$R2, TransferFinalizing, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TransferFinalizingCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImageSelectionLimitNoticeMapper
    extends SubClassMapperBase<ImageSelectionLimitNotice> {
  ImageSelectionLimitNoticeMapper._();

  static ImageSelectionLimitNoticeMapper? _instance;
  static ImageSelectionLimitNoticeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ImageSelectionLimitNoticeMapper._(),
      );
      ComposerImageNoticeMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImageSelectionLimitNotice';

  static int _$id(ImageSelectionLimitNotice v) => v.id;
  static const Field<ImageSelectionLimitNotice, int> _f$id = Field('id', _$id);
  static int _$maxImages(ImageSelectionLimitNotice v) => v.maxImages;
  static const Field<ImageSelectionLimitNotice, int> _f$maxImages = Field(
    'maxImages',
    _$maxImages,
  );
  static int _$acceptedCount(ImageSelectionLimitNotice v) => v.acceptedCount;
  static const Field<ImageSelectionLimitNotice, int> _f$acceptedCount = Field(
    'acceptedCount',
    _$acceptedCount,
  );

  @override
  final MappableFields<ImageSelectionLimitNotice> fields = const {
    #id: _f$id,
    #maxImages: _f$maxImages,
    #acceptedCount: _f$acceptedCount,
  };

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImageSelectionLimitNotice';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageNoticeMapper.ensureInitialized();

  static ImageSelectionLimitNotice _instantiate(DecodingData data) {
    return ImageSelectionLimitNotice(
      id: data.dec(_f$id),
      maxImages: data.dec(_f$maxImages),
      acceptedCount: data.dec(_f$acceptedCount),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ImageSelectionLimitNotice fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImageSelectionLimitNotice>(map);
  }

  static ImageSelectionLimitNotice fromJson(String json) {
    return ensureInitialized().decodeJson<ImageSelectionLimitNotice>(json);
  }
}

mixin ImageSelectionLimitNoticeMappable {
  String toJson() {
    return ImageSelectionLimitNoticeMapper.ensureInitialized()
        .encodeJson<ImageSelectionLimitNotice>(
          this as ImageSelectionLimitNotice,
        );
  }

  Map<String, dynamic> toMap() {
    return ImageSelectionLimitNoticeMapper.ensureInitialized()
        .encodeMap<ImageSelectionLimitNotice>(
          this as ImageSelectionLimitNotice,
        );
  }

  ImageSelectionLimitNoticeCopyWith<
    ImageSelectionLimitNotice,
    ImageSelectionLimitNotice,
    ImageSelectionLimitNotice
  >
  get copyWith =>
      _ImageSelectionLimitNoticeCopyWithImpl<
        ImageSelectionLimitNotice,
        ImageSelectionLimitNotice
      >(this as ImageSelectionLimitNotice, $identity, $identity);
  @override
  String toString() {
    return ImageSelectionLimitNoticeMapper.ensureInitialized().stringifyValue(
      this as ImageSelectionLimitNotice,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImageSelectionLimitNoticeMapper.ensureInitialized().equalsValue(
      this as ImageSelectionLimitNotice,
      other,
    );
  }

  @override
  int get hashCode {
    return ImageSelectionLimitNoticeMapper.ensureInitialized().hashValue(
      this as ImageSelectionLimitNotice,
    );
  }
}

extension ImageSelectionLimitNoticeValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImageSelectionLimitNotice, $Out> {
  ImageSelectionLimitNoticeCopyWith<$R, ImageSelectionLimitNotice, $Out>
  get $asImageSelectionLimitNotice => $base.as(
    (v, t, t2) => _ImageSelectionLimitNoticeCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ImageSelectionLimitNoticeCopyWith<
  $R,
  $In extends ImageSelectionLimitNotice,
  $Out
>
    implements ComposerImageNoticeCopyWith<$R, $In, $Out> {
  @override
  $R call({int? id, int? maxImages, int? acceptedCount});
  ImageSelectionLimitNoticeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ImageSelectionLimitNoticeCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImageSelectionLimitNotice, $Out>
    implements
        ImageSelectionLimitNoticeCopyWith<$R, ImageSelectionLimitNotice, $Out> {
  _ImageSelectionLimitNoticeCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImageSelectionLimitNotice> $mapper =
      ImageSelectionLimitNoticeMapper.ensureInitialized();
  @override
  $R call({int? id, int? maxImages, int? acceptedCount}) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (maxImages != null) #maxImages: maxImages,
      if (acceptedCount != null) #acceptedCount: acceptedCount,
    }),
  );
  @override
  ImageSelectionLimitNotice $make(CopyWithData data) =>
      ImageSelectionLimitNotice(
        id: data.get(#id, or: $value.id),
        maxImages: data.get(#maxImages, or: $value.maxImages),
        acceptedCount: data.get(#acceptedCount, or: $value.acceptedCount),
      );

  @override
  ImageSelectionLimitNoticeCopyWith<$R2, ImageSelectionLimitNotice, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ImageSelectionLimitNoticeCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class UnsupportedImagesNoticeMapper
    extends SubClassMapperBase<UnsupportedImagesNotice> {
  UnsupportedImagesNoticeMapper._();

  static UnsupportedImagesNoticeMapper? _instance;
  static UnsupportedImagesNoticeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = UnsupportedImagesNoticeMapper._(),
      );
      ComposerImageNoticeMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'UnsupportedImagesNotice';

  static int _$id(UnsupportedImagesNotice v) => v.id;
  static const Field<UnsupportedImagesNotice, int> _f$id = Field('id', _$id);
  static int _$count(UnsupportedImagesNotice v) => v.count;
  static const Field<UnsupportedImagesNotice, int> _f$count = Field(
    'count',
    _$count,
  );

  @override
  final MappableFields<UnsupportedImagesNotice> fields = const {
    #id: _f$id,
    #count: _f$count,
  };

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'UnsupportedImagesNotice';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageNoticeMapper.ensureInitialized();

  static UnsupportedImagesNotice _instantiate(DecodingData data) {
    return UnsupportedImagesNotice(
      id: data.dec(_f$id),
      count: data.dec(_f$count),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UnsupportedImagesNotice fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UnsupportedImagesNotice>(map);
  }

  static UnsupportedImagesNotice fromJson(String json) {
    return ensureInitialized().decodeJson<UnsupportedImagesNotice>(json);
  }
}

mixin UnsupportedImagesNoticeMappable {
  String toJson() {
    return UnsupportedImagesNoticeMapper.ensureInitialized()
        .encodeJson<UnsupportedImagesNotice>(this as UnsupportedImagesNotice);
  }

  Map<String, dynamic> toMap() {
    return UnsupportedImagesNoticeMapper.ensureInitialized()
        .encodeMap<UnsupportedImagesNotice>(this as UnsupportedImagesNotice);
  }

  UnsupportedImagesNoticeCopyWith<
    UnsupportedImagesNotice,
    UnsupportedImagesNotice,
    UnsupportedImagesNotice
  >
  get copyWith =>
      _UnsupportedImagesNoticeCopyWithImpl<
        UnsupportedImagesNotice,
        UnsupportedImagesNotice
      >(this as UnsupportedImagesNotice, $identity, $identity);
  @override
  String toString() {
    return UnsupportedImagesNoticeMapper.ensureInitialized().stringifyValue(
      this as UnsupportedImagesNotice,
    );
  }

  @override
  bool operator ==(Object other) {
    return UnsupportedImagesNoticeMapper.ensureInitialized().equalsValue(
      this as UnsupportedImagesNotice,
      other,
    );
  }

  @override
  int get hashCode {
    return UnsupportedImagesNoticeMapper.ensureInitialized().hashValue(
      this as UnsupportedImagesNotice,
    );
  }
}

extension UnsupportedImagesNoticeValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UnsupportedImagesNotice, $Out> {
  UnsupportedImagesNoticeCopyWith<$R, UnsupportedImagesNotice, $Out>
  get $asUnsupportedImagesNotice => $base.as(
    (v, t, t2) => _UnsupportedImagesNoticeCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class UnsupportedImagesNoticeCopyWith<
  $R,
  $In extends UnsupportedImagesNotice,
  $Out
>
    implements ComposerImageNoticeCopyWith<$R, $In, $Out> {
  @override
  $R call({int? id, int? count});
  UnsupportedImagesNoticeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UnsupportedImagesNoticeCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UnsupportedImagesNotice, $Out>
    implements
        UnsupportedImagesNoticeCopyWith<$R, UnsupportedImagesNotice, $Out> {
  _UnsupportedImagesNoticeCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UnsupportedImagesNotice> $mapper =
      UnsupportedImagesNoticeMapper.ensureInitialized();
  @override
  $R call({int? id, int? count}) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (count != null) #count: count,
    }),
  );
  @override
  UnsupportedImagesNotice $make(CopyWithData data) => UnsupportedImagesNotice(
    id: data.get(#id, or: $value.id),
    count: data.get(#count, or: $value.count),
  );

  @override
  UnsupportedImagesNoticeCopyWith<$R2, UnsupportedImagesNotice, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _UnsupportedImagesNoticeCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ImagePickerFailedNoticeMapper
    extends SubClassMapperBase<ImagePickerFailedNotice> {
  ImagePickerFailedNoticeMapper._();

  static ImagePickerFailedNoticeMapper? _instance;
  static ImagePickerFailedNoticeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ImagePickerFailedNoticeMapper._(),
      );
      ComposerImageNoticeMapper.ensureInitialized().addSubMapper(_instance!);
    }
    return _instance!;
  }

  @override
  final String id = 'ImagePickerFailedNotice';

  static int _$id(ImagePickerFailedNotice v) => v.id;
  static const Field<ImagePickerFailedNotice, int> _f$id = Field('id', _$id);

  @override
  final MappableFields<ImagePickerFailedNotice> fields = const {#id: _f$id};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ImagePickerFailedNotice';
  @override
  late final ClassMapperBase superMapper =
      ComposerImageNoticeMapper.ensureInitialized();

  static ImagePickerFailedNotice _instantiate(DecodingData data) {
    return ImagePickerFailedNotice(id: data.dec(_f$id));
  }

  @override
  final Function instantiate = _instantiate;

  static ImagePickerFailedNotice fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ImagePickerFailedNotice>(map);
  }

  static ImagePickerFailedNotice fromJson(String json) {
    return ensureInitialized().decodeJson<ImagePickerFailedNotice>(json);
  }
}

mixin ImagePickerFailedNoticeMappable {
  String toJson() {
    return ImagePickerFailedNoticeMapper.ensureInitialized()
        .encodeJson<ImagePickerFailedNotice>(this as ImagePickerFailedNotice);
  }

  Map<String, dynamic> toMap() {
    return ImagePickerFailedNoticeMapper.ensureInitialized()
        .encodeMap<ImagePickerFailedNotice>(this as ImagePickerFailedNotice);
  }

  ImagePickerFailedNoticeCopyWith<
    ImagePickerFailedNotice,
    ImagePickerFailedNotice,
    ImagePickerFailedNotice
  >
  get copyWith =>
      _ImagePickerFailedNoticeCopyWithImpl<
        ImagePickerFailedNotice,
        ImagePickerFailedNotice
      >(this as ImagePickerFailedNotice, $identity, $identity);
  @override
  String toString() {
    return ImagePickerFailedNoticeMapper.ensureInitialized().stringifyValue(
      this as ImagePickerFailedNotice,
    );
  }

  @override
  bool operator ==(Object other) {
    return ImagePickerFailedNoticeMapper.ensureInitialized().equalsValue(
      this as ImagePickerFailedNotice,
      other,
    );
  }

  @override
  int get hashCode {
    return ImagePickerFailedNoticeMapper.ensureInitialized().hashValue(
      this as ImagePickerFailedNotice,
    );
  }
}

extension ImagePickerFailedNoticeValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ImagePickerFailedNotice, $Out> {
  ImagePickerFailedNoticeCopyWith<$R, ImagePickerFailedNotice, $Out>
  get $asImagePickerFailedNotice => $base.as(
    (v, t, t2) => _ImagePickerFailedNoticeCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ImagePickerFailedNoticeCopyWith<
  $R,
  $In extends ImagePickerFailedNotice,
  $Out
>
    implements ComposerImageNoticeCopyWith<$R, $In, $Out> {
  @override
  $R call({int? id});
  ImagePickerFailedNoticeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ImagePickerFailedNoticeCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ImagePickerFailedNotice, $Out>
    implements
        ImagePickerFailedNoticeCopyWith<$R, ImagePickerFailedNotice, $Out> {
  _ImagePickerFailedNoticeCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ImagePickerFailedNotice> $mapper =
      ImagePickerFailedNoticeMapper.ensureInitialized();
  @override
  $R call({int? id}) => $apply(FieldCopyWithData({if (id != null) #id: id}));
  @override
  ImagePickerFailedNotice $make(CopyWithData data) =>
      ImagePickerFailedNotice(id: data.get(#id, or: $value.id));

  @override
  ImagePickerFailedNoticeCopyWith<$R2, ImagePickerFailedNotice, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ImagePickerFailedNoticeCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
