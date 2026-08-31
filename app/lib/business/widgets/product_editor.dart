import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

typedef ProductImagePicker =
    Future<ProfileImagePickResult?> Function(
      void Function(Uint8List bytes) onPreviewReady,
    );

class ProductEditor extends ConsumerStatefulWidget {
  const ProductEditor({
    required this.onSave,
    this.initial,
    this.pickImage,
    this.onCancel,
    super.key,
  });

  final ProductDraft? initial;
  final ValueChanged<ProductDraft> onSave;
  final ProductImagePicker? pickImage;
  final VoidCallback? onCancel;

  @override
  ConsumerState<ProductEditor> createState() => _ProductEditorState();
}

class _ProductEditorState extends ConsumerState<ProductEditor> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _title;
  late final TextEditingController _destination;
  late final TextEditingController _amount;
  late final TextEditingController _currency;
  late final TextEditingController _alt;
  late BusinessImageDraft _image;
  Uint8List? _pendingPreview;
  bool _uploading = false;
  bool _showImageError = false;
  bool _showPriceError = false;
  String? _uploadError;
  late final ActiveAccountLease? _owner;

  @override
  void initState() {
    super.initState();
    _owner = ref.read(sessionRegistryProvider).value?.activeLease;
    final initial = widget.initial;
    _title = TextEditingController(text: initial?.title);
    _destination = TextEditingController(text: initial?.destination);
    _amount = TextEditingController(text: initial?.amount);
    _currency = TextEditingController(text: initial?.currency);
    _image = initial?.image ?? const MissingBusinessImageDraft();
    _alt = TextEditingController(text: _image.alt);
  }

  @override
  void dispose() {
    _title.dispose();
    _destination.dispose();
    _amount.dispose();
    _currency.dispose();
    _alt.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Form(
      key: _formKey,
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              widget.initial == null
                  ? l10n.businessProductEditorAddTitle
                  : l10n.businessProductEditorEditTitle,
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 20),
            TextFormField(
              key: const ValueKey('product-title'),
              controller: _title,
              decoration: InputDecoration(
                labelText: l10n.businessProductTitleLabel,
              ),
              maxLength: businessProductTitleLimit,
              validator: (value) {
                if (value == null || value.trim().isEmpty) {
                  return l10n.businessProductTitleRequired;
                }
                return null;
              },
            ),
            TextFormField(
              key: const ValueKey('product-destination'),
              controller: _destination,
              decoration: InputDecoration(
                labelText: l10n.businessProductDestinationLabel,
                hintText: l10n.businessProductDestinationHint,
              ),
              keyboardType: TextInputType.url,
              validator: (value) {
                final uri = Uri.tryParse(value ?? '');
                if (uri == null ||
                    uri.scheme != 'https' ||
                    uri.host.isEmpty ||
                    uri.userInfo.isNotEmpty) {
                  return l10n.businessProductDestinationInvalid;
                }
                return null;
              },
            ),
            const SizedBox(height: 12),
            if (_pendingPreview != null)
              SizedBox(
                height: 140,
                child: Image.memory(_pendingPreview!, fit: BoxFit.cover),
              )
            else if (_image.hasImage)
              const SizedBox(
                height: 96,
                child: Icon(Icons.image_outlined, size: 56),
              ),
            if (_uploading)
              LinearProgressIndicator(
                semanticsLabel: l10n.businessImageUploading,
              ),
            Wrap(
              spacing: 8,
              children: [
                TextButton.icon(
                  onPressed: _uploading ? null : _pickImage,
                  icon: const Icon(Icons.add_photo_alternate_outlined),
                  label: Text(
                    _image.hasImage
                        ? l10n.businessProductReplaceImage
                        : l10n.businessProductAddImage,
                  ),
                ),
                if (_image.hasImage)
                  TextButton.icon(
                    onPressed: _uploading
                        ? null
                        : () => setState(() {
                            _image = const RemovedBusinessImageDraft();
                            _pendingPreview = null;
                          }),
                    icon: const Icon(Icons.delete_outline),
                    label: Text(l10n.businessProductRemoveImage),
                  ),
              ],
            ),
            if (_showImageError && !_image.hasImage)
              Text(
                l10n.businessProductImageRequired,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            if (_uploadError != null)
              Text(
                _uploadError!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            TextField(
              key: const ValueKey('product-alt'),
              controller: _alt,
              decoration: InputDecoration(
                labelText: l10n.businessProductAltLabel,
              ),
              maxLength: businessImageAltLimit,
            ),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    key: const ValueKey('product-amount'),
                    controller: _amount,
                    decoration: InputDecoration(
                      labelText: l10n.businessProductAmountLabel,
                    ),
                    keyboardType: const TextInputType.numberWithOptions(
                      decimal: true,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextField(
                    key: const ValueKey('product-currency'),
                    controller: _currency,
                    decoration: InputDecoration(
                      labelText: l10n.businessProductCurrencyLabel,
                    ),
                    textCapitalization: TextCapitalization.characters,
                    maxLength: 3,
                  ),
                ),
              ],
            ),
            if (_showPriceError)
              Text(
                l10n.businessProductPriceInvalid,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            const SizedBox(height: 16),
            Wrap(
              alignment: WrapAlignment.end,
              spacing: 8,
              children: [
                TextButton(
                  onPressed: widget.onCancel,
                  child: Text(l10n.businessProductCancel),
                ),
                FilledButton(
                  onPressed: _uploading ? null : _save,
                  child: Text(l10n.businessProductSave),
                ),
              ],
            ),
          ],
        ),
      ),
    );
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

  bool _isCurrent(ActiveAccountLease? ownership) {
    if (!mounted) return false;
    if (ownership == null) return true;
    return ref.read(sessionRegistryProvider).value?.isCurrent(ownership) ??
        false;
  }

  void _save() {
    if (!_isCurrent(_owner)) return;
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
      amount: _amount.text,
      currency: _currency.text,
    );
    final errors = product.validate();
    final formValid = _formKey.currentState?.validate() ?? false;
    setState(() {
      _showImageError = errors.contains(ProductDraftError.imageRequired);
      _showPriceError =
          errors.contains(ProductDraftError.priceIncomplete) ||
          errors.contains(ProductDraftError.priceInvalid);
    });
    if (formValid && errors.isEmpty) widget.onSave(product);
  }
}
