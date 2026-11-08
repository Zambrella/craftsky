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
    r'0182f8fc5bf280c7859f711f6524a3222040885e';

abstract class _$ProfileCustomisationEditor
    extends $AsyncNotifier<ProfileCustomisationEditorState> {
  FutureOr<ProfileCustomisationEditorState> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
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
    return element.handleCreate(ref, build);
  }
}
