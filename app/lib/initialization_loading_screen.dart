import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:material_ui/material_ui.dart';

class InitializationLoadingScreen extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: StitchProgressIndicator()),
    );
  }
}
