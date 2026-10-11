/// Work whose account authority ended while it was pending.
///
/// Keeps StateError catch semantics for callers while allowing diagnostics to
/// distinguish normal account transitions from broken application invariants.
final class AccountOperationCanceled extends StateError {
  AccountOperationCanceled() : super('Account operation canceled');
}
