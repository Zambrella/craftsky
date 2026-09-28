import 'package:craftsky_app/moderation/models/account_moderation.dart';

abstract interface class ModerationRepository {
  Future<AccountStanding> getStanding();

  Future<ModerationHistoryPage> getHistory({String? cursor, int? limit});

  Future<ModerationHistoryPage> getHistoryEntry(
    ModerationCaseReference reference,
  );
}
