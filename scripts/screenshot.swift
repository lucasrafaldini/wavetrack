// Renders WaveTrack pages off-screen and saves Retina PNGs for the documentation.
// macOS only; needs no screen-recording permission because it never touches the screen.
//
//   swift scripts/screenshot.swift <base-url> <admin-token> <out-dir> <name>=<route>[@width x height][:theme] ...
//   e.g. now=/now history=/history@1440x900:dark audit=/audit:classic join=/join/abc@390x844
//   Themes: light (default), dark, classic, code, contrast.
//   Append |Button label to click a button before the shot: invite=/people|Invite with QR code
//   The route /signin signs out first, to capture the sign-in page.

import AppKit
import WebKit

struct Shot {
    let name: String
    let route: String
    let click: String?
    let size: CGSize
    let theme: String
    var dark: Bool { ["dark", "code", "contrast"].contains(theme) }
}

let themes = ["light", "dark", "classic", "code", "contrast"]

func parse(_ arg: String) -> Shot {
    var rest = arg
    var click: String?
    if let bar = rest.firstIndex(of: "|") {
        click = String(rest[rest.index(after: bar)...])
        rest = String(rest[..<bar])
    }
    var theme = "light"
    if let colon = rest.lastIndex(of: ":"), themes.contains(String(rest[rest.index(after: colon)...])) {
        theme = String(rest[rest.index(after: colon)...])
        rest = String(rest[..<colon])
    }
    var size = CGSize(width: 1440, height: 900)
    if let at = rest.lastIndex(of: "@") {
        let dims = rest[rest.index(after: at)...].split(separator: "x").compactMap { Double($0) }
        if dims.count == 2 { size = CGSize(width: dims[0], height: dims[1]) }
        rest = String(rest[..<at])
    }
    let parts = rest.split(separator: "=", maxSplits: 1).map(String.init)
    // Click labels may themselves contain ":" or "@", so dimensions and theme come first.
    return Shot(name: parts[0], route: parts.count > 1 ? parts[1] : "/now", click: click, size: size, theme: theme)
}

@MainActor
final class Shooter: NSObject, WKNavigationDelegate {
    let base: String
    let token: String
    let outDir: URL
    var shots: [Shot]
    let webView: WKWebView
    var loaded: CheckedContinuation<Void, Never>?

    init(base: String, token: String, outDir: URL, shots: [Shot]) {
        self.base = base
        self.token = token
        self.outDir = outDir
        self.shots = shots
        let config = WKWebViewConfiguration()
        config.websiteDataStore = .nonPersistent()
        // Scrollbars add noise to screenshots. Set through the CSSOM, which the page's CSP allows.
        config.userContentController.addUserScript(WKUserScript(
            source: "document.documentElement.style.scrollbarWidth = 'none'; document.documentElement.style.overflow = 'hidden';",
            injectionTime: .atDocumentEnd, forMainFrameOnly: true))
        webView = WKWebView(frame: CGRect(x: 0, y: 0, width: 1440, height: 900), configuration: config)
        super.init()
        webView.navigationDelegate = self
    }

    nonisolated func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        Task { @MainActor in
            self.loaded?.resume()
            self.loaded = nil
        }
    }

    func load(_ url: String) async {
        await withCheckedContinuation { continuation in
            loaded = continuation
            webView.load(URLRequest(url: URL(string: url)!))
        }
    }

    func pause(_ seconds: Double) async {
        try? await Task.sleep(for: .milliseconds(Int(seconds * 1000)))
    }

    func run() async {
        // Sign in once through the fragment link; the session cookie covers the rest.
        await load("\(base)/?theme=light#token=\(token)")
        await pause(1.5)

        for shot in shots {
            webView.appearance = NSAppearance(named: shot.dark ? .darkAqua : .aqua)
            webView.setFrameSize(shot.size)
            let theme = shot.theme
            if shot.route == "/signin" {
                _ = try? await webView.evaluateJavaScript("fetch('/api/logout', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: '{}'})")
                await pause(0.5)
                await load("\(base)/?theme=\(theme)")
            } else if shot.route.hasPrefix("/join/") {
                await load("\(base)\(shot.route)")
            } else {
                // A unique query forces a full load; a fragment-only change would never finish loading.
                await load("\(base)/?theme=\(theme)&shot=\(shot.name)#\(shot.route)")
            }
            await pause(2.0)
            if let click = shot.click {
                let label = click.replacingOccurrences(of: "'", with: "\\'")
                _ = try? await webView.evaluateJavaScript(
                    "[...document.querySelectorAll('button')].find(b => b.getAttribute('aria-label') === '\(label)' || b.textContent.trim() === '\(label)')?.click()")
                // Off-screen views don't advance CSS animations, so finish them by hand.
                _ = try? await webView.evaluateJavaScript("document.querySelectorAll('dialog').forEach(d => d.style.animation = 'none')")
                await pause(1.2)
            }

            let config = WKSnapshotConfiguration()
            config.rect = CGRect(origin: .zero, size: shot.size)
            config.afterScreenUpdates = true
            do {
                let image = try await webView.takeSnapshot(configuration: config)
                try save(image, as: shot.name)
                print("saved \(shot.name).png")
            } catch {
                print("failed \(shot.name): \(error)")
            }
        }
        NSApp.terminate(nil)
    }

    func save(_ image: NSImage, as name: String) throws {
        guard let tiff = image.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff),
              let png = rep.representation(using: .png, properties: [:])
        else { throw CocoaError(.fileWriteUnknown) }
        try png.write(to: outDir.appendingPathComponent("\(name).png"))
    }
}

let args = CommandLine.arguments
guard args.count >= 5 else {
    print("usage: swift screenshot.swift <base-url> <token> <out-dir> <name>=<route>[@WxH][:light|dark|classic|code|contrast] ...")
    exit(2)
}
let outDir = URL(fileURLWithPath: args[3], isDirectory: true)
try? FileManager.default.createDirectory(at: outDir, withIntermediateDirectories: true)

let app = NSApplication.shared
app.setActivationPolicy(.prohibited)
let shooter = MainActor.assumeIsolated {
    Shooter(base: args[1], token: args[2], outDir: outDir, shots: args[4...].map(parse))
}
Task { @MainActor in await shooter.run() }
app.run()
