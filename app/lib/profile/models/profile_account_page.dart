import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'profile_account_page.mapper.dart';

@MappableClass()
class ProfileAccountPage with ProfileAccountPageMappable {
  const ProfileAccountPage({
    required this.items,
    required this.totalCount,
    this.cursor,
  });

  final List<ProfileAccountSummary> items;
  final String? cursor;
  final int totalCount;
}
