// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post_viewer_state.dart';

class SavedPostPresentationMapper
    extends ClassMapperBase<SavedPostPresentation> {
  SavedPostPresentationMapper._();

  static SavedPostPresentationMapper? _instance;
  static SavedPostPresentationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostPresentationMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostPresentation';

  static bool _$initialized(SavedPostPresentation v) => v.initialized;
  static const Field<SavedPostPresentation, bool> _f$initialized = Field(
    'initialized',
    _$initialized,
  );
  static bool _$isSaved(SavedPostPresentation v) => v.isSaved;
  static const Field<SavedPostPresentation, bool> _f$isSaved = Field(
    'isSaved',
    _$isSaved,
  );
  static int _$revision(SavedPostPresentation v) => v.revision;
  static const Field<SavedPostPresentation, int> _f$revision = Field(
    'revision',
    _$revision,
  );
  static String? _$folderId(SavedPostPresentation v) => v.folderId;
  static const Field<SavedPostPresentation, String> _f$folderId = Field(
    'folderId',
    _$folderId,
    opt: true,
  );
  static DateTime? _$savedAt(SavedPostPresentation v) => v.savedAt;
  static const Field<SavedPostPresentation, DateTime> _f$savedAt = Field(
    'savedAt',
    _$savedAt,
    opt: true,
  );
  static SavedPostMutation? _$pendingMutation(SavedPostPresentation v) =>
      v.pendingMutation;
  static const Field<SavedPostPresentation, SavedPostMutation>
  _f$pendingMutation = Field('pendingMutation', _$pendingMutation, opt: true);
  static Object? _$lastError(SavedPostPresentation v) => v.lastError;
  static const Field<SavedPostPresentation, Object> _f$lastError = Field(
    'lastError',
    _$lastError,
    opt: true,
  );

  @override
  final MappableFields<SavedPostPresentation> fields = const {
    #initialized: _f$initialized,
    #isSaved: _f$isSaved,
    #revision: _f$revision,
    #folderId: _f$folderId,
    #savedAt: _f$savedAt,
    #pendingMutation: _f$pendingMutation,
    #lastError: _f$lastError,
  };

  static SavedPostPresentation _instantiate(DecodingData data) {
    return SavedPostPresentation(
      initialized: data.dec(_f$initialized),
      isSaved: data.dec(_f$isSaved),
      revision: data.dec(_f$revision),
      folderId: data.dec(_f$folderId),
      savedAt: data.dec(_f$savedAt),
      pendingMutation: data.dec(_f$pendingMutation),
      lastError: data.dec(_f$lastError),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostPresentationMappable {
  SavedPostPresentationCopyWith<
    SavedPostPresentation,
    SavedPostPresentation,
    SavedPostPresentation
  >
  get copyWith =>
      _SavedPostPresentationCopyWithImpl<
        SavedPostPresentation,
        SavedPostPresentation
      >(this as SavedPostPresentation, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return SavedPostPresentationMapper.ensureInitialized().equalsValue(
      this as SavedPostPresentation,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostPresentationMapper.ensureInitialized().hashValue(
      this as SavedPostPresentation,
    );
  }
}

extension SavedPostPresentationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostPresentation, $Out> {
  SavedPostPresentationCopyWith<$R, SavedPostPresentation, $Out>
  get $asSavedPostPresentation => $base.as(
    (v, t, t2) => _SavedPostPresentationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavedPostPresentationCopyWith<
  $R,
  $In extends SavedPostPresentation,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    bool? initialized,
    bool? isSaved,
    int? revision,
    String? folderId,
    DateTime? savedAt,
    SavedPostMutation? pendingMutation,
    Object? lastError,
  });
  SavedPostPresentationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostPresentationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostPresentation, $Out>
    implements SavedPostPresentationCopyWith<$R, SavedPostPresentation, $Out> {
  _SavedPostPresentationCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostPresentation> $mapper =
      SavedPostPresentationMapper.ensureInitialized();
  @override
  $R call({
    bool? initialized,
    bool? isSaved,
    int? revision,
    Object? folderId = $none,
    Object? savedAt = $none,
    Object? pendingMutation = $none,
    Object? lastError = $none,
  }) => $apply(
    FieldCopyWithData({
      if (initialized != null) #initialized: initialized,
      if (isSaved != null) #isSaved: isSaved,
      if (revision != null) #revision: revision,
      if (folderId != $none) #folderId: folderId,
      if (savedAt != $none) #savedAt: savedAt,
      if (pendingMutation != $none) #pendingMutation: pendingMutation,
      if (lastError != $none) #lastError: lastError,
    }),
  );
  @override
  SavedPostPresentation $make(CopyWithData data) => SavedPostPresentation(
    initialized: data.get(#initialized, or: $value.initialized),
    isSaved: data.get(#isSaved, or: $value.isSaved),
    revision: data.get(#revision, or: $value.revision),
    folderId: data.get(#folderId, or: $value.folderId),
    savedAt: data.get(#savedAt, or: $value.savedAt),
    pendingMutation: data.get(#pendingMutation, or: $value.pendingMutation),
    lastError: data.get(#lastError, or: $value.lastError),
  );

  @override
  SavedPostPresentationCopyWith<$R2, SavedPostPresentation, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SavedPostPresentationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

