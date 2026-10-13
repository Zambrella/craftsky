// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'scheduled_posts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ScheduledPosts)
final scheduledPostsProvider = ScheduledPostsFamily._();

final class ScheduledPostsProvider
    extends $AsyncNotifierProvider<ScheduledPosts, ScheduledPostListState> {
  ScheduledPostsProvider._({
    required ScheduledPostsFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'scheduledPostsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$scheduledPostsHash();

  @override
  String toString() {
    return r'scheduledPostsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ScheduledPosts create() => ScheduledPosts();

  @override
  bool operator ==(Object other) {
    return other is ScheduledPostsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$scheduledPostsHash() => r'37b595c104fc9a255b487b3586f756e98bef70c8';

final class ScheduledPostsFamily extends $Family
    with
        $ClassFamilyOverride<
          ScheduledPosts,
          AsyncValue<ScheduledPostListState>,
          ScheduledPostListState,
          FutureOr<ScheduledPostListState>,
          AccountKey
        > {
  ScheduledPostsFamily._()
    : super(
        retry: null,
        name: r'scheduledPostsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ScheduledPostsProvider call(AccountKey account) =>
      ScheduledPostsProvider._(argument: account, from: this);

  @override
  String toString() => r'scheduledPostsProvider';
}

abstract class _$ScheduledPosts extends $AsyncNotifier<ScheduledPostListState> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<ScheduledPostListState> build(AccountKey account);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<ScheduledPostListState>, ScheduledPostListState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<ScheduledPostListState>,
                ScheduledPostListState
              >,
              AsyncValue<ScheduledPostListState>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
