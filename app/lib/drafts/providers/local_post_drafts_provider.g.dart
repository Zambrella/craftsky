// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'local_post_drafts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(LocalPostDrafts)
final localPostDraftsProvider = LocalPostDraftsFamily._();

final class LocalPostDraftsProvider
    extends $AsyncNotifierProvider<LocalPostDrafts, LocalPostDraftListState> {
  LocalPostDraftsProvider._({
    required LocalPostDraftsFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'localPostDraftsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$localPostDraftsHash();

  @override
  String toString() {
    return r'localPostDraftsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  LocalPostDrafts create() => LocalPostDrafts();

  @override
  bool operator ==(Object other) {
    return other is LocalPostDraftsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$localPostDraftsHash() => r'8e4816d0b049d65ed6a99f792bf7bbcde7ad09f2';

final class LocalPostDraftsFamily extends $Family
    with
        $ClassFamilyOverride<
          LocalPostDrafts,
          AsyncValue<LocalPostDraftListState>,
          LocalPostDraftListState,
          FutureOr<LocalPostDraftListState>,
          AccountKey
        > {
  LocalPostDraftsFamily._()
    : super(
        retry: null,
        name: r'localPostDraftsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  LocalPostDraftsProvider call(AccountKey account) =>
      LocalPostDraftsProvider._(argument: account, from: this);

  @override
  String toString() => r'localPostDraftsProvider';
}

abstract class _$LocalPostDrafts
    extends $AsyncNotifier<LocalPostDraftListState> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<LocalPostDraftListState> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<LocalPostDraftListState>,
              LocalPostDraftListState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<LocalPostDraftListState>,
                LocalPostDraftListState
              >,
              AsyncValue<LocalPostDraftListState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
