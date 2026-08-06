// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_pins_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfilePins)
final profilePinsProvider = ProfilePinsFamily._();

final class ProfilePinsProvider
    extends $AsyncNotifierProvider<ProfilePins, ProfilePinsPresentation> {
  ProfilePinsProvider._({
    required ProfilePinsFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'profilePinsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profilePinsHash();

  @override
  String toString() {
    return r'profilePinsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfilePins create() => ProfilePins();

  @override
  bool operator ==(Object other) {
    return other is ProfilePinsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profilePinsHash() => r'1d2f86ef60cd70b54e616de71c7ca5691e1be3b5';

final class ProfilePinsFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfilePins,
          AsyncValue<ProfilePinsPresentation>,
          ProfilePinsPresentation,
          FutureOr<ProfilePinsPresentation>,
          ActiveAccountLease
        > {
  ProfilePinsFamily._()
    : super(
        retry: null,
        name: r'profilePinsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfilePinsProvider call(ActiveAccountLease accountLease) =>
      ProfilePinsProvider._(argument: accountLease, from: this);

  @override
  String toString() => r'profilePinsProvider';
}

abstract class _$ProfilePins extends $AsyncNotifier<ProfilePinsPresentation> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get accountLease => _$args;

  FutureOr<ProfilePinsPresentation> build(ActiveAccountLease accountLease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<ProfilePinsPresentation>,
              ProfilePinsPresentation
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<ProfilePinsPresentation>,
                ProfilePinsPresentation
              >,
              AsyncValue<ProfilePinsPresentation>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
