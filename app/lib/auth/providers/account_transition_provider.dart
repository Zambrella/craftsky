import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_transition_provider.g.dart';

@Riverpod(keepAlive: true)
class AccountTransitionState extends _$AccountTransitionState {
  @override
  AccountTransition? build() => null;

  AccountTransition? get transition => state;

  set transition(AccountTransition? value) => state = value;
}
