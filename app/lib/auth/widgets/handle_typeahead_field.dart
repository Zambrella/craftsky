import 'dart:async';

import 'package:craftsky_app/auth/providers/handle_typeahead_provider.dart';
import 'package:craftsky_app/auth/services/handle_typeahead_service.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Optional public identity suggestions; never resolves or authenticates users.
class HandleTypeaheadField extends ConsumerStatefulWidget {
  const HandleTypeaheadField({
    required this.controller,
    required this.onSubmitted,
    super.key,
    this.hintText = 'alice.bsky.social',
    this.enabled = true,
  });

  final TextEditingController controller;
  final ValueChanged<String> onSubmitted;
  final String hintText;
  final bool enabled;

  @override
  ConsumerState<HandleTypeaheadField> createState() =>
      _HandleTypeaheadFieldState();
}

class _HandleTypeaheadFieldState extends ConsumerState<HandleTypeaheadField> {
  final _focusNode = FocusNode();
  final GlobalKey _anchorKey = GlobalKey();
  final _overlayController = OverlayPortalController();
  ScrollPosition? _scrollPosition;
  Timer? _debounce;
  CancelToken? _request;
  int _generation = 0;
  String _lastText = '';
  bool _wasComposing = false;
  List<HandleSuggestion> _suggestions = [];
  List<GlobalKey> _suggestionKeys = [];
  int _highlight = 0;

  @override
  void initState() {
    super.initState();
    _lastText = widget.controller.text;
    _wasComposing = !widget.controller.value.composing.isCollapsed;
    widget.controller.addListener(_onTextChanged);
    _focusNode.addListener(_onFocusChanged);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _scrollPosition?.removeListener(_onScroll);
    _scrollPosition = Scrollable.maybeOf(context)?.position;
    _scrollPosition?.addListener(_onScroll);
  }

  void _onScroll() {
    if (_overlayController.isShowing) setState(() {});
  }

