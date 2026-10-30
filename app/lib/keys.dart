import 'package:craftsky_app/auth/keys.dart';
import 'package:craftsky_app/feed/keys.dart';
import 'package:craftsky_app/feed/widgets/keys.dart';
import 'package:craftsky_app/onboarding/keys.dart';
import 'package:craftsky_app/shared/image/keys.dart';
import 'package:craftsky_app/shared/media/keys.dart';
import 'package:craftsky_app/theme/keys.dart';

final keys = Keys();

class Keys {
  final auth = AuthKeys();
  final composerImages = ComposerImageKeys();
  final dialog = DialogKeys();
  final feed = FeedKeys();
  final imageEditor = ImageEditorKeys();
  final imageSource = ImageSourceKeys();
  final onboarding = OnboardingKeys();
}
