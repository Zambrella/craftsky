// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'instagram_migration_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(instagramMigrationRepository)
final instagramMigrationRepositoryProvider =
    InstagramMigrationRepositoryFamily._();

final class InstagramMigrationRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<InstagramMigrationRepository>,
          InstagramMigrationRepository,
          FutureOr<InstagramMigrationRepository>
        >
    with
        $FutureModifier<InstagramMigrationRepository>,
        $FutureProvider<InstagramMigrationRepository> {
  InstagramMigrationRepositoryProvider._({
    required InstagramMigrationRepositoryFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'instagramMigrationRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$instagramMigrationRepositoryHash();

  @override
  String toString() {
    return r'instagramMigrationRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<InstagramMigrationRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<InstagramMigrationRepository> create(Ref ref) {
    final argument = this.argument as ActiveAccountLease;
    return instagramMigrationRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is InstagramMigrationRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$instagramMigrationRepositoryHash() =>
    r'758ec948e63d69a8a758ee5f81f7547b1bb3150b';

final class InstagramMigrationRepositoryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<InstagramMigrationRepository>,
          ActiveAccountLease
        > {
  InstagramMigrationRepositoryFamily._()
    : super(
        retry: null,
        name: r'instagramMigrationRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  InstagramMigrationRepositoryProvider call(ActiveAccountLease lease) =>
      InstagramMigrationRepositoryProvider._(argument: lease, from: this);

  @override
  String toString() => r'instagramMigrationRepositoryProvider';
}
