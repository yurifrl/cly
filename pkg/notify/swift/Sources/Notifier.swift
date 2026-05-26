import Foundation
import UserNotifications

// Notifier wraps UNUserNotificationCenter. Title + body + sound only.
// No action buttons, no categories — kept simple by design.
final class Notifier: NSObject, UNUserNotificationCenterDelegate {
    private let center = UNUserNotificationCenter.current()
    private let socket: SocketServer

    init(socket: SocketServer) {
        self.socket = socket
        super.init()
        center.delegate = self
    }

    func requestAuthorization(_ completion: @escaping (Bool) -> Void) {
        center.requestAuthorization(options: [.alert, .sound, .badge]) { granted, err in
            if let err = err {
                FileHandle.standardError.write("cly-notifier: auth error: \(err)\n".data(using: .utf8)!)
            }
            completion(granted)
        }
    }

    // handleLine is called by SocketServer for each newline-terminated JSON line.
    func handleLine(_ line: String) {
        guard let data = line.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let op = obj["op"] as? String else {
            socket.send(["op": "error", "message": "invalid json line"])
            return
        }
        switch op {
        case "send":
            schedule(payload: obj)
        case "ping":
            socket.send(["op": "pong"])
        default:
            socket.send(["op": "error", "message": "unknown op: \(op)"])
        }
    }

    private func schedule(payload: [String: Any]) {
        let group = (payload["group"] as? String) ?? "cly.default"
        let title = (payload["title"] as? String) ?? ""
        let body = (payload["body"] as? String) ?? ""
        let sound = payload["sound"] as? String

        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.threadIdentifier = group
        if let s = sound, !s.isEmpty {
            content.sound = UNNotificationSound(named: UNNotificationSoundName(rawValue: s + ".aiff"))
        } else {
            content.sound = .default
        }

        // Use the group as the request identifier so a new send replaces the
        // previous one (e.g. failing → recovered updates the same notification).
        let req = UNNotificationRequest(identifier: group, content: content, trigger: nil)
        center.add(req) { err in
            if let err = err {
                self.socket.send(["op": "error", "message": "schedule: \(err.localizedDescription)"])
            }
        }
    }

    // Show banners even when the daemon is the foreground process.
    func userNotificationCenter(_ center: UNUserNotificationCenter,
                                willPresent notification: UNNotification,
                                withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .sound, .list])
    }
}
