// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'recent_members_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(recentMembers)
final recentMembersProvider = RecentMembersFamily._();

final class RecentMembersProvider
    extends
        $FunctionalProvider<
          AsyncValue<List<ProfileAccountSummary>>,
          List<ProfileAccountSummary>,
          FutureOr<List<ProfileAccountSummary>>
        >
    with
        $FutureModifier<List<ProfileAccountSummary>>,
        $FutureProvider<List<ProfileAccountSummary>> {
  RecentMembersProvider._({
    required RecentMembersFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'recentMembersProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$recentMembersHash();

  @override
  String toString() {
    return r'recentMembersProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<List<ProfileAccountSummary>> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<List<ProfileAccountSummary>> create(Ref ref) {
    final argument = this.argument as ActiveAccountLease;
    return recentMembers(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is RecentMembersProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$recentMembersHash() => r'59938c53e831a8fc489a51b5789b07baee5758aa';

final class RecentMembersFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<List<ProfileAccountSummary>>,
          ActiveAccountLease
        > {
  RecentMembersFamily._()
    : super(
        retry: null,
        name: r'recentMembersProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  RecentMembersProvider call(ActiveAccountLease lease) =>
      RecentMembersProvider._(argument: lease, from: this);

  @override
  String toString() => r'recentMembersProvider';
}
