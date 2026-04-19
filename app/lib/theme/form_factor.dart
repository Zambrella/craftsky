import 'package:flutter/material.dart';

enum FormFactor {
  mobile(breakpoint: 600),
  tablet(breakpoint: 900),
  laptop(breakpoint: 1200),
  desktop(breakpoint: double.infinity)
  ;

  const FormFactor({required this.breakpoint});

  final double breakpoint;

  bool get isSmall => this == FormFactor.mobile || this == FormFactor.tablet;
  bool get isLarge => this == FormFactor.laptop || this == FormFactor.desktop;

  static FormFactor fromWidth(double width) {
    if (width <= FormFactor.mobile.breakpoint) return FormFactor.mobile;
    if (width <= FormFactor.tablet.breakpoint) return FormFactor.tablet;
    if (width <= FormFactor.laptop.breakpoint) return FormFactor.laptop;
    return FormFactor.desktop;
  }
}

/// Provides the current [FormFactor] to the subtree.
///
/// Recomputes on every build so orientation and resize changes propagate.
class FormFactorWidget extends StatelessWidget {
  const FormFactorWidget({required this.child, super.key});

  final Widget child;

  static FormFactor of(BuildContext context) {
    final scope = context
        .dependOnInheritedWidgetOfExactType<_FormFactorScope>();
    assert(scope != null, 'No FormFactorWidget found in context');
    return scope!.formFactor;
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final formFactor = FormFactor.fromWidth(width);
    return _FormFactorScope(formFactor: formFactor, child: child);
  }
}

class _FormFactorScope extends InheritedWidget {
  const _FormFactorScope({required this.formFactor, required super.child});

  final FormFactor formFactor;

  @override
  bool updateShouldNotify(_FormFactorScope oldWidget) =>
      formFactor != oldWidget.formFactor;
}
