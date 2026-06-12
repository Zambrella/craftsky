import 'dart:async';

import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';

class CraftskySelectOption<T> {
  const CraftskySelectOption({
    required this.value,
    required this.label,
    this.description,
  });

  final T value;
  final String label;
  final String? description;
}

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
      _overlayEntry = OverlayEntry(
        builder: (context) => _AnchoredSelectOverlay(
          anchorKey: _anchorKey,
          layerLink: _layerLink,
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
    _overlayEntry?.remove();
    _overlayEntry = null;
  }

  KeyEventResult _handleSearchKey(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    final options = _filteredOptions;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.tab:
        _setOpen(false);
        if (HardwareKeyboard.instance.isShiftPressed) {
          _searchFocusNode.previousFocus();
        } else {
          _searchFocusNode.nextFocus();
        }
        return KeyEventResult.handled;
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
                          canRequestFocus: false,
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

class CraftskySearchableMultiSelectInput<T> extends StatefulWidget {
  const CraftskySearchableMultiSelectInput({
    required this.label,
    required this.options,
    super.key,
    this.values = const [],
    this.helperText,
    this.errorText,
    this.enabled = true,
    this.maxSelected,
    this.searchHintText,
    this.disabledText,
    this.maxSelectedErrorText,
    this.keyPrefix,
    this.onChanged,
  });

  final String label;
  final List<CraftskySelectOption<T>> options;
  final List<T> values;
  final String? helperText;
  final String? errorText;
  final bool enabled;
  final int? maxSelected;
  final String? searchHintText;
  final String? disabledText;
  final String? maxSelectedErrorText;
  final String? keyPrefix;
  final ValueChanged<List<T>>? onChanged;

  @override
  State<CraftskySearchableMultiSelectInput<T>> createState() =>
      _CraftskySearchableMultiSelectInputState<T>();
}

class _CraftskySearchableMultiSelectInputState<T>
    extends State<CraftskySearchableMultiSelectInput<T>> {
  final _focusNode = FocusNode();
  final _searchFocusNode = FocusNode();
  final _searchController = TextEditingController();
  final GlobalKey _anchorKey = GlobalKey();
  final LayerLink _layerLink = LayerLink();
  OverlayEntry? _overlayEntry;
  bool _open = false;
  String _query = '';
  String? _limitText;
  late List<T> _values;

  @override
  void initState() {
    super.initState();
    _values = List<T>.from(widget.values);
    _focusNode.addListener(_handleFocusChanged);
    _searchFocusNode.addListener(_handleFocusChanged);
  }

  @override
  void didUpdateWidget(
    covariant CraftskySearchableMultiSelectInput<T> oldWidget,
  ) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.enabled && !widget.enabled && _open) {
      _setOpen(false);
    }
    if (!_listEquals(_values, widget.values)) {
      _values = List<T>.from(widget.values);
      if (_isBelowSelectionLimit(_values.length)) {
        _limitText = null;
      }
      _markOverlayNeedsBuildAfterFrame();
    }
  }

  @override
  void dispose() {
    _removeOverlay();
    _focusNode.removeListener(_handleFocusChanged);
    _searchFocusNode.removeListener(_handleFocusChanged);
    _focusNode.dispose();
    _searchFocusNode.dispose();
    _searchController.dispose();
    super.dispose();
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
    setState(() => _open = shouldOpen);
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
      _overlayEntry = OverlayEntry(
        builder: (context) => _AnchoredSelectOverlay(
          anchorKey: _anchorKey,
          layerLink: _layerLink,
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
    _searchFocusNode.requestFocus();
  }

  void _removeOverlay() {
    _overlayEntry?.remove();
    _overlayEntry = null;
  }

  KeyEventResult _handleSearchKey(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    switch (event.logicalKey) {
      case LogicalKeyboardKey.tab:
        _setOpen(false);
        if (HardwareKeyboard.instance.isShiftPressed) {
          _searchFocusNode.previousFocus();
        } else {
          _searchFocusNode.nextFocus();
        }
        return KeyEventResult.handled;
      case LogicalKeyboardKey.escape:
        if (_open) {
          _setOpen(false);
          return KeyEventResult.handled;
        }
      case LogicalKeyboardKey.enter ||
          LogicalKeyboardKey.numpadEnter ||
          LogicalKeyboardKey.select:
        if (_filteredOptions.isNotEmpty) {
          _toggle(_filteredOptions.first.value);
          return KeyEventResult.handled;
        }
    }
    return KeyEventResult.ignored;
  }

  void _markOverlayNeedsBuild() {
    _overlayEntry?.markNeedsBuild();
  }

  void _markOverlayNeedsBuildAfterFrame() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        _markOverlayNeedsBuild();
      }
    });
  }

  bool _isBelowSelectionLimit(int length) {
    final max = widget.maxSelected;
    return max == null || length < max;
  }

  Map<T, String> get _labelByValue => {
    for (final option in widget.options) option.value: option.label,
  };

  List<CraftskySelectOption<T>> get _filteredOptions {
    final query = _query.trim().toLowerCase();
    if (query.isEmpty) return widget.options;
    return widget.options
        .where((option) => option.label.toLowerCase().contains(query))
        .toList(growable: false);
  }

  void _setValues(List<T> values) {
    final nextValues = List<T>.unmodifiable(values);
    setState(() => _values = nextValues);
    widget.onChanged?.call(nextValues);
    _markOverlayNeedsBuild();
    _searchFocusNode.requestFocus();
  }

  void _toggle(T value) {
    if (!widget.enabled) return;
    final selected = List<T>.from(_values);
    if (selected.contains(value)) {
      selected.remove(value);
      setState(() => _limitText = null);
      _setValues(selected);
      return;
    }
    final max = widget.maxSelected;
    if (max != null && selected.length >= max) {
      setState(() => _limitText = widget.maxSelectedErrorText);
      _markOverlayNeedsBuild();
      return;
    }
    selected.add(value);
    _searchController.clear();
    setState(() {
      _limitText = null;
      _query = '';
    });
    _setValues(selected);
    _setOpen(false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final keyPrefix = widget.keyPrefix ?? widget.label;
    final selectedSummary = _values
        .map((value) => _labelByValue[value] ?? value.toString())
        .join(', ');
    final errorText = widget.errorText ?? _limitText;
    return CraftskyFieldScaffold(
      label: widget.label,
      focusNode: _searchFocusNode,
      helperText: errorText == null ? widget.helperText : null,
      errorText: errorText,
      enabled: widget.enabled,
      semanticValue: selectedSummary.isEmpty
          ? 'No selections'
          : selectedSummary,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          CompositedTransformTarget(
            link: _layerLink,
            child: KeyedSubtree(
              key: _anchorKey,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(radii.r3),
                child: Material(
                  color: Colors.transparent,
                  child: InkWell(
                    key: Key('$keyPrefix-select-button'),
                    canRequestFocus: false,
                    onTap: widget.enabled
                        ? () {
                            _searchFocusNode.requestFocus();
                            _setOpen(true);
                          }
                        : null,
                    child: InputDecorator(
                      isFocused: _searchFocusNode.hasFocus,
                      decoration: InputDecoration(
                        enabled: widget.enabled,
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 10,
                        ),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (_values.isNotEmpty) ...[
                            _SelectedChips<T>(
                              name: widget.label,
                              keyPrefix: keyPrefix,
                              values: _values,
                              labelByValue: _labelByValue,
                              enabled: widget.enabled,
                              onRemove: _toggle,
                            ),
                            const SizedBox(height: 8),
                          ],
                          Row(
                            children: [
                              Expanded(
                                child: Focus(
                                  canRequestFocus: false,
                                  onKeyEvent: _handleSearchKey,
                                  child: TextField(
                                    key: Key(
                                      '$keyPrefix-search-input',
                                    ),
                                    controller: _searchController,
                                    focusNode: _searchFocusNode,
                                    enabled: widget.enabled,
                                    decoration: InputDecoration.collapsed(
                                      hintText: widget.searchHintText,
                                    ),
                                    onTap: () => _setOpen(true),
                                    onChanged: (value) {
                                      setState(() => _query = value);
                                      _setOpen(true);
                                      _markOverlayNeedsBuild();
                                    },
                                  ),
                                ),
                              ),
                              const Icon(Icons.search),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
          if (!widget.enabled && widget.disabledText != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                widget.disabledText!,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: Theme.of(context).colorScheme.outline,
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildMenuContent() {
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
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (_filteredOptions.isEmpty)
            const ListTile(title: Text('No results'))
          else
            for (final (index, option) in _filteredOptions.indexed)
              CheckboxListTile(
                key: Key(
                  _optionKey(widget.keyPrefix, widget.label, option.value),
                ),
                value: _values.contains(option.value),
                tileColor: index == 0 ? selectedTileColor : null,
                shape: selectedTileShape,
                title: Text(option.label),
                controlAffinity: ListTileControlAffinity.leading,
                onChanged: widget.enabled ? (_) => _toggle(option.value) : null,
              ),
        ],
      ),
    );
  }
}

class CraftskyTokenInput extends StatefulWidget {
  const CraftskyTokenInput({
    required this.label,
    super.key,
    this.values = const [],
    this.helperText,
    this.errorText,
    this.enabled = true,
    this.maxSelected,
    this.inputHintText,
    this.addButtonLabel,
    this.disabledText,
    this.maxSelectedErrorText,
    this.keyPrefix,
    this.onChanged,
  });

  final String label;
  final List<String> values;
  final String? helperText;
  final String? errorText;
  final bool enabled;
  final int? maxSelected;
  final String? inputHintText;
  final String? addButtonLabel;
  final String? disabledText;
  final String? maxSelectedErrorText;
  final String? keyPrefix;
  final ValueChanged<List<String>>? onChanged;

  @override
  State<CraftskyTokenInput> createState() => _CraftskyTokenInputState();
}

class _CraftskyTokenInputState extends State<CraftskyTokenInput> {
  final _controller = TextEditingController();
  final _focusNode = FocusNode();
  String? _limitText;

  bool get _canAdd => widget.enabled && _controller.text.trim().isNotEmpty;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleTextChanged);
    _focusNode.addListener(_handleTextChanged);
  }

  void _handleTextChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _controller
      ..removeListener(_handleTextChanged)
      ..dispose();
    _focusNode
      ..removeListener(_handleTextChanged)
      ..dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant CraftskyTokenInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!_listEquals(widget.values, oldWidget.values) &&
        _isBelowSelectionLimit(widget.values.length)) {
      _limitText = null;
    }
  }

  bool _isBelowSelectionLimit(int length) {
    final max = widget.maxSelected;
    return max == null || length < max;
  }

  void _setValues(List<String> values) {
    widget.onChanged?.call(List<String>.unmodifiable(values));
  }

  void _addCurrent() {
    if (!widget.enabled) return;
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    final selected = List<String>.from(widget.values);
    if (selected.contains(text)) {
      _controller.clear();
      _focusNode.requestFocus();
      return;
    }
    final max = widget.maxSelected;
    if (max != null && selected.length >= max) {
      setState(() => _limitText = widget.maxSelectedErrorText);
      _focusNode.requestFocus();
      return;
    }
    selected.add(text);
    _controller.clear();
    setState(() => _limitText = null);
    _setValues(selected);
    _focusNode.requestFocus();
  }

  void _remove(String value) {
    if (!widget.enabled) return;
    final selected = List<String>.from(widget.values)..remove(value);
    setState(() => _limitText = null);
    _setValues(selected);
  }

  @override
  Widget build(BuildContext context) {
    final errorText = widget.errorText ?? _limitText;
    final selectedSummary = widget.values.join(', ');
    return CraftskyFieldScaffold(
      label: widget.label,
      focusNode: _focusNode,
      helperText: errorText == null ? widget.helperText : null,
      errorText: errorText,
      enabled: widget.enabled,
      semanticValue: selectedSummary.isEmpty ? 'No entries' : selectedSummary,
      child: InputDecorator(
        isFocused: _focusNode.hasFocus,
        decoration: InputDecoration(
          enabled: widget.enabled,
          contentPadding: const EdgeInsets.symmetric(
            horizontal: 16,
            vertical: 8,
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (widget.values.isNotEmpty)
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: [
                  for (final value in widget.values)
                    InputChip(
                      label: Text(value),
                      onDeleted: widget.enabled ? () => _remove(value) : null,
                      deleteIcon: Icon(
                        Icons.close,
                        key: Key(
                          '${widget.keyPrefix ?? widget.label}-remove-$value',
                        ),
                      ),
                    ),
                ],
              ),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    key: Key(
                      '${widget.keyPrefix ?? widget.label}-custom-input',
                    ),
                    controller: _controller,
                    focusNode: _focusNode,
                    enabled: widget.enabled,
                    decoration: InputDecoration.collapsed(
                      hintText: widget.inputHintText,
                    ),
                    onSubmitted: (_) => _addCurrent(),
                  ),
                ),
                TextButton(
                  key: Key('${widget.keyPrefix ?? widget.label}-add-custom'),
                  style: TextButton.styleFrom(
                    minimumSize: Size.zero,
                    padding: EdgeInsets.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  onPressed: _canAdd ? _addCurrent : null,
                  child: Text(widget.addButtonLabel ?? 'Add'),
                ),
              ],
            ),
            if (!widget.enabled && widget.disabledText != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  widget.disabledText!,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).colorScheme.outline,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class CraftskyFormSingleSelectField<T> extends StatelessWidget {
  const CraftskyFormSingleSelectField({
    required this.name,
    required this.label,
    required this.options,
    super.key,
    this.initialValue,
    this.helperText,
    this.enabled = true,
    this.searchThreshold = 5,
    this.searchHintText = 'Search',
    this.keyPrefix,
    this.validator,
    this.onChanged,
  });

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final T? initialValue;
  final String? helperText;
  final bool enabled;
  final int searchThreshold;
  final String searchHintText;
  final String? keyPrefix;
  final FormFieldValidator<T>? validator;
  final ValueChanged<T?>? onChanged;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<T>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return CraftskySingleSelectInput<T>(
          label: label,
          options: options,
          value: field.value,
          helperText: field.errorText == null ? helperText : null,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          searchThreshold: searchThreshold,
          searchHintText: searchHintText,
          keyPrefix: keyPrefix ?? name,
          onChanged: (value) {
            field.didChange(value);
            onChanged?.call(value);
          },
        );
      },
    );
  }
}

class CraftskyFormSearchableMultiSelectField<T> extends StatelessWidget {
  const CraftskyFormSearchableMultiSelectField({
    required this.name,
    required this.label,
    required this.options,
    super.key,
    this.initialValue = const [],
    this.helperText,
    this.enabled = true,
    this.validator,
    this.onChanged,
    this.maxSelected,
    this.searchHintText,
    this.disabledText,
    this.maxSelectedErrorText,
    this.keyPrefix,
  });

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final List<T> initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<List<T>>? validator;
  final ValueChanged<List<T>>? onChanged;
  final int? maxSelected;
  final String? searchHintText;
  final String? disabledText;
  final String? maxSelectedErrorText;
  final String? keyPrefix;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<T>>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return CraftskySearchableMultiSelectInput<T>(
          label: label,
          options: options,
          values: List<T>.from(field.value ?? const []),
          helperText: helperText,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          maxSelected: maxSelected,
          searchHintText: searchHintText,
          disabledText: disabledText,
          maxSelectedErrorText: maxSelectedErrorText,
          keyPrefix: keyPrefix ?? name,
          onChanged: (values) {
            field.didChange(values);
            onChanged?.call(values);
          },
        );
      },
    );
  }
}

