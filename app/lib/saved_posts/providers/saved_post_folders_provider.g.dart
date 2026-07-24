// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'saved_post_folders_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SavedPostFolders)
final savedPostFoldersProvider = SavedPostFoldersFamily._();

final class SavedPostFoldersProvider
    extends $AsyncNotifierProvider<SavedPostFolders, SavedPostFolderListState> {
  SavedPostFoldersProvider._({
    required SavedPostFoldersFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'savedPostFoldersProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$savedPostFoldersHash();

  @override
  String toString() {
    return r'savedPostFoldersProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  SavedPostFolders create() => SavedPostFolders();

  @override
  bool operator ==(Object other) {
    return other is SavedPostFoldersProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$savedPostFoldersHash() => r'4c6960a7e0b84e72c2cd03744052f597ee9d48bb';

final class SavedPostFoldersFamily extends $Family
    with
        $ClassFamilyOverride<
          SavedPostFolders,
          AsyncValue<SavedPostFolderListState>,
          SavedPostFolderListState,
          FutureOr<SavedPostFolderListState>,
          AccountKey
        > {
  SavedPostFoldersFamily._()
    : super(
        retry: null,
        name: r'savedPostFoldersProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SavedPostFoldersProvider call(AccountKey account) =>
      SavedPostFoldersProvider._(argument: account, from: this);

  @override
  String toString() => r'savedPostFoldersProvider';
}

abstract class _$SavedPostFolders
    extends $AsyncNotifier<SavedPostFolderListState> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<SavedPostFolderListState> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<SavedPostFolderListState>,
              SavedPostFolderListState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<SavedPostFolderListState>,
                SavedPostFolderListState
              >,
              AsyncValue<SavedPostFolderListState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
