// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_customisation_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileCustomisationEditor)
final profileCustomisationEditorProvider =
    ProfileCustomisationEditorProvider._();

final class ProfileCustomisationEditorProvider
    extends
        $AsyncNotifierProvider<
          ProfileCustomisationEditor,
          ProfileCustomisationEditorState
        > {
  ProfileCustomisationEditorProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'profileCustomisationEditorProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$profileCustomisationEditorHash();

  @$internal
  @override
  ProfileCustomisationEditor create() => ProfileCustomisationEditor();
}

String _$profileCustomisationEditorHash() =>
    r'1dfeec97d8982f79e791bfc489efb24b761a176a';

abstract class _$ProfileCustomisationEditor
    extends $AsyncNotifier<ProfileCustomisationEditorState> {
  FutureOr<ProfileCustomisationEditorState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<ProfileCustomisationEditorState>,
              ProfileCustomisationEditorState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<ProfileCustomisationEditorState>,
                ProfileCustomisationEditorState
              >,
              AsyncValue<ProfileCustomisationEditorState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
