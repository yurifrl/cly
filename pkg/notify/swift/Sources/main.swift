import Foundation

// Entry point. Parses --socket <path> and runs the daemon until the parent
// disconnects from the socket.
//
// Protocol: newline-delimited JSON over a Unix socket.
//   inbound  → {"op":"send","group":"...","title":"...","body":"...","sound":"..."}
//   inbound  → {"op":"ping"}
//   outbound → {"op":"ready","authorized":true|false}
//   outbound → {"op":"pong"}
//   outbound → {"op":"error","message":"..."}

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

var authState: Bool? = nil
var clientConnected = false
var readySent = false

func emitReady() {
    if !clientConnected { return }
    let granted = authState ?? false
    socket.send(["op": "ready", "authorized": granted])
    readySent = true
}

socket.onLine = { line in
    notifier.handleLine(line)
}

socket.onDisconnect = {
    FileHandle.standardError.write("cly-notifier: parent disconnected, exiting\n".data(using: .utf8)!)
    exit(0)
}

socket.onConnect = {
    clientConnected = true
    emitReady()
}

notifier.requestAuthorization { granted in
    authState = granted
    if !readySent {
        emitReady()
    } else {
        socket.send(["op": "ready", "authorized": granted])
    }
}

socket.start()

RunLoop.main.run()
