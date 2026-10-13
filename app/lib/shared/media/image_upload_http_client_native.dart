import 'package:dio/dio.dart';

const imageUploadSupportsSendProgress = true;

Dio newImageUploadHttpClient() => Dio(
  BaseOptions(
    connectTimeout: const Duration(seconds: 15),
    sendTimeout: const Duration(minutes: 2),
    followRedirects: false,
    maxRedirects: 0,
  ),
);
