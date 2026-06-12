part of '../craftsky_select_inputs.dart';

class CraftskySingleSelectInput<T> extends StatefulWidget {
  const CraftskySingleSelectInput({
    required this.label,
    required this.options,
    super.key,
    this.value,
    this.helperText,
    this.errorText,
    this.enabled = true,
    this.searchThreshold = 5,
    this.searchHintText = 'Search',
    this.noResultsText = 'No results',
    this.keyPrefix,
    this.onChanged,
  });

  final String label;
  final List<CraftskySelectOption<T>> options;
  final T? value;
  final String? helperText;
  final String? errorText;
  final bool enabled;
  final int searchThreshold;
  final String searchHintText;
  final String noResultsText;
  final String? keyPrefix;
  final ValueChanged<T?>? onChanged;

  @override
  State<CraftskySingleSelectInput<T>> createState() =>
      _CraftskySingleSelectInputState<T>();
}

class _CraftskySingleSelectInputState<T>
    extends State<CraftskySingleSelectInput<T>> {
  final _focusNode = FocusNode();
  final _searchFocusNode = FocusNode();
  final _searchController = TextEditingController();
  final _optionsScrollController = ScrollController();
  final GlobalKey _anchorKey = GlobalKey();
  final GlobalKey _optionsViewportKey = GlobalKey();
  final LayerLink _layerLink = LayerLink();
  final _optionVisibilityKeys = <Object?, GlobalKey>{};
  OverlayEntry? _overlayEntry;
  ScrollPosition? _overlayScrollPosition;
  bool _open = false;
  String _query = '';
  int _highlightedIndex = 0;

  bool get _searchable => widget.options.length > widget.searchThreshold;

  CraftskySelectOption<T>? get _selectedOption {
    for (final option in widget.options) {
      if (option.value == widget.value) return option;
    }
    return null;
  }

  List<CraftskySelectOption<T>> get _filteredOptions {
    final query = _query.trim().toLowerCase();
    if (!_searchable) return widget.options;
    if (query.isEmpty) return widget.options;
    return widget.options
        .where((option) => option.label.toLowerCase().contains(query))
        .toList(growable: false);
  }

  @override
  void initState() {
    super.initState();
    _focusNode.addListener(_handleFocusChanged);
    _searchFocusNode.addListener(_handleFocusChanged);
  }

  @override
  void dispose() {
    _removeOverlay();
    _focusNode.removeListener(_handleFocusChanged);
    _searchFocusNode.removeListener(_handleFocusChanged);
    _focusNode.dispose();
    _searchFocusNode.dispose();
    _searchController.dispose();
    _optionsScrollController.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant CraftskySingleSelectInput<T> oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.enabled && !widget.enabled && _open) {
      _setOpen(false);
    }
  }

  void _handleFocusChanged() {
    if (mounted) setState(() {});
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (widget.enabled && _searchFocusNode.hasFocus && !_open) {
        _setOpen(true);
        return;
      }
      if (!_open) return;
      if (!_focusNode.hasFocus && !_searchFocusNode.hasFocus) {
        _setOpen(false);
      }
    });
  }

  void _setOpen(bool open) {
    if (open && !widget.enabled) return;
    final shouldOpen = open;
    if (shouldOpen == _open) return;
    setState(() {
      _open = shouldOpen;
      _highlightedIndex = 0;
      if (!shouldOpen) {
        _query = '';
        _searchController.clear();
      }
    });
    if (shouldOpen) {
      _showOverlay();
    } else {
      _removeOverlay();
    }
  }

  void _showOverlay() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_open) return;
      _removeOverlay();
      _overlayScrollPosition = Scrollable.maybeOf(context)?.position
        ?..addListener(_markOverlayNeedsBuild);
      _overlayEntry = OverlayEntry(
        builder: (context) => _AnchoredSelectOverlay(
          anchorKey: _anchorKey,
          onDismiss: () => _setOpen(false),
          onEscape: _closeOverlayAndRefocus,
          child: _buildMenuContent(),
        ),
      );
      Overlay.of(context).insert(_overlayEntry!);
    });
  }

  void _closeOverlayAndRefocus() {
    _setOpen(false);
    if (_searchable) {
      _searchFocusNode.requestFocus();
    } else {
      _focusNode.requestFocus();
    }
  }

  void _removeOverlay() {
    _overlayScrollPosition?.removeListener(_markOverlayNeedsBuild);
    _overlayScrollPosition = null;
    _overlayEntry?.remove();
    _overlayEntry = null;
  }

  KeyEventResult _handleSearchKey(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    final options = _filteredOptions;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.tab:
        _setOpen(false);
        return KeyEventResult.ignored;
      case LogicalKeyboardKey.escape:
        if (_open) {
          _setOpen(false);
          return KeyEventResult.handled;
        }
      case LogicalKeyboardKey.enter ||
          LogicalKeyboardKey.numpadEnter ||
          LogicalKeyboardKey.select:
        if (options.isNotEmpty) {
          _select(
            options[_highlightedIndex.clamp(0, options.length - 1)].value,
          );
          return KeyEventResult.handled;
        }
      case LogicalKeyboardKey.arrowDown:
        if (!_open) {
          _setOpen(true);
        } else if (options.isNotEmpty) {
          _moveHighlight(1);
        }
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowUp:
        if (!_open) {
          _setOpen(true);
        } else if (options.isNotEmpty) {
          _moveHighlight(-1);
        }
        return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

  void _markOverlayNeedsBuild() {
    _overlayEntry?.markNeedsBuild();
  }

  GlobalKey _optionVisibilityKey(T value) {
    return _optionVisibilityKeys.putIfAbsent(value, GlobalKey.new);
  }

  void _moveHighlight(int delta) {
    final options = _filteredOptions;
    if (options.isEmpty) return;
    setState(() {
      _highlightedIndex = (_highlightedIndex + delta).clamp(
        0,
        options.length - 1,
      );
    });
    _markOverlayNeedsBuild();
    _scrollHighlightedOptionIntoView(movingDown: delta > 0);
  }

  void _scrollHighlightedOptionIntoView({required bool movingDown}) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_open) return;
      final options = _filteredOptions;
      if (options.isEmpty) return;
      final highlightedIndex = _highlightedIndex.clamp(0, options.length - 1);
      final option = options[highlightedIndex];
      if (!_optionsScrollController.hasClients) return;
      final optionBox = _optionVisibilityKeys[option.value]?.currentContext
          ?.findRenderObject();
      final viewportBox = _optionsViewportKey.currentContext
          ?.findRenderObject();
      if (viewportBox is! RenderBox) return;
      if (optionBox is! RenderBox) {
        const estimatedOptionExtent = 56.0;
        const panelPadding = 8.0;
        final viewportHeight = viewportBox.size.height;
        final estimatedTop =
            panelPadding + highlightedIndex * estimatedOptionExtent;
        final estimatedBottom = estimatedTop + estimatedOptionExtent;
        final position = _optionsScrollController.position;
        final targetOffset =
            (movingDown
                    ? estimatedBottom - viewportHeight
                    : estimatedTop - panelPadding)
                .clamp(position.minScrollExtent, position.maxScrollExtent);
        unawaited(
          _optionsScrollController.animateTo(
            targetOffset,
            duration: const Duration(milliseconds: 100),
            curve: Curves.easeOut,
          ),
        );
        return;
      }
      final optionTopLeft = optionBox.localToGlobal(Offset.zero);
      final optionRect = optionTopLeft & optionBox.size;
      final viewportTopLeft = viewportBox.localToGlobal(Offset.zero);
      final viewportRect = viewportTopLeft & viewportBox.size;
      final scrollDelta = switch ((optionRect.top, optionRect.bottom)) {
        (final top, _) when top < viewportRect.top => top - viewportRect.top,
        (_, final bottom) when bottom > viewportRect.bottom =>
          bottom - viewportRect.bottom,
        _ => 0.0,
      };
      if (scrollDelta == 0) return;
      final position = _optionsScrollController.position;
      final targetOffset = (position.pixels + scrollDelta).clamp(
        position.minScrollExtent,
        position.maxScrollExtent,
      );
      unawaited(
        _optionsScrollController.animateTo(
          targetOffset,
          duration: const Duration(milliseconds: 100),
          curve: Curves.easeOut,
        ),
      );
    });
  }

  void _select(T value) {
    widget.onChanged?.call(value);
    _setOpen(false);
    if (_searchable) {
      _searchFocusNode.requestFocus();
    } else {
      _focusNode.requestFocus();
    }
  }

  KeyEventResult _handleKey(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent || !widget.enabled) {
      return KeyEventResult.ignored;
    }
    final options = _filteredOptions;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.enter || LogicalKeyboardKey.space:
        if (!_open) {
          _setOpen(true);
          return KeyEventResult.handled;
        }
        if (options.isNotEmpty) {
          _select(
            options[_highlightedIndex.clamp(0, options.length - 1)].value,
          );
        }
        return KeyEventResult.handled;
      case LogicalKeyboardKey.numpadEnter || LogicalKeyboardKey.select:
        if (!_open) {
          _setOpen(true);
          return KeyEventResult.handled;
        }
        if (options.isNotEmpty) {
          _select(
            options[_highlightedIndex.clamp(0, options.length - 1)].value,
          );
        }
        return KeyEventResult.handled;
      case LogicalKeyboardKey.escape:
        if (_open) {
          _setOpen(false);
          return KeyEventResult.handled;
        }
      case LogicalKeyboardKey.arrowDown:
        if (!_open) {
          _setOpen(true);
        } else if (options.isNotEmpty) {
          _moveHighlight(1);
        }
        return KeyEventResult.handled;
      case LogicalKeyboardKey.arrowUp:
        if (!_open) {
          _setOpen(true);
        } else if (options.isNotEmpty) {
          _moveHighlight(-1);
        }
        return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

  @override
  Widget build(BuildContext context) {
    final selectedLabel = _selectedOption?.label;
    final emptySelectionText = 'Select ${widget.label}';
    final keyPrefix = widget.keyPrefix ?? widget.label;
    return CraftskyFieldScaffold(
      label: widget.label,
      focusNode: _searchable ? _searchFocusNode : _focusNode,
      helperText: widget.helperText,
      errorText: widget.errorText,
      enabled: widget.enabled,
      semanticValue: selectedLabel ?? 'No selection',
      semanticHint: _open ? 'Expanded' : 'Collapsed',
      child: CompositedTransformTarget(
        link: _layerLink,
        child: KeyedSubtree(
          key: _anchorKey,
          child: _searchable
              ? InputDecorator(
                  key: Key('$keyPrefix-select-button'),
                  isFocused: _searchFocusNode.hasFocus,
                  decoration: InputDecoration(
                    enabled: widget.enabled,
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 14,
                    ),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Focus(
                          skipTraversal: true,
                          descendantsAreFocusable: true,
                          descendantsAreTraversable: true,
                          onKeyEvent: _handleSearchKey,
                          child: TextField(
                            key: Key('$keyPrefix-search-input'),
                            controller: _searchController,
                            focusNode: _searchFocusNode,
                            enabled: widget.enabled,
                            decoration: InputDecoration.collapsed(
                              hintText: selectedLabel ?? emptySelectionText,
                              hintStyle: selectedLabel == null
                                  ? Theme.of(
                                      context,
                                    ).textTheme.bodyLarge?.copyWith(
                                      color: Theme.of(
                                        context,
                                      ).colorScheme.onSurfaceVariant,
                                    )
                                  : Theme.of(context).textTheme.bodyLarge,
                            ),
                            onTap: () => _setOpen(true),
                            onChanged: (value) {
                              setState(() {
                                _query = value;
                                _highlightedIndex = 0;
                              });
                              _setOpen(true);
                              _markOverlayNeedsBuild();
                            },
                          ),
                        ),
                      ),
                      const Icon(Icons.search),
                    ],
                  ),
                )
              : Focus(
                  focusNode: _focusNode,
                  canRequestFocus: widget.enabled,
                  descendantsAreFocusable: false,
                  descendantsAreTraversable: false,
                  onKeyEvent: _handleKey,
                  child: InkWell(
                    key: Key('$keyPrefix-select-button'),
                    canRequestFocus: false,
                    onTap: widget.enabled
                        ? () {
                            _focusNode.requestFocus();
                            _setOpen(!_open);
                          }
                        : null,
                    child: InputDecorator(
                      isFocused: _focusNode.hasFocus,
                      decoration: InputDecoration(
                        enabled: widget.enabled,
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 14,
                        ),
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(
                              selectedLabel ?? emptySelectionText,
                              style: Theme.of(context).textTheme.bodyLarge
                                  ?.copyWith(
                                    color: selectedLabel == null
                                        ? Theme.of(
                                            context,
                                          ).colorScheme.onSurfaceVariant
                                        : null,
                                  ),
                            ),
                          ),
                          Icon(_open ? Icons.expand_less : Icons.expand_more),
                        ],
                      ),
                    ),
                  ),
                ),
        ),
      ),
    );
  }

  Widget _buildMenuContent() {
    final options = _filteredOptions;
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final selectedTileColor = theme.colorScheme.primaryContainer.withValues(
      alpha: 0.42,
    );
    final selectedTileShape = RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(radii.r2),
    );
    return _CraftskyOptionsPanel(
      key: Key('${widget.keyPrefix ?? widget.label}-options-panel'),
      scrollable: false,
      child: ListView(
        key: _optionsViewportKey,
        controller: _optionsScrollController,
        shrinkWrap: true,
        primary: false,
        padding: const EdgeInsets.all(8),
        children: [
          if (options.isEmpty)
            ListTile(title: Text(widget.noResultsText))
          else
            for (final (index, option) in options.indexed)
              Container(
                key: _optionVisibilityKey(option.value),
                child: ListTile(
                  key: Key(
                    _optionKey(widget.keyPrefix, widget.label, option.value),
                  ),
                  selected: index == _highlightedIndex,
                  selectedTileColor: selectedTileColor,
                  shape: selectedTileShape,
                  title: Text(option.label),
                  subtitle: option.description == null
                      ? null
                      : Text(option.description!),
                  onTap: () => _select(option.value),
                ),
              ),
        ],
      ),
    );
  }
}
