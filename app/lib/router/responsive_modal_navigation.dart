import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';

/// Returns the navigator that should present a full-screen modal.
///
/// Compact modals continue to cover the entire application, including bottom
/// navigation. Large-screen modals stay within the content navigator so the
/// persistent navigation rail remains visible.
NavigatorState responsiveModalNavigator(BuildContext context) => Navigator.of(
  context,
  rootNavigator: FormFactor.fromWidth(MediaQuery.sizeOf(context).width).isSmall,
);
