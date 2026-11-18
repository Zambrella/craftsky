import AVFoundation
import Flutter
import UIKit
import StoreKit

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let ownershipChannel = FlutterMethodChannel(
      name: "social.craftsky/store_ownership",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    ownershipChannel.setMethodCallHandler { call, result in
      if call.method == "canChangePlan" {
        guard #available(iOS 15.0, *),
              let arguments = call.arguments as? [String: Any],
              let oldID = arguments["oldProductId"] as? String,
              let newID = arguments["newProductId"] as? String,
              oldID != newID else {
          result(false)
          return
        }
        Task {
          do {
            let products = try await Product.products(for: [oldID, newID])
            let oldGroup = products.first { $0.id == oldID }?.subscription?.subscriptionGroupID
            let newGroup = products.first { $0.id == newID }?.subscription?.subscriptionGroupID
            let allowed = oldGroup != nil && oldGroup == newGroup
            await MainActor.run { result(allowed) }
          } catch {
            await MainActor.run { result(false) }
          }
        }
        return
      }
      guard call.method == "check" else {
        result(FlutterMethodNotImplemented)
        return
      }
      guard let arguments = call.arguments as? [String: Any],
            let appUserId = arguments["appUserId"] as? String,
            let expected = UUID(uuidString: appUserId) else {
        result(["status": "unverified"])
        return
      }
      guard #available(iOS 15.0, *) else {
        result(["status": "unverified"])
        return
      }
      Task {
        // Include expired subscriptions: RevenueCat's original-customer receipt
        // association can survive expiry. Never sync or transfer a receipt here.
        var status = "allowed"
        var conflictingOwner: UUID?
        var hasActiveSubscription = false
        for await verification in StoreKit.Transaction.all {
          switch verification {
          case .verified(let transaction):
            guard transaction.productType == .autoRenewable else { continue }
            guard let owner = transaction.appAccountToken else {
              status = "unverified"
              continue
            }
            if owner != expected {
              status = "conflict"
              conflictingOwner = owner
              break
            }
          case .unverified:
            status = "unverified"
          }
          if status == "conflict" { break }
        }
        // Current entitlements are store evidence independent of RevenueCat
        // indexing. Unfinished purchases also prevent a second independent
        // subscription while the first transaction is still being delivered.
        if status != "conflict" {
          for await verification in StoreKit.Transaction.currentEntitlements {
            switch verification {
            case .verified(let transaction):
              guard transaction.productType == .autoRenewable else { continue }
              hasActiveSubscription = true
              guard let owner = transaction.appAccountToken else {
                status = "unverified"
                continue
              }
              if owner != expected {
                status = "conflict"
                conflictingOwner = owner
              }
            case .unverified:
              status = "unverified"
            }
            if status == "conflict" { break }
          }
        }
        if status != "conflict" {
          for await verification in StoreKit.Transaction.unfinished {
            switch verification {
            case .verified(let transaction):
              guard transaction.productType == .autoRenewable else { continue }
              hasActiveSubscription = true
              guard let owner = transaction.appAccountToken else {
                status = "unverified"
                continue
              }
              if owner != expected {
                status = "conflict"
                conflictingOwner = owner
              }
            case .unverified:
              status = "unverified"
            }
            if status == "conflict" { break }
          }
        }
        await MainActor.run {
          var response: [String: Any] = [
            "status": status,
            "hasActiveSubscription": hasActiveSubscription
          ]
          if let owner = conflictingOwner { response["ownerAppUserId"] = owner.uuidString.lowercased() }
          result(response)
        }
      }
    }
    let channel = FlutterMethodChannel(
      name: "social.craftsky/local_video",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    channel.setMethodCallHandler { call, result in
      guard call.method == "probe" || call.method == "midpointFrame" else {
        result(FlutterMethodNotImplemented)
        return
      }
      guard let arguments = call.arguments as? [String: Any],
            let path = arguments["path"] as? String, !path.isEmpty else {
        result(FlutterError(code: "invalid_video_path", message: "Video path required", details: nil))
        return
      }
      Task {
        do {
          if call.method == "midpointFrame" {
            guard let timeUs = arguments["timeUs"] as? NSNumber,
                  timeUs.int64Value >= 0 else {
              await MainActor.run { result(nil) }
              return
            }
            let bytes = try await LocalVideoProbe.midpointFrame(path: path, timeUs: timeUs.int64Value)
            await MainActor.run { result(bytes) }
          } else {
            let metadata = try await LocalVideoProbe.probe(path: path)
            await MainActor.run { result(metadata) }
          }
        } catch {
          let cause = error as NSError
          await MainActor.run {
            result(FlutterError(
              code: "video_probe_failed",
              message: "Unable to prepare local video",
              details: ["domain": cause.domain, "code": cause.code]
            ))
          }
        }
      }
    }
  }
}

