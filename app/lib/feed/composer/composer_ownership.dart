import 'package:craftsky_app/auth/models/account_operation_canceled.dart';

/// A known ownership fence, retaining StateError catch compatibility while
/// distinguishing intentional account retirement from Riverpod misuse.
final class ComposerOwnershipChanged extends StateError
    implements AccountOperationCancellation {
  new() : super('submission ownership changed');
}

/// Preserve the same content-free failure at each submission/media boundary.
void requireComposerOwnership(bool Function() isCurrent) {
  if (!isCurrent()) throw ComposerOwnershipChanged();
}