  @override
  void didUpdateWidget(covariant HandleTypeaheadField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.removeListener(_onTextChanged);
      widget.controller.addListener(_onTextChanged);
      _lastText = widget.controller.text;
      _wasComposing = !widget.controller.value.composing.isCollapsed;
      _clear();
    }
    if (oldWidget.enabled && !widget.enabled) {
      _clear();
      _focusNode.unfocus();
    }
  }

  void _cancel() {
    _generation++;
    _debounce?.cancel();
    _request?.cancel();
    _request = null;
  }

  void _clear() {
    _cancel();
    setState(() {
      _suggestions = [];
      _suggestionKeys = [];
      _highlight = 0;
    });
    _updateOverlay();
  }

  void _updateOverlay() {
    if (SchedulerBinding.instance.schedulerPhase ==
        SchedulerPhase.persistentCallbacks) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _updateOverlay();
      });
      return;
    }
    if (widget.enabled && _focusNode.hasFocus && _suggestions.isNotEmpty) {
      _overlayController.show();
    } else {
      _overlayController.hide();
    }
  }

  void _onFocusChanged() {
    if (_focusNode.hasFocus) {
      _schedule();
    } else {
      _clear();
    }
  }

  void _onTextChanged() {
    final composing = !widget.controller.value.composing.isCollapsed;
    if (_lastText == widget.controller.text && _wasComposing == composing) {
      return;
    }
    _lastText = widget.controller.text;
    _wasComposing = composing;
    _schedule();
  }

  void _schedule() {
    _clear();
    final query = widget.controller.text.trim().replaceFirst(RegExp('^@'), '');
    if (!widget.enabled ||
        !_focusNode.hasFocus ||
        query.length < 2 ||
        !widget.controller.value.composing.isCollapsed) {
      return;
    }
    final generation = _generation;
    _debounce = Timer(const Duration(milliseconds: 300), () {
      unawaited(_search(query, generation));
    });
  }

  Future<void> _search(String query, int generation) async {
    final token = CancelToken();
    _request = token;
    try {
      final suggestions = await ref
          .read(handleTypeaheadServiceProvider)
          .search(query, cancelToken: token);
      if (!mounted || generation != _generation) return;
      setState(() {
        _suggestions = suggestions;
        _suggestionKeys = [
          for (final _ in suggestions) GlobalKey(),
        ];
        _highlight = 0;
      });
      _updateOverlay();
    } on Object {
      // Suggestions are best-effort. Manual sign-in must remain available.
    }
  }

  void _select(HandleSuggestion suggestion) {
    final handle = suggestion.handle.toString();
    _lastText = handle;
    _wasComposing = false;
    widget.controller.value = TextEditingValue(
      text: handle,
      selection: TextSelection.collapsed(offset: handle.length),
    );
    _clear();
    _focusNode.requestFocus();
  }

  void _submit(String value) {
    if (_suggestions.isNotEmpty) {
      _select(_suggestions[_highlight]);
    } else {
      _clear();
      widget.onSubmitted(value);
    }
  }

  void _moveHighlight(int delta) {
    setState(() {
      _highlight = (_highlight + delta) % _suggestions.length;
    });
    final key = _suggestionKeys[_highlight];
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final optionContext = key.currentContext;
      if (!mounted || optionContext == null) return;
      unawaited(
        Scrollable.ensureVisible(
          optionContext,
          duration: const Duration(milliseconds: 100),
          alignment: 0.5,
        ),
      );
    });
  }

  KeyEventResult _onKeyEvent(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent || _suggestions.isEmpty) {
      return KeyEventResult.ignored;
    }
    if (event.logicalKey == LogicalKeyboardKey.escape) {
      _clear();
    } else if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
      _moveHighlight(1);
    } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
      _moveHighlight(-1);
    } else if (event.logicalKey == LogicalKeyboardKey.enter ||
        event.logicalKey == LogicalKeyboardKey.numpadEnter) {
      _select(_suggestions[_highlight]);
    } else {
      return KeyEventResult.ignored;
    }
    return KeyEventResult.handled;
  }

  @override
  void dispose() {
    _cancel();
    _scrollPosition?.removeListener(_onScroll);
    widget.controller.removeListener(_onTextChanged);
    _focusNode
      ..removeListener(_onFocusChanged)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Keep the auto-dispose transport alive for the lifetime of this field,
    // including while a debounced request is in flight.
    ref.watch(handleTypeaheadServiceProvider);
    return Focus(
      canRequestFocus: false,
      onKeyEvent: _onKeyEvent,
      child: OverlayPortal(
        controller: _overlayController,
        overlayChildBuilder: (context) => MediaQuery.fromView(
          // Scaffold removes keyboard insets from its body. The overlay needs
          // live view geometry to avoid drawing behind the keyboard.
          view: View.of(context),
          child: CraftskyAnchoredSelectOverlay(
            anchorKey: _anchorKey,
            onDismiss: _clear,
            onEscape: _clear,
            child: TextFieldTapRegion(
              child: CraftskyOptionsPanel(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    for (var index = 0; index < _suggestions.length; index++)
                      ListTile(
                        key: _suggestionKeys[index],
                        dense: true,
                        selected: index == _highlight,
                        leading: ExcludeSemantics(
                          child: ProfileAvatar(
                            seed:
                                _suggestions[index].displayName ??
                                _suggestions[index].handle.toString(),
                            avatarUrl: _suggestions[index].avatarUrl,
                            size: ProfileAvatarSize.small,
                            showShadow: false,
                          ),
                        ),
                        title: Text(
                          _suggestions[index].displayName ??
                              _suggestions[index].handle.toString(),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: Text(
                          '@${_suggestions[index].handle}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        onTap: () => _select(_suggestions[index]),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ),
        child: BrandTextField(
          key: _anchorKey,
          label: AppLocalizations.of(context).signInHandleLabel,
          hintText: widget.hintText,
          controller: widget.controller,
          focusNode: _focusNode,
          enabled: widget.enabled,
          keyboardType: TextInputType.url,
          textInputAction: TextInputAction.done,
          autofillHints: const [AutofillHints.username],
          autocorrect: false,
          enableSuggestions: false,
          onSubmitted: _submit,
        ),
      ),
    );
  }
}
