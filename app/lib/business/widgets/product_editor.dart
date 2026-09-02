import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:l10n_currencies/l10n_currencies.dart';
import 'package:sealed_countries/sealed_countries.dart';
import 'package:uuid/uuid.dart';

typedef ProductImagePicker =
    Future<ProfileImagePickResult?> Function(
      void Function(Uint8List bytes) onPreviewReady,
    );

Future<ProductDraft?> showProductEditorSheet(
  BuildContext context, {
  ProductDraft? initial,
  ProductImagePicker? pickImage,
  Future<bool> Function(ProductDraft product)? persist,
  bool Function(String destination)? destinationExists,
}) {
  return responsiveModalNavigator(context).push<ProductDraft>(
    MaterialPageRoute<ProductDraft>(
      fullscreenDialog: true,
      builder: (routeContext) => ProductEditor(
        initial: initial,
        pickImage: pickImage,
        persist: persist,
        destinationExists: destinationExists,
        onCancel: () => Navigator.of(routeContext).pop(),
        onSave: (product) => Navigator.of(routeContext).pop(product),
      ),
    ),
  );
}

class ProductEditor extends ConsumerStatefulWidget {
  const ProductEditor({
    required this.onSave,
    this.initial,
    this.pickImage,
    this.onCancel,
    this.locale,
    this.persist,
    this.destinationExists,
    super.key,
  });

  final ProductDraft? initial;
  final ValueChanged<ProductDraft> onSave;
  final ProductImagePicker? pickImage;
  final VoidCallback? onCancel;
  final Locale? locale;
  final Future<bool> Function(ProductDraft product)? persist;
  final bool Function(String destination)? destinationExists;

  @override
  ConsumerState<ProductEditor> createState() => _ProductEditorState();
}

