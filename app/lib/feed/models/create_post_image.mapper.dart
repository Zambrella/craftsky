// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'create_post_image.dart';

class CreatePostImageMapper extends ClassMapperBase<CreatePostImage> {
  CreatePostImageMapper._();

  static CreatePostImageMapper? _instance;
  static CreatePostImageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CreatePostImageMapper._());
      CreatePostBlobMapper.ensureInitialized();
      CreatePostImageAspectRatioMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'CreatePostImage';

  static CreatePostBlob _$blob(CreatePostImage v) => v.blob;
  static const Field<CreatePostImage, CreatePostBlob> _f$blob = Field(
    'blob',
    _$blob,
    key: r'image',
  );
  static String _$alt(CreatePostImage v) => v.alt;
  static const Field<CreatePostImage, String> _f$alt = Field(
    'alt',
    _$alt,
    opt: true,
    def: '',
  );
  static CreatePostImageAspectRatio? _$aspectRatio(CreatePostImage v) =>
      v.aspectRatio;
  static const Field<CreatePostImage, CreatePostImageAspectRatio>
  _f$aspectRatio = Field('aspectRatio', _$aspectRatio, opt: true);

  @override
  final MappableFields<CreatePostImage> fields = const {
    #blob: _f$blob,
    #alt: _f$alt,
    #aspectRatio: _f$aspectRatio,
  };
  @override
  final bool ignoreNull = true;

  static CreatePostImage _instantiate(DecodingData data) {
    return CreatePostImage(
      blob: data.dec(_f$blob),
      alt: data.dec(_f$alt),
      aspectRatio: data.dec(_f$aspectRatio),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static CreatePostImage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CreatePostImage>(map);
  }

  static CreatePostImage fromJson(String json) {
    return ensureInitialized().decodeJson<CreatePostImage>(json);
  }
}

mixin CreatePostImageMappable {
  String toJson() {
    return CreatePostImageMapper.ensureInitialized()
        .encodeJson<CreatePostImage>(this as CreatePostImage);
  }

  Map<String, dynamic> toMap() {
    return CreatePostImageMapper.ensureInitialized().encodeMap<CreatePostImage>(
      this as CreatePostImage,
    );
  }

  CreatePostImageCopyWith<CreatePostImage, CreatePostImage, CreatePostImage>
  get copyWith =>
      _CreatePostImageCopyWithImpl<CreatePostImage, CreatePostImage>(
        this as CreatePostImage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return CreatePostImageMapper.ensureInitialized().stringifyValue(
      this as CreatePostImage,
    );
  }

  @override
  bool operator ==(Object other) {
    return CreatePostImageMapper.ensureInitialized().equalsValue(
      this as CreatePostImage,
      other,
    );
  }

  @override
  int get hashCode {
    return CreatePostImageMapper.ensureInitialized().hashValue(
      this as CreatePostImage,
    );
  }
}

