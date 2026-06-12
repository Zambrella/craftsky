part of '../craftsky_select_inputs.dart';

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
                                    key: Key('$keyPrefix-search-input'),
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
