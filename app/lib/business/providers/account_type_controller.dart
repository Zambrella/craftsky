import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_type_controller.g.dart';

typedef AccountTypeReconciler = void Function(AccountType accountType);

@Riverpod(keepAlive: true)
AccountTypeReconciler accountTypeProfileReconciler(Ref ref) => (accountType) {
  final profile = ref.read(activeAccountIdentityProvider).value?.profile;
  if (profile == null) return;

  final updated = profile.copyWith(
    accountType: accountType,
    business: accountType == AccountType.regular ? null : profile.business,
  );
  for (final id in <String>{profile.handle.value, profile.did.value}) {
    final provider = userProfileProvider(id);
    if (ref.exists(provider)) {
      ref.read(provider.notifier).setCached(updated);
    }
  }
};

@Riverpod(keepAlive: true)
AccountTypeReconciler accountTypeStateInvalidator(Ref ref) => (accountType) {
  final profile = ref.read(activeAccountIdentityProvider).value?.profile;
  if (profile == null) return;

  for (final id in <String>{profile.handle.value, profile.did.value}) {
    ref.invalidate(userProfileProvider(id));
  }
  ref.invalidate(activeAccountIdentityProvider);
};

@Riverpod(keepAlive: true)
class AccountTypeController extends _$AccountTypeController {
  @override
  FutureOr<AccountType?> build() =>
      ref.read(activeAccountIdentityProvider).value?.profile.accountType;

  Future<bool> setAccountType(AccountType requested) async {
    if (state.isLoading) return false;

    final confirmed =
        state.value ??
        ref.read(activeAccountIdentityProvider).value?.profile.accountType;
    state = const AsyncLoading<AccountType?>();
    try {
      final ownership = captureActiveAccountOperation(ref);
      final returned = await ref
          .read(businessRepositoryProvider)
          .updateAccountType(requested);
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;

      ref.read(accountTypeProfileReconcilerProvider)(returned);
      ref.read(accountTypeStateInvalidatorProvider)(returned);
      state = AsyncData(returned);
      return true;
    } on Object {
      if (!ref.mounted) return false;
      state = AsyncData(confirmed);
      return false;
    }
  }
}
