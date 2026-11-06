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
    required (
      Did,
      RecordKey,
      PostInteractionAccountKind, {
      BrowsingContext? audience,
    })
    super.argument,
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
    r'634191240aa173ba7e0712e25cf2d284737452ed';

final class PostInteractionAccountsFamily extends $Family
    with
        $ClassFamilyOverride<
          PostInteractionAccounts,
          AsyncValue<PostInteractionAccountsState>,
          PostInteractionAccountsState,
          FutureOr<PostInteractionAccountsState>,
          (
            Did,
            RecordKey,
            PostInteractionAccountKind, {
            BrowsingContext? audience,
          })
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
    PostInteractionAccountKind kind, {
    BrowsingContext? audience,
  }) => PostInteractionAccountsProvider._(
    argument: (did, rkey, kind, audience: audience),
    from: this,
  );

  @override
  String toString() => r'postInteractionAccountsProvider';
}

abstract class _$PostInteractionAccounts
    extends $AsyncNotifier<PostInteractionAccountsState> {
  late final _$args =
      ref.$arg
          as (
            Did,
            RecordKey,
            PostInteractionAccountKind, {
            BrowsingContext? audience,
          });
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;
  PostInteractionAccountKind get kind => _$args.$3;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<PostInteractionAccountsState> build(
    Did did,
    RecordKey rkey,
    PostInteractionAccountKind kind, {
    BrowsingContext? audience,
  });
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
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
    return element.handleCreate(
      ref,
      () => build(_$args.$1, _$args.$2, _$args.$3, audience: _$args.audience),
    );
  }
}

@ProviderFor(PostQuotes)
final postQuotesProvider = PostQuotesFamily._();

final class PostQuotesProvider
    extends $AsyncNotifierProvider<PostQuotes, PostQuotesState> {
  PostQuotesProvider._({
    required PostQuotesFamily super.from,
    required (Did, RecordKey, {BrowsingContext? audience}) super.argument,
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

String _$postQuotesHash() => r'2193424f3ac3d6c1bf474bb863b5848d20b17d91';

final class PostQuotesFamily extends $Family
    with
        $ClassFamilyOverride<
          PostQuotes,
          AsyncValue<PostQuotesState>,
          PostQuotesState,
          FutureOr<PostQuotesState>,
          (Did, RecordKey, {BrowsingContext? audience})
        > {
  PostQuotesFamily._()
    : super(
        retry: null,
        name: r'postQuotesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostQuotesProvider call(
    Did did,
    RecordKey rkey, {
    BrowsingContext? audience,
  }) => PostQuotesProvider._(
    argument: (did, rkey, audience: audience),
    from: this,
  );

  @override
  String toString() => r'postQuotesProvider';
}

abstract class _$PostQuotes extends $AsyncNotifier<PostQuotesState> {
  late final _$args = ref.$arg as (Did, RecordKey, {BrowsingContext? audience});
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<PostQuotesState> build(
    Did did,
    RecordKey rkey, {
    BrowsingContext? audience,
  });
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<PostQuotesState>, PostQuotesState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<PostQuotesState>, PostQuotesState>,
              AsyncValue<PostQuotesState>,
              Object?,
              Object?
            >;
    return element.handleCreate(
      ref,
      () => build(_$args.$1, _$args.$2, audience: _$args.audience),
    );
  }
}
