// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'relationship_list_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(RelationshipList)
final relationshipListProvider = RelationshipListFamily._();

final class RelationshipListProvider
    extends $AsyncNotifierProvider<RelationshipList, RelationshipListState> {
  RelationshipListProvider._({
    required RelationshipListFamily super.from,
    required RelationshipListKind super.argument,
  }) : super(
         retry: null,
         name: r'relationshipListProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$relationshipListHash();

  @override
  String toString() {
    return r'relationshipListProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  RelationshipList create() => RelationshipList();

  @override
  bool operator ==(Object other) {
    return other is RelationshipListProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$relationshipListHash() => r'603723b0a4bfbb101a537bf9eace416c7bf4fd17';

final class RelationshipListFamily extends $Family
    with
        $ClassFamilyOverride<
          RelationshipList,
          AsyncValue<RelationshipListState>,
          RelationshipListState,
          FutureOr<RelationshipListState>,
          RelationshipListKind
        > {
  RelationshipListFamily._()
    : super(
        retry: null,
        name: r'relationshipListProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  RelationshipListProvider call(RelationshipListKind kind) =>
      RelationshipListProvider._(argument: kind, from: this);

  @override
  String toString() => r'relationshipListProvider';
}

abstract class _$RelationshipList
    extends $AsyncNotifier<RelationshipListState> {
  late final _$args = ref.$arg as RelationshipListKind;
  RelationshipListKind get kind => _$args;

  FutureOr<RelationshipListState> build(RelationshipListKind kind);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<RelationshipListState>, RelationshipListState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<RelationshipListState>,
                RelationshipListState
              >,
              AsyncValue<RelationshipListState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
