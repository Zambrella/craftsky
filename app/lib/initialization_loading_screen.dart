import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';

class InitializationLoadingScreen extends StatelessWidget {
  const InitializationLoadingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: StitchProgressIndicator()),
    );
  }
}
