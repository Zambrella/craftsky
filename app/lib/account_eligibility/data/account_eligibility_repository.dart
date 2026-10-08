import 'package:craftsky_app/account_eligibility/models/account_eligibility_status.dart';

// Keep the repository as an injectable boundary even though it currently has
// one operation.
// ignore: one_member_abstracts
abstract interface class AccountEligibilityRepository {
  Future<AccountEligibilityStatus> readStatus();
}