class _ProductEditorState extends ConsumerState<ProductEditor> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _title;
  late final TextEditingController _destination;
  late final TextEditingController _amount;
  late final TextEditingController _alt;
  late String _currency;
  late List<CraftskySelectOption<String>> _currencyOptions;
  late BusinessImageDraft _image;
  Uint8List? _pendingPreview;
  bool _uploading = false;
  bool _showImageError = false;
  bool _showPriceError = false;
  String? _uploadError;
  late final ActiveAccountLease? _owner;
  bool _dirty = false;
  bool _currencyInitialized = false;
  bool _saving = false;
  String? _saveError;

  @override
  void initState() {
    super.initState();
    _owner = ref.read(sessionRegistryProvider).value?.activeLease;
    final initial = widget.initial;
    _title = TextEditingController(text: initial?.title);
    _destination = TextEditingController(text: initial?.destination);
    _amount = TextEditingController(
      text: initial == null
          ? null
          : _editableAmount(initial.amount, initial.currency),
    );
    _currency = initial?.currency ?? '';
    _image = initial?.image ?? const MissingBusinessImageDraft();
    _alt = TextEditingController(text: _image.alt);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_currencyInitialized) return;
    _currencyInitialized = true;
    final locale = widget.locale ?? View.of(context).platformDispatcher.locale;
    final localeName = locale.toString();
    final localized = CurrenciesLocaleMapper().localize(
      businessProductCurrencies,
      mainLocale: localeName,
      fallbackLocale: 'en',
    );
    final names = <String, String>{};
    for (final entry in localized.entries) {
      names.putIfAbsent(entry.key.isoCode, () => entry.value);
    }
    _currencyOptions =
        businessProductCurrencies
            .map(
              (code) => CraftskySelectOption(
                value: code,
                label: '$code - ${names[code] ?? code}',
              ),
            )
            .toList()
          ..sort((a, b) => a.label.compareTo(b.label));
    if (widget.initial == null) {
      final countryCode = locale.countryCode;
      _currency = countryCode == null
          ? ''
          : WorldCountry.maybeFromCodeShort(
                  countryCode,
                )?.currencies?.firstOrNull?.code ??
                '';
    }
  }

  @override
  void dispose() {
    _title.dispose();
    _destination.dispose();
    _amount.dispose();
    _alt.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    return PopScope<Object?>(
      canPop: !_saving && !_dirty,
      onPopInvokedWithResult: (didPop, _) async {
        if (_saving) return;
        if (!didPop) await _cancel();
      },
      child: Scaffold(
        backgroundColor: swatches.paper,
        appBar: AppBar(
          leading: CloseButton(onPressed: _saving ? null : _cancel),
          title: Text(
            widget.initial == null
                ? l10n.businessProductEditorAddTitle
                : l10n.businessProductEditorEditTitle,
          ),
        ),
        body: Form(
          key: _formKey,
          onChanged: _markDirty,
          child: Stack(
            children: [
              SingleChildScrollView(
                padding: EdgeInsets.fromLTRB(
                  spacing.sp4,
                  spacing.sp4,
                  spacing.sp4,
                  0,
                ),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 720),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        CraftskyTextFormField(
                          textFieldKey: const ValueKey('product-title'),
                          controller: _title,
                          label: l10n.businessProductTitleLabel,
                          required: true,
                          maxLength: businessProductTitleLimit,
                          validator: (value) {
                            if (value == null || value.trim().isEmpty) {
                              return l10n.businessProductTitleRequired;
                            }
                            return null;
                          },
                        ),
                        SizedBox(height: spacing.sp5),
                        CraftskyTextFormField(
                          textFieldKey: const ValueKey('product-destination'),
                          controller: _destination,
                          label: l10n.businessProductDestinationLabel,
                          hintText: l10n.businessProductDestinationHint,
                          required: true,
                          keyboardType: TextInputType.url,
                          validator: (value) {
                            final uri = Uri.tryParse(value ?? '');
                            if (uri == null ||
                                uri.scheme != 'https' ||
                                uri.host.isEmpty ||
                                uri.userInfo.isNotEmpty) {
                              return l10n.businessProductDestinationInvalid;
                            }
                            if (widget.destinationExists?.call(value!) ??
                                false) {
                              return l10n.businessProductDestinationDuplicate;
                            }
                            return null;
                          },
                        ),
                        SizedBox(height: spacing.sp5),
                        ComposerImageAttachmentSection(
                          imagesState: ComposerImagesState(
                            images: [?_attachmentDraft],
                          ),
                          enabled: !_uploading,
                          required: true,
                          maxImages: 1,
                          keyPrefix: 'product',
                          imageUrlFor: (_) => _image.previewUrl,
                          onAddImages: _pickImage,
                          onAltTextChanged: (_, value) {
                            _alt.text = value;
                            _markDirty();
                          },
                          onRemove: (_) => _removeImage(),
                          onReplace: (_) => _pickImage(),
                          onReplaceUnavailable: (_) => _pickImage(),
                          onReorder: (_, _) {},
                          validationErrorText:
                              _uploadError ??
                              (_showImageError && !_image.hasImage
                                  ? l10n.businessProductImageRequired
                                  : null),
                        ),
                        SizedBox(height: spacing.sp5),
                        LayoutBuilder(
                          builder: (context, constraints) => _priceFields(
                            l10n,
                            spacing,
                            sideBySide: constraints.maxWidth >= 520,
                          ),
                        ),
                        if (_showPriceError) ...[
                          SizedBox(height: spacing.sp2),
                          Text(
                            l10n.businessProductPriceInvalid,
                            style: TextStyle(color: theme.colorScheme.error),
                          ),
                        ],
                        if (_saveError != null) ...[
                          SizedBox(height: spacing.sp2),
                          Text(
                            _saveError!,
                            style: TextStyle(color: theme.colorScheme.error),
                          ),
                        ],
                        SizedBox(
                          key: const Key('product-editor-bottom-safe-space'),
                          height:
                              spacing.sp9 +
                              MediaQuery.paddingOf(context).bottom,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              PositionedDirectional(
                start: spacing.sp4,
                end: spacing.sp4,
                bottom: 0,
                child: SafeArea(
                  top: false,
                  minimum: EdgeInsets.only(bottom: spacing.sp4),
                  child: ChunkyButton(
                    key: const ValueKey('product-submit'),
                    onPressed: _uploading || _saving ? null : _save,
                    style: ButtonStyle(
                      minimumSize: WidgetStatePropertyAll(
                        Size.fromHeight(spacing.sp7),
                      ),
                    ),
                    child: _saving
                        ? SizedBox.square(
                            dimension: spacing.sp5,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              semanticsLabel: l10n.businessSaving,
                            ),
                          )
                        : Text(l10n.businessProductSave),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _priceFields(
    AppLocalizations l10n,
    SpacingTheme spacing, {
    required bool sideBySide,
  }) {
    final amount = CraftskyTextInput(
      label: l10n.businessProductAmountLabel,
      textFieldKey: const ValueKey('product-amount'),
      controller: _amount,
      keyboardType: const TextInputType.numberWithOptions(decimal: true),
      inputFormatters: [_productAmountFormatter],
    );
    final currency = CraftskySingleSelectInput<String>(
      label: l10n.businessProductCurrencyLabel,
      value: _currency.isEmpty ? null : _currency,
      options: _currencyOptions,
      keyPrefix: 'product-currency',
      searchThreshold: 0,
      onChanged: (value) {
        setState(() => _currency = value ?? '');
        _markDirty();
      },
    );
    if (!sideBySide) {
      return Column(
        children: [
          amount,
          SizedBox(height: spacing.sp5),
          currency,
        ],
      );
    }
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(child: amount),
        SizedBox(width: spacing.sp4),
        Expanded(child: currency),
      ],
    );
  }

  Future<void> _cancel() async {
    if (_saving) return;
    if (_dirty) {
      final l10n = AppLocalizations.of(context);
      final discard = await showCraftskyConfirmDialog(
        context,
        title: l10n.editProfileDiscardTitle,
        message: l10n.editProfileDiscardMessage,
        confirmLabel: l10n.editProfileDiscardConfirm,
        cancelLabel: l10n.editProfileDiscardCancel,
      );
      if (!discard || !mounted) return;
      _dirty = false;
    }
    final onCancel = widget.onCancel;
    if (onCancel != null) {
      onCancel();
    } else if (Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
    }
  }

  Future<void> _pickImage() async {
    final l10n = AppLocalizations.of(context);
    final ownership = _owner;
    final picker =
        widget.pickImage ??
        (onPreviewReady) => ref
            .read(profileImagePickerProvider)
            .pickAndUpload(onPreviewReady: onPreviewReady);
    setState(() {
      _uploading = true;
      _uploadError = null;
      _pendingPreview = null;
    });
    try {
      final result = await picker(
        (bytes) {
          if (_isCurrent(ownership)) {
            setState(() => _pendingPreview = bytes);
          }
        },
      );
      if (!_isCurrent(ownership)) return;
      if (result != null) {
        setState(() {
          _image = UploadedBusinessImageDraft.fromUpload(
            result.uploaded,
            alt: _alt.text,
            previewBytes: result.previewBytes,
          );
          _pendingPreview = result.previewBytes;
          _showImageError = false;
          _dirty = true;
        });
      } else {
        setState(() => _pendingPreview = null);
      }
    } on Object {
      if (!_isCurrent(ownership)) return;
      setState(() {
        _pendingPreview = null;
        _uploadError = l10n.businessProductsUploadError;
      });
    } finally {
      if (mounted) {
        setState(() {
          _uploading = false;
          if (!_isCurrent(ownership)) _pendingPreview = null;
        });
      }
    }
  }

  void _removeImage() {
    setState(() {
      _image = const RemovedBusinessImageDraft();
      _pendingPreview = null;
      _dirty = true;
    });
  }

  bool _isCurrent(ActiveAccountLease? ownership) {
    if (!mounted) return false;
    if (ownership == null) return true;
    return ref.read(sessionRegistryProvider).value?.isCurrent(ownership) ??
        false;
  }

  Future<void> _save() async {
    if (!_isCurrent(_owner)) return;
    final l10n = AppLocalizations.of(context);
    final amount = _canonicalAmount(_amount.text);
    final image = switch (_image) {
      final ExistingBusinessImageDraft value => value.withAlt(_alt.text),
      final UploadedBusinessImageDraft value => value.withAlt(_alt.text),
      _ => _image,
    };
    final product = ProductDraft(
      id: widget.initial?.id ?? const Uuid().v4(),
      title: _title.text,
      destination: _destination.text,
      image: image,
      amount: amount,
      currency: amount.isEmpty ? '' : _currency,
    );
    final errors = product.validate();
    final formValid = _formKey.currentState?.validate() ?? false;
    setState(() {
      _showImageError = errors.contains(ProductDraftError.imageRequired);
      _showPriceError =
          errors.contains(ProductDraftError.priceIncomplete) ||
          errors.contains(ProductDraftError.priceInvalid);
      _saveError = null;
    });
    if (!formValid || errors.isNotEmpty) return;
    final persist = widget.persist;
    if (persist != null) {
      setState(() => _saving = true);
      final success = await persist(product);
      if (!mounted) return;
      setState(() {
        _saving = false;
        _saveError = success ? null : l10n.businessProductsSaveError;
      });
      if (!success) return;
    }
    _dirty = false;
    widget.onSave(product);
  }

  ComposerImageDraft? get _attachmentDraft {
    final bytes = _pendingPreview ?? _image.previewBytes;
    if (!_uploading && !_image.hasImage && bytes == null) return null;
    final businessRatio = switch (_image) {
      ExistingBusinessImageDraft(:final aspectRatio) ||
      UploadedBusinessImageDraft(:final aspectRatio) => aspectRatio,
      _ => null,
    };
    final ratio = businessRatio == null
        ? null
        : CreatePostImageAspectRatio(
            width: businessRatio.width,
            height: businessRatio.height,
          );
    final phase = _uploading
        ? const ImagePreparing()
        : ImageUploaded(
            UploadedDraftImage(
              cid: switch (_image) {
                ExistingBusinessImageDraft(:final cid) => cid,
                UploadedBusinessImageDraft(:final cid) => cid,
                _ => '',
              },
              mime: switch (_image) {
                ExistingBusinessImageDraft(:final mime) => mime,
                UploadedBusinessImageDraft(:final mime) => mime,
                _ => 'image/jpeg',
              },
              size: switch (_image) {
                ExistingBusinessImageDraft(:final size) => size,
                UploadedBusinessImageDraft(:final size) => size,
                _ => bytes?.length ?? 0,
              },
              aspectRatio: ratio,
            ),
          );
    return ComposerImageDraft(
      id: 'image',
      fileName: 'product-image',
      mimeType: switch (_image) {
        ExistingBusinessImageDraft(:final mime) => mime,
        UploadedBusinessImageDraft(:final mime) => mime,
        _ => 'image/jpeg',
      },
      altText: _alt.text,
      phase: phase,
      previewBytes: bytes,
      previewAspectRatio: ratio,
    );
  }

  void _markDirty() {
    if (!_dirty) setState(() => _dirty = true);
  }
}

final _productAmountFormatter = TextInputFormatter.withFunction((old, next) {
  return RegExp(
        r'^(?:|0|[1-9][0-9]{0,11})(?:\.[0-9]{0,4})?$',
      ).hasMatch(next.text)
      ? next
      : old;
});

String _canonicalAmount(String value) {
  if (!value.contains('.')) return value;
  final canonical = value.replaceFirst(RegExp(r'0+$'), '');
  return canonical.endsWith('.')
      ? canonical.substring(0, canonical.length - 1)
      : canonical;
}

String _editableAmount(String value, String currency) {
  if (value.isEmpty || !businessProductCurrencies.contains(currency)) {
    return value;
  }
  final scale = businessCurrencyScale(currency);
  if (scale == 0) return value;
  final parts = value.split('.');
  if (parts.length > 2) return value;
  final fraction = parts.length == 1 ? '' : parts[1];
  if (fraction.length > scale) return value;
  return '${parts[0]}.${fraction.padRight(scale, '0')}';
}
