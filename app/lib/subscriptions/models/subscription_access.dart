import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'subscription_access.mapper.dart';

const int _subscriptionAccessMethods =
    GenerateMethods.decode | GenerateMethods.copy | GenerateMethods.equals;

@MappableEnum()
enum SubscriptionTier { free, plus, business }

@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [DidMapper()],
  generateMethods: _subscriptionAccessMethods,
)
final class SubscriptionAccess with SubscriptionAccessMappable {
  const SubscriptionAccess({
    required this.did,
    required this.effectiveTier,
    required this.givesAccess,
    this.accessEndsAt,
    this.assignedTier,
  });

  factory SubscriptionAccess.fromMap(Map<String, dynamic> map) {
    final access = SubscriptionAccessMapper.fromMap(map);
    final assignedTier = access.assignedTier;
    final valid = assignedTier == null
        ? !access.givesAccess && access.effectiveTier == SubscriptionTier.free
        : assignedTier != SubscriptionTier.free &&
              (access.givesAccess
                  ? access.effectiveTier == assignedTier
                  : access.effectiveTier == SubscriptionTier.free);
    if (!valid) {
      throw const FormatException('Contradictory subscription access');
    }
    return access;
  }

  final Did did;
  final SubscriptionTier effectiveTier;
  final bool givesAccess;
  final DateTime? accessEndsAt;
  final SubscriptionTier? assignedTier;

  @override
  String toString() => 'SubscriptionAccess([REDACTED])';
}
