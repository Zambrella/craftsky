import Foundation
import Darwin
import PatrolImpl
import XCTest

// Patrol's standard iOS enterText action includes the entered value in NSLog.
// Keep this narrowly scoped route in the UI-test runner, never in Runner.app.
@objc(CraftSkySecureOAuthEntry)
final class SecureOAuthEntry: NSObject, PatrolServerExtension {
    var name: String { "CraftSky secure OAuth entry" }

    func register(on registrar: PatrolRouteRegistrar) {
        let idleTimeoutConfigured = configureIdleTimeout()
        registrar.post("craftsky/oauthState") { _ in
            return try DispatchQueue.main.sync {
                let safari = XCUIApplication(bundleIdentifier: "com.apple.mobilesafari")
                guard safari.state == .runningForeground else {
                    return Data("{\"buttons\":[],\"passwordVisible\":false,\"wrongCredentials\":false}".utf8)
                }
                // Avoid Patrol's full-tree decoder: newer iOS element types
                // are not in its enum. Only return these public control names.
                let buttons = ["Not Now", "Authorize", "Continue", "Open", "Open CraftSky"].filter { label in
                    let button = safari.buttons.matching(identifier: label).firstMatch
                    return button.exists && button.isHittable
                }
                return try JSONSerialization.data(withJSONObject: [
                    "buttons": buttons,
                    "passwordVisible": safari.secureTextFields.firstMatch.exists,
                    "wrongCredentials": safari.staticTexts["Wrong identifier or password"].exists,
                ])
            }
        }
        registrar.post("craftsky/enterOAuthPassword") { data in
            guard idleTimeoutConfigured else {
                return Data("{\"error\":\"nativeIdleConfigurationUnavailable\"}".utf8)
            }
            struct Request: Decodable { let password: String }
            let request = try JSONDecoder().decode(Request.self, from: data)
            // Route handlers run off the main thread. XCTest interactions must
            // use the main queue, just like Patrol's built-in iOS actions.
            return DispatchQueue.main.sync {
                let safari = XCUIApplication(bundleIdentifier: "com.apple.mobilesafari")
                let urlField = safari.textFields.matching(
                    NSPredicate(format: "identifier == %@ OR identifier == %@", "URL", "TabBarItemTitle")
                ).firstMatch
                // Safari hides its address bar while the page's password field
                // has focus. Tap outside the fields before reading the origin.
                if !urlField.exists {
                    let heading = safari.staticTexts["Confirm your password to continue"]
                    if heading.exists {
                        heading.tap()
                    } else {
                        let done = safari.toolbars.buttons["Done"]
                        if done.exists { done.tap() }
                    }
                    if !urlField.exists { safari.swipeDown() }
                }
                guard urlField.waitForExistence(timeout: 5) else {
                    return Data("{\"error\":\"safariURLUnavailable\"}".utf8)
                }
                urlField.tap()
                // First-use sheets appear asynchronously after address editing.
                for _ in 0..<2 {
                    let introduction = safari.descendants(matching: .any).matching(NSPredicate(
                        format: "label BEGINSWITH %@ OR label BEGINSWITH %@",
                        "Speed up your typing by sliding", "Safari search now shows"
                    )).firstMatch
                    if !introduction.waitForExistence(timeout: 2) { break }
                    let buttons = safari.buttons.matching(identifier: "Continue").allElementsBoundByIndex
                    guard let button = buttons.reversed().first(where: { $0.isHittable }) else {
                        return Data("{\"error\":\"safariIntroductionBlocked\"}".utf8)
                    }
                    button.tap()
                }
                let address = urlField.value as? String ?? ""
                let url = URL(string: address.contains("://") ? address : "https://" + address)
                let cancel = safari.buttons.matching(NSPredicate(
                    format: "identifier == %@ OR label == %@", "CancelBarItemButton", "Cancel"
                )).firstMatch
                if cancel.waitForExistence(timeout: 2) { cancel.tap() }
                guard safari.keyboards.firstMatch.waitForNonExistence(timeout: 5) else {
                    return Data("{\"error\":\"addressKeyboardStillVisible\"}".utf8)
                }
                let field = safari.secureTextFields.firstMatch
                guard url?.host == "bsky.social" else {
                    return Data("{\"error\":\"unexpectedOrigin\"}".utf8)
                }
                guard field.waitForExistence(timeout: 5), !request.password.isEmpty else {
                    return Data("{\"error\":\"passwordFieldUnavailable\"}".utf8)
                }
                field.tap()
                guard safari.keyboards.firstMatch.waitForExistence(timeout: 5) else {
                    return Data("{\"error\":\"keyboardUnavailable\"}".utf8)
                }
                // XCTest redacts typeText values for secure text fields.
                field.typeText(request.password)
                let heading = safari.staticTexts["Confirm your password to continue"]
                if heading.exists {
                    heading.tap()
                } else {
                    let done = safari.toolbars.buttons["Done"]
                    if done.exists { done.tap() }
                }
                guard safari.keyboards.firstMatch.waitForNonExistence(timeout: 5) else {
                    return Data("{\"error\":\"passwordKeyboardStillVisible\"}".utf8)
                }
                return Data("{}".utf8)
            }
        }
    }

    private func configureIdleTimeout() -> Bool {
        // Test-runner-only XCTest SPI also used by WebDriverAgent. Safari can
        // remain animated indefinitely; keep native idle checks below Patrol's
        // HTTP timeout and rely on the explicit element/keyboard waits above.
        // https://github.com/appium/WebDriverAgent/blob/master/WebDriverAgentLib/Utilities/FBXCAXClientProxy.m
        guard let library = dlopen(nil, RTLD_LAZY) else { return false }
        defer { dlclose(library) }
        guard let symbol = dlsym(library, "_XCTSetApplicationStateTimeout") else { return false }
        let setTimeout = unsafeBitCast(symbol, to: (@convention(c) (Double) -> Void).self)
        setTimeout(2)
        return true
    }
}
