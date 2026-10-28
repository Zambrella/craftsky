import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';

/// A published editor has a fixed record/base and activation authority. It is
/// separate from local draft and scheduled publication origins.
final class PostEditOrigin {
  const new({required this.seed, required this.ownership});

  final PostEditSeed seed;
  final ActiveAccountLease ownership;
}
