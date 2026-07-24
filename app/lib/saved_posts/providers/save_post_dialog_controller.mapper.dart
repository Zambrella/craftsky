// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'save_post_dialog_controller.dart';

class SavePostDialogStateMapper extends ClassMapperBase<SavePostDialogState> {
  SavePostDialogStateMapper._();

  static SavePostDialogStateMapper? _instance;
  static SavePostDialogStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavePostDialogStateMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavePostDialogState';

  static String? _$selectedFolderId(SavePostDialogState v) =>
      v.selectedFolderId;
  static const Field<SavePostDialogState, String> _f$selectedFolderId = Field(
    'selectedFolderId',
    _$selectedFolderId,
  );
  static bool _$isCreatingFolder(SavePostDialogState v) => v.isCreatingFolder;
  static const Field<SavePostDialogState, bool> _f$isCreatingFolder = Field(
    'isCreatingFolder',
    _$isCreatingFolder,
    opt: true,
    def: false,
  );
  static String _$createName(SavePostDialogState v) => v.createName;
  static const Field<SavePostDialogState, String> _f$createName = Field(
    'createName',
    _$createName,
    opt: true,
    def: '',
  );
  static bool _$isCreatePending(SavePostDialogState v) => v.isCreatePending;
  static const Field<SavePostDialogState, bool> _f$isCreatePending = Field(
    'isCreatePending',
    _$isCreatePending,
    opt: true,
    def: false,
  );
  static bool _$isConfirming(SavePostDialogState v) => v.isConfirming;
  static const Field<SavePostDialogState, bool> _f$isConfirming = Field(
    'isConfirming',
    _$isConfirming,
    opt: true,
    def: false,
  );
  static bool _$isConfirmed(SavePostDialogState v) => v.isConfirmed;
  static const Field<SavePostDialogState, bool> _f$isConfirmed = Field(
    'isConfirmed',
    _$isConfirmed,
    opt: true,
    def: false,
  );
  static bool _$isCancelled(SavePostDialogState v) => v.isCancelled;
  static const Field<SavePostDialogState, bool> _f$isCancelled = Field(
    'isCancelled',
    _$isCancelled,
    opt: true,
    def: false,
  );
  static SavePostDialogError? _$createError(SavePostDialogState v) =>
      v.createError;
  static const Field<SavePostDialogState, SavePostDialogError> _f$createError =
      Field('createError', _$createError, opt: true);
  static SavePostDialogError? _$confirmError(SavePostDialogState v) =>
      v.confirmError;
  static const Field<SavePostDialogState, SavePostDialogError> _f$confirmError =
      Field('confirmError', _$confirmError, opt: true);

  @override
  final MappableFields<SavePostDialogState> fields = const {
    #selectedFolderId: _f$selectedFolderId,
    #isCreatingFolder: _f$isCreatingFolder,
    #createName: _f$createName,
    #isCreatePending: _f$isCreatePending,
    #isConfirming: _f$isConfirming,
    #isConfirmed: _f$isConfirmed,
    #isCancelled: _f$isCancelled,
    #createError: _f$createError,
    #confirmError: _f$confirmError,
  };

  static SavePostDialogState _instantiate(DecodingData data) {
    return SavePostDialogState(
      selectedFolderId: data.dec(_f$selectedFolderId),
      isCreatingFolder: data.dec(_f$isCreatingFolder),
      createName: data.dec(_f$createName),
      isCreatePending: data.dec(_f$isCreatePending),
      isConfirming: data.dec(_f$isConfirming),
      isConfirmed: data.dec(_f$isConfirmed),
      isCancelled: data.dec(_f$isCancelled),
      createError: data.dec(_f$createError),
      confirmError: data.dec(_f$confirmError),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavePostDialogStateMappable {
  SavePostDialogStateCopyWith<
    SavePostDialogState,
    SavePostDialogState,
    SavePostDialogState
  >
  get copyWith =>
      _SavePostDialogStateCopyWithImpl<
        SavePostDialogState,
        SavePostDialogState
      >(this as SavePostDialogState, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return SavePostDialogStateMapper.ensureInitialized().equalsValue(
      this as SavePostDialogState,
      other,
    );
  }

  @override
  int get hashCode {
    return SavePostDialogStateMapper.ensureInitialized().hashValue(
      this as SavePostDialogState,
    );
  }
}

extension SavePostDialogStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavePostDialogState, $Out> {
  SavePostDialogStateCopyWith<$R, SavePostDialogState, $Out>
  get $asSavePostDialogState => $base.as(
    (v, t, t2) => _SavePostDialogStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavePostDialogStateCopyWith<
  $R,
  $In extends SavePostDialogState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? selectedFolderId,
    bool? isCreatingFolder,
    String? createName,
    bool? isCreatePending,
    bool? isConfirming,
    bool? isConfirmed,
    bool? isCancelled,
    SavePostDialogError? createError,
    SavePostDialogError? confirmError,
  });
  SavePostDialogStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavePostDialogStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavePostDialogState, $Out>
    implements SavePostDialogStateCopyWith<$R, SavePostDialogState, $Out> {
  _SavePostDialogStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavePostDialogState> $mapper =
      SavePostDialogStateMapper.ensureInitialized();
  @override
  $R call({
    Object? selectedFolderId = $none,
    bool? isCreatingFolder,
    String? createName,
    bool? isCreatePending,
    bool? isConfirming,
    bool? isConfirmed,
    bool? isCancelled,
    Object? createError = $none,
    Object? confirmError = $none,
  }) => $apply(
    FieldCopyWithData({
      if (selectedFolderId != $none) #selectedFolderId: selectedFolderId,
      if (isCreatingFolder != null) #isCreatingFolder: isCreatingFolder,
      if (createName != null) #createName: createName,
      if (isCreatePending != null) #isCreatePending: isCreatePending,
      if (isConfirming != null) #isConfirming: isConfirming,
      if (isConfirmed != null) #isConfirmed: isConfirmed,
      if (isCancelled != null) #isCancelled: isCancelled,
      if (createError != $none) #createError: createError,
      if (confirmError != $none) #confirmError: confirmError,
    }),
  );
  @override
  SavePostDialogState $make(CopyWithData data) => SavePostDialogState(
    selectedFolderId: data.get(#selectedFolderId, or: $value.selectedFolderId),
    isCreatingFolder: data.get(#isCreatingFolder, or: $value.isCreatingFolder),
    createName: data.get(#createName, or: $value.createName),
    isCreatePending: data.get(#isCreatePending, or: $value.isCreatePending),
    isConfirming: data.get(#isConfirming, or: $value.isConfirming),
    isConfirmed: data.get(#isConfirmed, or: $value.isConfirmed),
    isCancelled: data.get(#isCancelled, or: $value.isCancelled),
    createError: data.get(#createError, or: $value.createError),
    confirmError: data.get(#confirmError, or: $value.confirmError),
  );

  @override
  SavePostDialogStateCopyWith<$R2, SavePostDialogState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SavePostDialogStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

