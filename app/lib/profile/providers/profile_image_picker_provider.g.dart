// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_image_picker_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountProfileImagePicker)
final accountProfileImagePickerProvider = AccountProfileImagePickerFamily._();

final class AccountProfileImagePickerProvider
    extends
        $FunctionalProvider<
          AsyncValue<ProfileImagePicker>,
          ProfileImagePicker,
          FutureOr<ProfileImagePicker>
        >
    with
        $FutureModifier<ProfileImagePicker>,
        $FutureProvider<ProfileImagePicker> {
  AccountProfileImagePickerProvider._({
    required AccountProfileImagePickerFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'accountProfileImagePickerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountProfileImagePickerHash();

  @override
  String toString() {
    return r'accountProfileImagePickerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ProfileImagePicker> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ProfileImagePicker> create(Ref ref) {
    final argument = this.argument as ActiveAccountLease;
    return accountProfileImagePicker(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountProfileImagePickerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountProfileImagePickerHash() =>
    r'368ca24b3ff1f0b809452e679f16169bb3a5afcf';

final class AccountProfileImagePickerFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ProfileImagePicker>,
          ActiveAccountLease
        > {
  AccountProfileImagePickerFamily._()
    : super(
        retry: null,
        name: r'accountProfileImagePickerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountProfileImagePickerProvider call(ActiveAccountLease lease) =>
      AccountProfileImagePickerProvider._(argument: lease, from: this);

  @override
  String toString() => r'accountProfileImagePickerProvider';
}