class CraftskyFormTokenField extends StatelessWidget {
  const CraftskyFormTokenField({
    required this.name,
    required this.label,
    super.key,
    this.initialValue = const [],
    this.helperText,
    this.enabled = true,
    this.validator,
    this.onChanged,
    this.maxSelected,
    this.inputHintText,
    this.addButtonLabel,
    this.disabledText,
    this.maxSelectedErrorText,
    this.keyPrefix,
  });

  final String name;
  final String label;
  final List<String> initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<List<String>>? validator;
  final ValueChanged<List<String>>? onChanged;
  final int? maxSelected;
  final String? inputHintText;
  final String? addButtonLabel;
  final String? disabledText;
  final String? maxSelectedErrorText;
  final String? keyPrefix;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<String>>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return CraftskyTokenInput(
          label: label,
          values: List<String>.from(field.value ?? const []),
          helperText: helperText,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          maxSelected: maxSelected,
          inputHintText: inputHintText,
          addButtonLabel: addButtonLabel,
          disabledText: disabledText,
          maxSelectedErrorText: maxSelectedErrorText,
          keyPrefix: keyPrefix ?? name,
          onChanged: (values) {
            field.didChange(values);
            onChanged?.call(values);
          },
        );
      },
    );
  }
}

class _SelectedChips<T> extends StatelessWidget {
  const _SelectedChips({
    required this.name,
    required this.keyPrefix,
    required this.values,
    required this.labelByValue,
    required this.enabled,
    required this.onRemove,
  });

