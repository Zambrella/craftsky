part of '../craftsky_select_inputs.dart';

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
