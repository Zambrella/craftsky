import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'recent_members_provider.g.dart';

@riverpod
Future<List<ProfileAccountSummary>> recentMembers(
  Ref ref,
  ActiveAccountLease lease,
) async {
  final operation = ActiveAccountOperation.fromLease(ref, lease);
  if (!operation.isCurrent) throw AccountOperationCanceled();
  final repository = await ref.watch(
    accountProfileRepositoryProvider(lease).future,
  );
  if (!ref.mounted) throw AccountOperationCanceled();
  if (!operation.isCurrent) throw AccountOperationCanceled();
  final members = await repository.fetchRecentMembers();
  if (!ref.mounted) throw AccountOperationCanceled();
  if (!operation.isCurrent) throw AccountOperationCanceled();
  final controller = ref.read(pdsRecordOperationControllerProvider);
  return [
    for (final member in members)
      // Discovery omits relationship flags. Its default false cannot reconcile
      // a PDS-accepted block while the AppView projection is still catching up.
      if (controller
              .overlayFor(
                blockProfileMutationScopeFor(lease.session, member.did),
              )
              ?.optimisticValue !=
          true)
        member,
  ];
}