  final String name;
  final String keyPrefix;
  final List<T> values;
  final Map<T, String> labelByValue;
  final bool enabled;
  final ValueChanged<T> onRemove;

  @override
  Widget build(BuildContext context) {
    if (values.isEmpty) return const SizedBox(height: 32);
    return Wrap(
      spacing: 8,
      runSpacing: 4,
      children: [
        for (final value in values)
          InputChip(
            label: Text(labelByValue[value] ?? value.toString()),
            onDeleted: enabled ? () => onRemove(value) : null,
            deleteIcon: Icon(Icons.close, key: Key('$keyPrefix-remove-$value')),
          ),
      ],
    );
  }
}

class _CraftskyOptionsPanel extends StatelessWidget {
  const _CraftskyOptionsPanel({
    required this.child,
    super.key,
    this.scrollable = true,
  });

  final Widget child;
  final bool scrollable;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(radii.r3),
        border: Border.all(color: theme.colorScheme.outlineVariant),
        boxShadow: const [
          BoxShadow(
            blurRadius: 18,
            offset: Offset(0, 8),
            color: Color(0x26000000),
          ),
        ],
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(radii.r3),
        child: Material(
          type: MaterialType.transparency,
          child: scrollable
              ? SingleChildScrollView(
                  padding: const EdgeInsets.all(8),
                  child: child,
                )
              : child,
        ),
      ),
    );
  }
}

