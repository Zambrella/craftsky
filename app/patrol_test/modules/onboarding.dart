import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:patrol/patrol.dart';

final class OnboardingModule {
  new(this.$);

  final PatrolIntegrationTester $;

  Future<void> completeWithBio(String bio) async {
    await $(keys.onboarding.bio).scrollTo();
    await $(keys.onboarding.bio).enterText(bio);
    await $(keys.onboarding.next(OnboardingActionKind.saveAndNext))
        .which<ChunkyButton>((button) => button.onPressed != null)
        .tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.onboarding.next(OnboardingActionKind.next))
        .which<ChunkyButton>((button) => button.onPressed != null)
        .tap(settlePolicy: SettlePolicy.noSettle);
    await $(keys.onboarding.minimumAge).scrollTo();
    await $(keys.onboarding.minimumAge).tap();
    await $(keys.onboarding.next(OnboardingActionKind.finish))
        .which<ChunkyButton>((button) => button.onPressed != null)
        .tap(settlePolicy: SettlePolicy.noSettle);
  }
}
