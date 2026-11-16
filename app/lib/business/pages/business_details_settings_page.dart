import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:material_ui/material_ui.dart';

/// Reuses the versioned business form and its conflict/discard handling while
/// keeping business writes separate from personal profile editing.
class BusinessDetailsSettingsPage extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) =>
      const EditProfileDialog(businessDetails: true);
}
