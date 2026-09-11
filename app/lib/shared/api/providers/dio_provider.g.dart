// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'dio_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(anonymousDio)
final anonymousDioProvider = AnonymousDioProvider._();

final class AnonymousDioProvider extends $FunctionalProvider<Dio, Dio, Dio>
    with $Provider<Dio> {
  AnonymousDioProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'anonymousDioProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$anonymousDioHash();

  @$internal
  @override
  $ProviderElement<Dio> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Dio create(Ref ref) {
    return anonymousDio(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Dio value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Dio>(value),
    );
  }
}

String _$anonymousDioHash() => r'52cb91b88d5e577ad2094eceb63670047afd3219';

@ProviderFor(accountDio)
final accountDioProvider = AccountDioFamily._();

final class AccountDioProvider
    extends $FunctionalProvider<AsyncValue<Dio>, Dio, FutureOr<Dio>>
    with $FutureModifier<Dio>, $FutureProvider<Dio> {
  AccountDioProvider._({
    required AccountDioFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountDioProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountDioHash();

  @override
  String toString() {
    return r'accountDioProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<Dio> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<Dio> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountDio(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountDioProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountDioHash() => r'8816f33ed9739baa416412fb858a3d71582fa50b';

final class AccountDioFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<Dio>, AccountKey> {
  AccountDioFamily._()
    : super(
        retry: null,
        name: r'accountDioProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountDioProvider call(AccountKey account) =>
      AccountDioProvider._(argument: account, from: this);

  @override
  String toString() => r'accountDioProvider';
}

@ProviderFor(dio)
final dioProvider = DioProvider._();

final class DioProvider extends $FunctionalProvider<Dio, Dio, Dio>
    with $Provider<Dio> {
  DioProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'dioProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$dioHash();

  @$internal
  @override
  $ProviderElement<Dio> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Dio create(Ref ref) {
    return dio(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Dio value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Dio>(value),
    );
  }
}

String _$dioHash() => r'8f66433de4d3cddc766330a6d1b95fd3dce3e5a4';