class _AnchoredSelectOverlay extends StatelessWidget {
  const _AnchoredSelectOverlay({
    required this.anchorKey,
    required this.layerLink,
    required this.onDismiss,
    required this.onEscape,
    required this.child,
  });

  final GlobalKey anchorKey;
  final LayerLink layerLink;
  final VoidCallback onDismiss;
  final VoidCallback onEscape;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final anchorBox = anchorKey.currentContext?.findRenderObject();
    if (anchorBox is! RenderBox) {
      return const SizedBox.shrink();
    }

    const gap = 4.0;
    const preferredMaxHeight = 280.0;

    return Stack(
      children: [
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: onDismiss,
          ),
        ),
        Positioned.fill(
          child: CompositedTransformFollower(
            link: layerLink,
            targetAnchor: Alignment.bottomLeft,
            offset: const Offset(0, gap),
            child: Align(
              alignment: Alignment.topLeft,
              child: SizedBox(
                width: anchorBox.size.width,
                child: Focus(
                  onKeyEvent: (_, event) {
                    if (event is KeyDownEvent &&
                        event.logicalKey == LogicalKeyboardKey.escape) {
                      onEscape();
                      return KeyEventResult.handled;
                    }
                    return KeyEventResult.ignored;
                  },
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(
                      maxHeight: preferredMaxHeight,
                    ),
                    child: child,
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

String _optionKey(String? keyPrefix, String label, Object? value) {
  return '${keyPrefix ?? label}-option-$value';
}

bool _listEquals<T>(List<T> a, List<T> b) {
  if (identical(a, b)) return true;
  if (a.length != b.length) return false;
  for (var index = 0; index < a.length; index++) {
    if (a[index] != b[index]) return false;
  }
  return true;
}
