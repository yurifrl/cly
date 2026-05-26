import Foundation
import UserNotifications

// Notifier wraps UNUserNotificationCenter. Categories are registered lazily
// per unique action set so each Send can have its own buttons. Action clicks
// are forwarded to the SocketServer as JSON.
final class Notifier: NSObject, UNUserNotificationCenterDelegate {
    private let center = UNUserNotificationCenter.current()
    private let socket: SocketServer
    private var registeredCategories: Set<String> = []

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
        let actionsRaw = payload["actions"] as? [[String: String]] ?? []

        let categoryID = registerCategory(group: group, actions: actionsRaw) { _ in }

        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.categoryIdentifier = categoryID
        content.threadIdentifier = group
        if let s = sound, !s.isEmpty {
            content.sound = UNNotificationSound(named: UNNotificationSoundName(rawValue: s + ".aiff"))
        } else {
            content.sound = .default
        }
        // Stash group on userInfo so the delegate can forward it back.
        content.userInfo = ["cly_group": group]

        // Use the group as the request identifier so a new send replaces the
        // previous one (failing → recovered updates the same notification).
        let req = UNNotificationRequest(identifier: group, content: content, trigger: nil)
        center.add(req) { err in
            if let err = err {
                self.socket.send(["op": "error", "message": "schedule: \(err.localizedDescription)"])
            }
        }
    }

    // registerCategory builds and caches a UNNotificationCategory keyed by
    // group + action shape so identical action sets don't re-register.
    private func registerCategory(group: String, actions: [[String: String]], completion: @escaping (String) -> Void) -> String {
        let actionSig = actions.map { ($0["id"] ?? "") + ":" + ($0["title"] ?? "") }.joined(separator: "|")
        let categoryID = "cly.\(group).\(actionSig.hashValue)"
        if registeredCategories.contains(categoryID) {
            completion(categoryID)
            return categoryID
        }
        let unActions = actions.compactMap { dict -> UNNotificationAction? in
            guard let id = dict["id"], let title = dict["title"] else { return nil }
            var options: UNNotificationActionOptions = []
            if id == "dismiss" { options.insert(.destructive) }
            return UNNotificationAction(identifier: id, title: title, options: options)
        }
        let cat = UNNotificationCategory(
            identifier: categoryID,
            actions: unActions,
            intentIdentifiers: [],
            options: [.customDismissAction]
        )
        // Replace the entire category set each time. UN merges by identifier.
        center.getNotificationCategories { existing in
            var merged = existing
            merged.insert(cat)
            self.center.setNotificationCategories(merged)
            self.registeredCategories.insert(categoryID)
            completion(categoryID)
        }
        return categoryID
    }

    // MARK: UNUserNotificationCenterDelegate

    // Show banners even when the daemon is the foreground process.
    func userNotificationCenter(_ center: UNUserNotificationCenter,
                                willPresent notification: UNNotification,
                                withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .sound, .list])
    }

    // Action button tapped (or notification clicked / dismissed).
    func userNotificationCenter(_ center: UNUserNotificationCenter,
                                didReceive response: UNNotificationResponse,
                                withCompletionHandler completionHandler: @escaping () -> Void) {
        let group = (response.notification.request.content.userInfo["cly_group"] as? String) ?? response.notification.request.identifier
        var actionID = response.actionIdentifier
        switch actionID {
        case UNNotificationDefaultActionIdentifier:
            actionID = "default"
        case UNNotificationDismissActionIdentifier:
            actionID = "dismiss"
        default:
            break
        }
        socket.send(["op": "action", "group": group, "id": actionID])
        completionHandler()
    }
}