// Metadata and a bounded, correctly oriented poster; no player or GPU texture.
private enum LocalVideoProbe {
  private static func asset(path: String) -> AVURLAsset {
    let url = URL(string: path).flatMap { $0.isFileURL ? $0 : nil }
      ?? URL(fileURLWithPath: path)
    return AVURLAsset(url: url)
  }

  static func probe(path: String) async throws -> [String: Any] {
    let asset = asset(path: path)
    guard let track = try await asset.loadTracks(withMediaType: .video).first else {
      throw NSError(domain: "CraftSky.LocalVideo", code: 1)
    }
    let (size, transform) = try await track.load(.naturalSize, .preferredTransform)
    let bounds = CGRect(origin: .zero, size: size).applying(transform)
    let width = abs(bounds.width).rounded()
    let height = abs(bounds.height).rounded()
    guard width.isFinite, height.isFinite, width > 0, height > 0 else {
      throw NSError(domain: "CraftSky.LocalVideo", code: 2)
    }
    let duration = try await asset.load(.duration)
    let seconds = CMTimeGetSeconds(duration)
    let poster = try await frame(asset: asset, at: .zero, maxSize: 1280)
    var metadata: [String: Any] = [
      "width": Int(width), "height": Int(height),
      "posterBytes": poster
    ]
    if seconds.isFinite, seconds > 0, seconds * 1000 < Double(Int64.max) {
      metadata["durationMs"] = Int64((seconds * 1000).rounded())
    }
    return metadata
  }

  static func midpointFrame(path: String, timeUs: Int64) async throws -> FlutterStandardTypedData {
    try await frame(
      asset: asset(path: path), at: CMTime(value: timeUs, timescale: 1_000_000),
      maxSize: 640, precise: true
    )
  }

  private static func frame(
    asset: AVURLAsset, at time: CMTime, maxSize: CGFloat, precise: Bool = false
  ) async throws -> FlutterStandardTypedData {
    let generator = AVAssetImageGenerator(asset: asset)
    generator.appliesPreferredTrackTransform = true
    generator.maximumSize = CGSize(width: maxSize, height: maxSize)
    if precise {
      let tolerance = CMTime(value: 1, timescale: 30)
      generator.requestedTimeToleranceBefore = tolerance
      generator.requestedTimeToleranceAfter = tolerance
    }
    let image: CGImage = try await withCheckedThrowingContinuation { continuation in
      generator.generateCGImagesAsynchronously(forTimes: [NSValue(time: time)]) {
        _, image, _, status, error in
        if status == .succeeded, let image = image {
          continuation.resume(returning: image)
        } else {
          continuation.resume(throwing: error ?? NSError(domain: "CraftSky.LocalVideo", code: 3))
        }
      }
    }
    guard let poster = UIImage(cgImage: image).jpegData(compressionQuality: 0.85) else {
      throw NSError(domain: "CraftSky.LocalVideo", code: 4)
    }
    return FlutterStandardTypedData(bytes: poster)
  }
}
