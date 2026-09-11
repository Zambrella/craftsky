// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_interaction_lists_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(PostInteractionAccounts)
final postInteractionAccountsProvider = PostInteractionAccountsFamily._();

final class PostInteractionAccountsProvider
    extends
        $AsyncNotifierProvider<
          PostInteractionAccounts,
          PostInteractionAccountsState
        > {
  PostInteractionAccountsProvider._({
    required PostInteractionAccountsFamily super.from,
    required (Did, RecordKey, PostInteractionAccountKind) super.argument,
  }) : super(
         retry: null,
         name: r'postInteractionAccountsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postInteractionAccountsHash();

  @override
  String toString() {
    return r'postInteractionAccountsProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  PostInteractionAccounts create() => PostInteractionAccounts();

  @override
  bool operator ==(Object other) {
    return other is PostInteractionAccountsProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postInteractionAccountsHash() =>
    r'9f302d3a69c73809ba7fed80f11fa754bcad06c3';

final class PostInteractionAccountsFamily extends $Family
    with
        $ClassFamilyOverride<
          PostInteractionAccounts,
          AsyncValue<PostInteractionAccountsState>,
          PostInteractionAccountsState,
          FutureOr<PostInteractionAccountsState>,
          (Did, RecordKey, PostInteractionAccountKind)
        > {
  PostInteractionAccountsFamily._()
    : super(
        retry: null,
        name: r'postInteractionAccountsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostInteractionAccountsProvider call(
    Did did,
    RecordKey rkey,
    PostInteractionAccountKind kind,
  ) => PostInteractionAccountsProvider._(
    argument: (did, rkey, kind),
    from: this,
  );

  @override
  String toString() => r'postInteractionAccountsProvider';
}

abstract class _$PostInteractionAccounts
    extends $AsyncNotifier<PostInteractionAccountsState> {
  late final _$args = ref.$arg as (Did, RecordKey, PostInteractionAccountKind);
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;
  PostInteractionAccountKind get kind => _$args.$3;

  FutureOr<PostInteractionAccountsState> build(
    Did did,
    RecordKey rkey,
    PostInteractionAccountKind kind,
  );
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<PostInteractionAccountsState>,
              PostInteractionAccountsState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<PostInteractionAccountsState>,
                PostInteractionAccountsState
              >,
              AsyncValue<PostInteractionAccountsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args.$1, _$args.$2, _$args.$3));
  }
}

@ProviderFor(PostQuotes)
final postQuotesProvider = PostQuotesFamily._();

final class PostQuotesProvider
    extends $AsyncNotifierProvider<PostQuotes, PostQuotesState> {
  PostQuotesProvider._({
    required PostQuotesFamily super.from,
    required (Did, RecordKey) super.argument,
  }) : super(
         retry: null,
         name: r'postQuotesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postQuotesHash();

  @override
  String toString() {
    return r'postQuotesProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  PostQuotes create() => PostQuotes();

  @override
  bool operator ==(Object other) {
    return other is PostQuotesProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postQuotesHash() => r'd379105d19e93aa44751e570fd50b0cc768f55a9';

final class PostQuotesFamily extends $Family
    with
        $ClassFamilyOverride<
          PostQuotes,
          AsyncValue<PostQuotesState>,
          PostQuotesState,
          FutureOr<PostQuotesState>,
          (Did, RecordKey)
        > {
  PostQuotesFamily._()
    : super(
        retry: null,
        name: r'postQuotesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostQuotesProvider call(Did did, RecordKey rkey) =>
      PostQuotesProvider._(argument: (did, rkey), from: this);

  @override
  String toString() => r'postQuotesProvider';
}

abstract class _$PostQuotes extends $AsyncNotifier<PostQuotesState> {
  late final _$args = ref.$arg as (Did, RecordKey);
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;

  FutureOr<PostQuotesState> build(Did did, RecordKey rkey);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<PostQuotesState>, PostQuotesState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<PostQuotesState>, PostQuotesState>,
              AsyncValue<PostQuotesState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args.$1, _$args.$2));
  }
}
