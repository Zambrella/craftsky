// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'instagram_verification_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(InstagramVerification)
final instagramVerificationProvider = InstagramVerificationFamily._();

final class InstagramVerificationProvider
    extends
        $NotifierProvider<
          InstagramVerification,
          InstagramVerificationViewState
        > {
  InstagramVerificationProvider._({
    required InstagramVerificationFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'instagramVerificationProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$instagramVerificationHash();

  @override
  String toString() {
    return r'instagramVerificationProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  InstagramVerification create() => InstagramVerification();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(InstagramVerificationViewState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<InstagramVerificationViewState>(
        value,
      ),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is InstagramVerificationProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$instagramVerificationHash() =>
    r'd7f447ee705595df010c23b53ccd34fb8e1656ed';

final class InstagramVerificationFamily extends $Family
    with
        $ClassFamilyOverride<
          InstagramVerification,
          InstagramVerificationViewState,
          InstagramVerificationViewState,
          InstagramVerificationViewState,
          ActiveAccountLease
        > {
  InstagramVerificationFamily._()
    : super(
        retry: null,
        name: r'instagramVerificationProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  InstagramVerificationProvider call(ActiveAccountLease lease) =>
      InstagramVerificationProvider._(argument: lease, from: this);

  @override
  String toString() => r'instagramVerificationProvider';
}

abstract class _$InstagramVerification
    extends $Notifier<InstagramVerificationViewState> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get lease => _$args;

  InstagramVerificationViewState build(ActiveAccountLease lease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              InstagramVerificationViewState,
              InstagramVerificationViewState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                InstagramVerificationViewState,
                InstagramVerificationViewState
              >,
              InstagramVerificationViewState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
