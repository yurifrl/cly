import Foundation

// Entry point. Parses --socket <path> and runs the daemon until the socket
// is disconnected or the process is terminated.
//
// Protocol: newline-delimited JSON over a Unix socket.
//   inbound  → {"op":"send","group":"...","title":"...","body":"...","sound":"...","actions":[{"id":"...","title":"..."}]}
//   inbound  → {"op":"ping"}
//   outbound → {"op":"action","group":"...","id":"..."}
//   outbound → {"op":"ready","authorized":true|false}
//   outbound → {"op":"pong"}
//   outbound → {"op":"error","message":"..."}
//
// Socket disconnect (parent exits) terminates the daemon cleanly.

let args = CommandLine.arguments
var socketPath: String? = nil

var i = 1
while i < args.count {
    let a = args[i]
    if a == "--socket" && i + 1 < args.count {
        socketPath = args[i + 1]
        i += 2
        continue
    }
    if a.hasPrefix("--socket=") {
        socketPath = String(a.dropFirst("--socket=".count))
        i += 1
        continue
    }
    i += 1
}

guard let sock = socketPath, !sock.isEmpty else {
    FileHandle.standardError.write("cly-notifier: missing --socket <path>\n".data(using: .utf8)!)
    exit(2)
}

let socket = SocketServer(path: sock)
let notifier = Notifier(socket: socket)

socket.onLine = { line in
    notifier.handleLine(line)
}

socket.onDisconnect = {
    // Parent went away. Exit cleanly; macOS will clean up scheduled
    // notifications shortly after the process dies.
    FileHandle.standardError.write("cly-notifier: parent disconnected, exiting\n".data(using: .utf8)!)
    exit(0)
}

notifier.requestAuthorization { granted in
    socket.send(["op": "ready", "authorized": granted])
}

socket.start()

// Run main loop forever; UNUserNotificationCenter callbacks need a runloop.
RunLoop.main.run()
