/// Account-bound work discarded after its original authority ended.
abstract interface class AccountOperationCancellation;

/// Work whose account authority ended while it was pending.
///
/// Keeps StateError catch semantics for callers while allowing diagnostics to
/// distinguish normal account transitions from broken application invariants.
final class AccountOperationCanceled extends StateError
    implements AccountOperationCancellation {
  new() : super('Account operation canceled');
}
