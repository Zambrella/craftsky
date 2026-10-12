// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'instagram_account_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(InstagramAccount)
final instagramAccountProvider = InstagramAccountFamily._();

final class InstagramAccountProvider
    extends $AsyncNotifierProvider<InstagramAccount, InstagramAccountStatus> {
  InstagramAccountProvider._({
    required InstagramAccountFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'instagramAccountProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$instagramAccountHash();

  @override
  String toString() {
    return r'instagramAccountProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  InstagramAccount create() => InstagramAccount();

  @override
  bool operator ==(Object other) {
    return other is InstagramAccountProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$instagramAccountHash() => r'a5ad24e0973bfb4650612b8e22f12e3ce29c8e21';

final class InstagramAccountFamily extends $Family
    with
        $ClassFamilyOverride<
          InstagramAccount,
          AsyncValue<InstagramAccountStatus>,
          InstagramAccountStatus,
          FutureOr<InstagramAccountStatus>,
          ActiveAccountLease
        > {
  InstagramAccountFamily._()
    : super(
        retry: null,
        name: r'instagramAccountProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  InstagramAccountProvider call(ActiveAccountLease lease) =>
      InstagramAccountProvider._(argument: lease, from: this);

  @override
  String toString() => r'instagramAccountProvider';
}

abstract class _$InstagramAccount
    extends $AsyncNotifier<InstagramAccountStatus> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get lease => _$args;

  FutureOr<InstagramAccountStatus> build(ActiveAccountLease lease);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<InstagramAccountStatus>, InstagramAccountStatus>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<InstagramAccountStatus>,
                InstagramAccountStatus
              >,
              AsyncValue<InstagramAccountStatus>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