extension CreatePostImageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CreatePostImage, $Out> {
  CreatePostImageCopyWith<$R, CreatePostImage, $Out> get $asCreatePostImage =>
      $base.as((v, t, t2) => _CreatePostImageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class CreatePostImageCopyWith<$R, $In extends CreatePostImage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  CreatePostBlobCopyWith<$R, CreatePostBlob, CreatePostBlob> get blob;
  CreatePostImageAspectRatioCopyWith<
    $R,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >?
  get aspectRatio;
  $R call({
    CreatePostBlob? blob,
    String? alt,
    CreatePostImageAspectRatio? aspectRatio,
  });
  CreatePostImageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _CreatePostImageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CreatePostImage, $Out>
    implements CreatePostImageCopyWith<$R, CreatePostImage, $Out> {
  _CreatePostImageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CreatePostImage> $mapper =
      CreatePostImageMapper.ensureInitialized();
  @override
  CreatePostBlobCopyWith<$R, CreatePostBlob, CreatePostBlob> get blob =>
      $value.blob.copyWith.$chain((v) => call(blob: v));
  @override
  CreatePostImageAspectRatioCopyWith<
    $R,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >?
  get aspectRatio =>
      $value.aspectRatio?.copyWith.$chain((v) => call(aspectRatio: v));
  @override
  $R call({CreatePostBlob? blob, String? alt, Object? aspectRatio = $none}) =>
      $apply(
        FieldCopyWithData({
          if (blob != null) #blob: blob,
          if (alt != null) #alt: alt,
          if (aspectRatio != $none) #aspectRatio: aspectRatio,
        }),
      );
  @override
  CreatePostImage $make(CopyWithData data) => CreatePostImage(
    blob: data.get(#blob, or: $value.blob),
    alt: data.get(#alt, or: $value.alt),
    aspectRatio: data.get(#aspectRatio, or: $value.aspectRatio),
  );

  @override
  CreatePostImageCopyWith<$R2, CreatePostImage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _CreatePostImageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class CreatePostBlobMapper extends ClassMapperBase<CreatePostBlob> {
  CreatePostBlobMapper._();

  static CreatePostBlobMapper? _instance;
  static CreatePostBlobMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CreatePostBlobMapper._());
      CreatePostBlobRefMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'CreatePostBlob';

  static CreatePostBlobRef _$ref(CreatePostBlob v) => v.ref;
  static const Field<CreatePostBlob, CreatePostBlobRef> _f$ref = Field(
    'ref',
    _$ref,
  );
  static String _$mimeType(CreatePostBlob v) => v.mimeType;
  static const Field<CreatePostBlob, String> _f$mimeType = Field(
    'mimeType',
    _$mimeType,
  );
  static int _$size(CreatePostBlob v) => v.size;
  static const Field<CreatePostBlob, int> _f$size = Field('size', _$size);
  static String _$type(CreatePostBlob v) => v.type;
  static const Field<CreatePostBlob, String> _f$type = Field(
    'type',
    _$type,
    key: r'$type',
    opt: true,
    def: 'blob',
  );

  @override
  final MappableFields<CreatePostBlob> fields = const {
    #ref: _f$ref,
    #mimeType: _f$mimeType,
    #size: _f$size,
    #type: _f$type,
  };

  static CreatePostBlob _instantiate(DecodingData data) {
    return CreatePostBlob(
      ref: data.dec(_f$ref),
      mimeType: data.dec(_f$mimeType),
      size: data.dec(_f$size),
      type: data.dec(_f$type),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static CreatePostBlob fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CreatePostBlob>(map);
  }

  static CreatePostBlob fromJson(String json) {
    return ensureInitialized().decodeJson<CreatePostBlob>(json);
  }
}

mixin CreatePostBlobMappable {
  String toJson() {
    return CreatePostBlobMapper.ensureInitialized().encodeJson<CreatePostBlob>(
      this as CreatePostBlob,
    );
  }

  Map<String, dynamic> toMap() {
    return CreatePostBlobMapper.ensureInitialized().encodeMap<CreatePostBlob>(
      this as CreatePostBlob,
    );
  }

  CreatePostBlobCopyWith<CreatePostBlob, CreatePostBlob, CreatePostBlob>
  get copyWith => _CreatePostBlobCopyWithImpl<CreatePostBlob, CreatePostBlob>(
    this as CreatePostBlob,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return CreatePostBlobMapper.ensureInitialized().stringifyValue(
      this as CreatePostBlob,
    );
  }

  @override
  bool operator ==(Object other) {
    return CreatePostBlobMapper.ensureInitialized().equalsValue(
      this as CreatePostBlob,
      other,
    );
  }

  @override
  int get hashCode {
    return CreatePostBlobMapper.ensureInitialized().hashValue(
      this as CreatePostBlob,
    );
  }
}

extension CreatePostBlobValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CreatePostBlob, $Out> {
  CreatePostBlobCopyWith<$R, CreatePostBlob, $Out> get $asCreatePostBlob =>
      $base.as((v, t, t2) => _CreatePostBlobCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class CreatePostBlobCopyWith<$R, $In extends CreatePostBlob, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  CreatePostBlobRefCopyWith<$R, CreatePostBlobRef, CreatePostBlobRef> get ref;
  $R call({CreatePostBlobRef? ref, String? mimeType, int? size, String? type});
  CreatePostBlobCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _CreatePostBlobCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CreatePostBlob, $Out>
    implements CreatePostBlobCopyWith<$R, CreatePostBlob, $Out> {
  _CreatePostBlobCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CreatePostBlob> $mapper =
      CreatePostBlobMapper.ensureInitialized();
  @override
  CreatePostBlobRefCopyWith<$R, CreatePostBlobRef, CreatePostBlobRef> get ref =>
      $value.ref.copyWith.$chain((v) => call(ref: v));
  @override
  $R call({
    CreatePostBlobRef? ref,
    String? mimeType,
    int? size,
    String? type,
  }) => $apply(
    FieldCopyWithData({
      if (ref != null) #ref: ref,
      if (mimeType != null) #mimeType: mimeType,
      if (size != null) #size: size,
      if (type != null) #type: type,
    }),
  );
  @override
  CreatePostBlob $make(CopyWithData data) => CreatePostBlob(
    ref: data.get(#ref, or: $value.ref),
    mimeType: data.get(#mimeType, or: $value.mimeType),
    size: data.get(#size, or: $value.size),
    type: data.get(#type, or: $value.type),
  );

  @override
  CreatePostBlobCopyWith<$R2, CreatePostBlob, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _CreatePostBlobCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class CreatePostBlobRefMapper extends ClassMapperBase<CreatePostBlobRef> {
  CreatePostBlobRefMapper._();

  static CreatePostBlobRefMapper? _instance;
  static CreatePostBlobRefMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CreatePostBlobRefMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'CreatePostBlobRef';

  static String _$link(CreatePostBlobRef v) => v.link;
  static const Field<CreatePostBlobRef, String> _f$link = Field(
    'link',
    _$link,
    key: r'$link',
  );

  @override
  final MappableFields<CreatePostBlobRef> fields = const {#link: _f$link};

  static CreatePostBlobRef _instantiate(DecodingData data) {
    return CreatePostBlobRef(link: data.dec(_f$link));
  }

  @override
  final Function instantiate = _instantiate;

  static CreatePostBlobRef fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CreatePostBlobRef>(map);
  }

  static CreatePostBlobRef fromJson(String json) {
    return ensureInitialized().decodeJson<CreatePostBlobRef>(json);
  }
}

mixin CreatePostBlobRefMappable {
  String toJson() {
    return CreatePostBlobRefMapper.ensureInitialized()
        .encodeJson<CreatePostBlobRef>(this as CreatePostBlobRef);
  }

  Map<String, dynamic> toMap() {
    return CreatePostBlobRefMapper.ensureInitialized()
        .encodeMap<CreatePostBlobRef>(this as CreatePostBlobRef);
  }

  CreatePostBlobRefCopyWith<
    CreatePostBlobRef,
    CreatePostBlobRef,
    CreatePostBlobRef
  >
  get copyWith =>
      _CreatePostBlobRefCopyWithImpl<CreatePostBlobRef, CreatePostBlobRef>(
        this as CreatePostBlobRef,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return CreatePostBlobRefMapper.ensureInitialized().stringifyValue(
      this as CreatePostBlobRef,
    );
  }

  @override
  bool operator ==(Object other) {
    return CreatePostBlobRefMapper.ensureInitialized().equalsValue(
      this as CreatePostBlobRef,
      other,
    );
  }

  @override
  int get hashCode {
    return CreatePostBlobRefMapper.ensureInitialized().hashValue(
      this as CreatePostBlobRef,
    );
  }
}

extension CreatePostBlobRefValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CreatePostBlobRef, $Out> {
  CreatePostBlobRefCopyWith<$R, CreatePostBlobRef, $Out>
  get $asCreatePostBlobRef => $base.as(
    (v, t, t2) => _CreatePostBlobRefCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class CreatePostBlobRefCopyWith<
  $R,
  $In extends CreatePostBlobRef,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? link});
  CreatePostBlobRefCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _CreatePostBlobRefCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CreatePostBlobRef, $Out>
    implements CreatePostBlobRefCopyWith<$R, CreatePostBlobRef, $Out> {
  _CreatePostBlobRefCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CreatePostBlobRef> $mapper =
      CreatePostBlobRefMapper.ensureInitialized();
  @override
  $R call({String? link}) =>
      $apply(FieldCopyWithData({if (link != null) #link: link}));
  @override
  CreatePostBlobRef $make(CopyWithData data) =>
      CreatePostBlobRef(link: data.get(#link, or: $value.link));

  @override
  CreatePostBlobRefCopyWith<$R2, CreatePostBlobRef, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _CreatePostBlobRefCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class CreatePostImageAspectRatioMapper
    extends ClassMapperBase<CreatePostImageAspectRatio> {
  CreatePostImageAspectRatioMapper._();

  static CreatePostImageAspectRatioMapper? _instance;
  static CreatePostImageAspectRatioMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = CreatePostImageAspectRatioMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'CreatePostImageAspectRatio';

  static int _$width(CreatePostImageAspectRatio v) => v.width;
  static const Field<CreatePostImageAspectRatio, int> _f$width = Field(
    'width',
    _$width,
  );
  static int _$height(CreatePostImageAspectRatio v) => v.height;
  static const Field<CreatePostImageAspectRatio, int> _f$height = Field(
    'height',
    _$height,
  );

  @override
  final MappableFields<CreatePostImageAspectRatio> fields = const {
    #width: _f$width,
    #height: _f$height,
  };

  static CreatePostImageAspectRatio _instantiate(DecodingData data) {
    return CreatePostImageAspectRatio(
      width: data.dec(_f$width),
      height: data.dec(_f$height),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static CreatePostImageAspectRatio fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CreatePostImageAspectRatio>(map);
  }

  static CreatePostImageAspectRatio fromJson(String json) {
    return ensureInitialized().decodeJson<CreatePostImageAspectRatio>(json);
  }
}

mixin CreatePostImageAspectRatioMappable {
  String toJson() {
    return CreatePostImageAspectRatioMapper.ensureInitialized()
        .encodeJson<CreatePostImageAspectRatio>(
          this as CreatePostImageAspectRatio,
        );
  }

  Map<String, dynamic> toMap() {
    return CreatePostImageAspectRatioMapper.ensureInitialized()
        .encodeMap<CreatePostImageAspectRatio>(
          this as CreatePostImageAspectRatio,
        );
  }

  CreatePostImageAspectRatioCopyWith<
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio,
    CreatePostImageAspectRatio
  >
  get copyWith =>
      _CreatePostImageAspectRatioCopyWithImpl<
        CreatePostImageAspectRatio,
        CreatePostImageAspectRatio
      >(this as CreatePostImageAspectRatio, $identity, $identity);
  @override
  String toString() {
    return CreatePostImageAspectRatioMapper.ensureInitialized().stringifyValue(
      this as CreatePostImageAspectRatio,
    );
  }

  @override
  bool operator ==(Object other) {
    return CreatePostImageAspectRatioMapper.ensureInitialized().equalsValue(
      this as CreatePostImageAspectRatio,
      other,
    );
  }

  @override
  int get hashCode {
    return CreatePostImageAspectRatioMapper.ensureInitialized().hashValue(
      this as CreatePostImageAspectRatio,
    );
  }
}

extension CreatePostImageAspectRatioValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CreatePostImageAspectRatio, $Out> {
  CreatePostImageAspectRatioCopyWith<$R, CreatePostImageAspectRatio, $Out>
  get $asCreatePostImageAspectRatio => $base.as(
    (v, t, t2) => _CreatePostImageAspectRatioCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class CreatePostImageAspectRatioCopyWith<
  $R,
  $In extends CreatePostImageAspectRatio,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({int? width, int? height});
  CreatePostImageAspectRatioCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _CreatePostImageAspectRatioCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CreatePostImageAspectRatio, $Out>
    implements
        CreatePostImageAspectRatioCopyWith<
          $R,
          CreatePostImageAspectRatio,
          $Out
        > {
  _CreatePostImageAspectRatioCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CreatePostImageAspectRatio> $mapper =
      CreatePostImageAspectRatioMapper.ensureInitialized();
  @override
  $R call({int? width, int? height}) => $apply(
    FieldCopyWithData({
      if (width != null) #width: width,
      if (height != null) #height: height,
    }),
  );
  @override
  CreatePostImageAspectRatio $make(CopyWithData data) =>
      CreatePostImageAspectRatio(
        width: data.get(#width, or: $value.width),
        height: data.get(#height, or: $value.height),
      );

  @override
  CreatePostImageAspectRatioCopyWith<$R2, CreatePostImageAspectRatio, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _CreatePostImageAspectRatioCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
