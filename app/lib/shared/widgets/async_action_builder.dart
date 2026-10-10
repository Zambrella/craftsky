import 'dart:async';

import 'package:flutter/material.dart';

/// Local pending state for a group of actions sharing an interaction lock.
/// The caller keeps ownership of error presentation and operation semantics.
class AsyncActionBuilder extends StatefulWidget {
  const AsyncActionBuilder({required this.builder, super.key});

  final Widget Function(BuildContext context, AsyncActionState actions) builder;

  @override
  State<AsyncActionBuilder> createState() => AsyncActionState();
}

class AsyncActionState extends State<AsyncActionBuilder> {
  Enum? _pending;
  Object? _pendingTarget;

  bool get isBusy => _pending != null;
  bool isRunning(Enum action, {Object? target}) =>
      _pending == action && _pendingTarget == target;

  Future<void> run(
    Enum action,
    FutureOr<void> Function() operation, {
    Object? target,
  }) async {
    if (isBusy || !mounted) return;
    setState(() {
      _pending = action;
      _pendingTarget = target;
    });
    try {
      await operation();
    } finally {
      if (mounted) {
        setState(() {
          _pending = null;
          _pendingTarget = null;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) => widget.builder(context, this);
}
