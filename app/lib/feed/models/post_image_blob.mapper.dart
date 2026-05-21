// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'post_image_blob.dart';

class UploadedImageBlobMapper extends ClassMapperBase<UploadedImageBlob> {
  UploadedImageBlobMapper._();

  static UploadedImageBlobMapper? _instance;
  static UploadedImageBlobMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UploadedImageBlobMapper._());
      UploadedBlobMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'UploadedImageBlob';

  static UploadedBlob _$blob(UploadedImageBlob v) => v.blob;
  static const Field<UploadedImageBlob, UploadedBlob> _f$blob = Field(
    'blob',
    _$blob,
  );
  static String _$cid(UploadedImageBlob v) => v.cid;
  static const Field<UploadedImageBlob, String> _f$cid = Field('cid', _$cid);
  static String _$mime(UploadedImageBlob v) => v.mime;
  static const Field<UploadedImageBlob, String> _f$mime = Field('mime', _$mime);
  static int _$size(UploadedImageBlob v) => v.size;
  static const Field<UploadedImageBlob, int> _f$size = Field('size', _$size);

  @override
  final MappableFields<UploadedImageBlob> fields = const {
    #blob: _f$blob,
    #cid: _f$cid,
    #mime: _f$mime,
    #size: _f$size,
  };

  static UploadedImageBlob _instantiate(DecodingData data) {
    return UploadedImageBlob(
      blob: data.dec(_f$blob),
      cid: data.dec(_f$cid),
      mime: data.dec(_f$mime),
      size: data.dec(_f$size),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UploadedImageBlob fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UploadedImageBlob>(map);
  }

  static UploadedImageBlob fromJson(String json) {
    return ensureInitialized().decodeJson<UploadedImageBlob>(json);
  }
}

mixin UploadedImageBlobMappable {
  String toJson() {
    return UploadedImageBlobMapper.ensureInitialized()
        .encodeJson<UploadedImageBlob>(this as UploadedImageBlob);
  }

  Map<String, dynamic> toMap() {
    return UploadedImageBlobMapper.ensureInitialized()
        .encodeMap<UploadedImageBlob>(this as UploadedImageBlob);
  }

  UploadedImageBlobCopyWith<
    UploadedImageBlob,
    UploadedImageBlob,
    UploadedImageBlob
  >
  get copyWith =>
      _UploadedImageBlobCopyWithImpl<UploadedImageBlob, UploadedImageBlob>(
        this as UploadedImageBlob,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return UploadedImageBlobMapper.ensureInitialized().stringifyValue(
      this as UploadedImageBlob,
    );
  }

  @override
  bool operator ==(Object other) {
    return UploadedImageBlobMapper.ensureInitialized().equalsValue(
      this as UploadedImageBlob,
      other,
    );
  }

  @override
  int get hashCode {
    return UploadedImageBlobMapper.ensureInitialized().hashValue(
      this as UploadedImageBlob,
    );
  }
}

