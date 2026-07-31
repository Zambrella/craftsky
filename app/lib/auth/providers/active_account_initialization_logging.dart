import 'package:logging/logging.dart';

final _log = Logger('ActiveAccountInitialization');

/// Records an account initialization failure without attaching account data or
/// the underlying error.
void logActiveAccountInitializationFailure() {
  _log.severe('Active account failed to initialize');
}