extension UploadedImageBlobValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UploadedImageBlob, $Out> {
  UploadedImageBlobCopyWith<$R, UploadedImageBlob, $Out>
  get $asUploadedImageBlob => $base.as(
    (v, t, t2) => _UploadedImageBlobCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class UploadedImageBlobCopyWith<
  $R,
  $In extends UploadedImageBlob,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  UploadedBlobCopyWith<$R, UploadedBlob, UploadedBlob> get blob;
  $R call({UploadedBlob? blob, String? cid, String? mime, int? size});
  UploadedImageBlobCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UploadedImageBlobCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UploadedImageBlob, $Out>
    implements UploadedImageBlobCopyWith<$R, UploadedImageBlob, $Out> {
  _UploadedImageBlobCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UploadedImageBlob> $mapper =
      UploadedImageBlobMapper.ensureInitialized();
  @override
  UploadedBlobCopyWith<$R, UploadedBlob, UploadedBlob> get blob =>
      $value.blob.copyWith.$chain((v) => call(blob: v));
  @override
  $R call({UploadedBlob? blob, String? cid, String? mime, int? size}) => $apply(
    FieldCopyWithData({
      if (blob != null) #blob: blob,
      if (cid != null) #cid: cid,
      if (mime != null) #mime: mime,
      if (size != null) #size: size,
    }),
  );
  @override
  UploadedImageBlob $make(CopyWithData data) => UploadedImageBlob(
    blob: data.get(#blob, or: $value.blob),
    cid: data.get(#cid, or: $value.cid),
    mime: data.get(#mime, or: $value.mime),
    size: data.get(#size, or: $value.size),
  );

  @override
  UploadedImageBlobCopyWith<$R2, UploadedImageBlob, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _UploadedImageBlobCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class UploadedBlobMapper extends ClassMapperBase<UploadedBlob> {
  UploadedBlobMapper._();

  static UploadedBlobMapper? _instance;
  static UploadedBlobMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UploadedBlobMapper._());
      UploadedBlobRefMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'UploadedBlob';

  static String _$type(UploadedBlob v) => v.type;
  static const Field<UploadedBlob, String> _f$type = Field(
    'type',
    _$type,
    key: r'$type',
  );
  static UploadedBlobRef _$ref(UploadedBlob v) => v.ref;
  static const Field<UploadedBlob, UploadedBlobRef> _f$ref = Field(
    'ref',
    _$ref,
  );
  static String _$mimeType(UploadedBlob v) => v.mimeType;
  static const Field<UploadedBlob, String> _f$mimeType = Field(
    'mimeType',
    _$mimeType,
  );
  static int _$size(UploadedBlob v) => v.size;
  static const Field<UploadedBlob, int> _f$size = Field('size', _$size);

  @override
  final MappableFields<UploadedBlob> fields = const {
    #type: _f$type,
    #ref: _f$ref,
    #mimeType: _f$mimeType,
    #size: _f$size,
  };

  static UploadedBlob _instantiate(DecodingData data) {
    return UploadedBlob(
      type: data.dec(_f$type),
      ref: data.dec(_f$ref),
      mimeType: data.dec(_f$mimeType),
      size: data.dec(_f$size),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UploadedBlob fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UploadedBlob>(map);
  }

  static UploadedBlob fromJson(String json) {
    return ensureInitialized().decodeJson<UploadedBlob>(json);
  }
}

mixin UploadedBlobMappable {
  String toJson() {
    return UploadedBlobMapper.ensureInitialized().encodeJson<UploadedBlob>(
      this as UploadedBlob,
    );
  }

  Map<String, dynamic> toMap() {
    return UploadedBlobMapper.ensureInitialized().encodeMap<UploadedBlob>(
      this as UploadedBlob,
    );
  }

  UploadedBlobCopyWith<UploadedBlob, UploadedBlob, UploadedBlob> get copyWith =>
      _UploadedBlobCopyWithImpl<UploadedBlob, UploadedBlob>(
        this as UploadedBlob,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return UploadedBlobMapper.ensureInitialized().stringifyValue(
      this as UploadedBlob,
    );
  }

  @override
  bool operator ==(Object other) {
    return UploadedBlobMapper.ensureInitialized().equalsValue(
      this as UploadedBlob,
      other,
    );
  }

  @override
  int get hashCode {
    return UploadedBlobMapper.ensureInitialized().hashValue(
      this as UploadedBlob,
    );
  }
}

extension UploadedBlobValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UploadedBlob, $Out> {
  UploadedBlobCopyWith<$R, UploadedBlob, $Out> get $asUploadedBlob =>
      $base.as((v, t, t2) => _UploadedBlobCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class UploadedBlobCopyWith<$R, $In extends UploadedBlob, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  UploadedBlobRefCopyWith<$R, UploadedBlobRef, UploadedBlobRef> get ref;
  $R call({String? type, UploadedBlobRef? ref, String? mimeType, int? size});
  UploadedBlobCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _UploadedBlobCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UploadedBlob, $Out>
    implements UploadedBlobCopyWith<$R, UploadedBlob, $Out> {
  _UploadedBlobCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UploadedBlob> $mapper =
      UploadedBlobMapper.ensureInitialized();
  @override
  UploadedBlobRefCopyWith<$R, UploadedBlobRef, UploadedBlobRef> get ref =>
      $value.ref.copyWith.$chain((v) => call(ref: v));
  @override
  $R call({String? type, UploadedBlobRef? ref, String? mimeType, int? size}) =>
      $apply(
        FieldCopyWithData({
          if (type != null) #type: type,
          if (ref != null) #ref: ref,
          if (mimeType != null) #mimeType: mimeType,
          if (size != null) #size: size,
        }),
      );
  @override
  UploadedBlob $make(CopyWithData data) => UploadedBlob(
    type: data.get(#type, or: $value.type),
    ref: data.get(#ref, or: $value.ref),
    mimeType: data.get(#mimeType, or: $value.mimeType),
    size: data.get(#size, or: $value.size),
  );

  @override
  UploadedBlobCopyWith<$R2, UploadedBlob, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _UploadedBlobCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class UploadedBlobRefMapper extends ClassMapperBase<UploadedBlobRef> {
  UploadedBlobRefMapper._();

  static UploadedBlobRefMapper? _instance;
  static UploadedBlobRefMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UploadedBlobRefMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'UploadedBlobRef';

  static String _$link(UploadedBlobRef v) => v.link;
  static const Field<UploadedBlobRef, String> _f$link = Field(
    'link',
    _$link,
    key: r'$link',
  );

  @override
  final MappableFields<UploadedBlobRef> fields = const {#link: _f$link};

  static UploadedBlobRef _instantiate(DecodingData data) {
    return UploadedBlobRef(link: data.dec(_f$link));
  }

  @override
  final Function instantiate = _instantiate;

  static UploadedBlobRef fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UploadedBlobRef>(map);
  }

  static UploadedBlobRef fromJson(String json) {
    return ensureInitialized().decodeJson<UploadedBlobRef>(json);
  }
}

mixin UploadedBlobRefMappable {
  String toJson() {
    return UploadedBlobRefMapper.ensureInitialized()
        .encodeJson<UploadedBlobRef>(this as UploadedBlobRef);
  }

  Map<String, dynamic> toMap() {
    return UploadedBlobRefMapper.ensureInitialized().encodeMap<UploadedBlobRef>(
      this as UploadedBlobRef,
    );
  }

  UploadedBlobRefCopyWith<UploadedBlobRef, UploadedBlobRef, UploadedBlobRef>
  get copyWith =>
      _UploadedBlobRefCopyWithImpl<UploadedBlobRef, UploadedBlobRef>(
        this as UploadedBlobRef,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return UploadedBlobRefMapper.ensureInitialized().stringifyValue(
      this as UploadedBlobRef,
    );
  }

  @override
  bool operator ==(Object other) {
    return UploadedBlobRefMapper.ensureInitialized().equalsValue(
      this as UploadedBlobRef,
      other,
    );
  }

  @override
  int get hashCode {
    return UploadedBlobRefMapper.ensureInitialized().hashValue(
      this as UploadedBlobRef,
    );
  }
}

extension UploadedBlobRefValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UploadedBlobRef, $Out> {
  UploadedBlobRefCopyWith<$R, UploadedBlobRef, $Out> get $asUploadedBlobRef =>
      $base.as((v, t, t2) => _UploadedBlobRefCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class UploadedBlobRefCopyWith<$R, $In extends UploadedBlobRef, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? link});
  UploadedBlobRefCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UploadedBlobRefCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UploadedBlobRef, $Out>
    implements UploadedBlobRefCopyWith<$R, UploadedBlobRef, $Out> {
  _UploadedBlobRefCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UploadedBlobRef> $mapper =
      UploadedBlobRefMapper.ensureInitialized();
  @override
  $R call({String? link}) =>
      $apply(FieldCopyWithData({if (link != null) #link: link}));
  @override
  UploadedBlobRef $make(CopyWithData data) =>
      UploadedBlobRef(link: data.get(#link, or: $value.link));

  @override
  UploadedBlobRefCopyWith<$R2, UploadedBlobRef, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _UploadedBlobRefCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

