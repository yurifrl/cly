import Foundation
import Darwin

// SocketServer listens on a Unix domain socket, accepts a single client
// (the parent Go process), and exchanges newline-delimited JSON messages.
//
// Threading model: a single dispatch queue handles I/O. JSON parsing and
// notifier dispatch happen on the main thread (RunLoop.main) so UN* delegate
// callbacks land in the same place.
final class SocketServer {
    private let path: String
    private var listenFD: Int32 = -1
    private var clientFD: Int32 = -1
    private let queue = DispatchQueue(label: "dev.yurifrl.cly.socket")
    private var readBuffer = Data()

    var onLine: ((String) -> Void)?
    var onDisconnect: (() -> Void)?
    var onConnect: (() -> Void)?

    init(path: String) {
        self.path = path
    }

    func start() {
        // Remove any stale socket file at the path.
        try? FileManager.default.removeItem(atPath: path)

        let fd = Darwin.socket(AF_UNIX, SOCK_STREAM, 0)
        guard fd >= 0 else {
            fatalError("socket(): \(String(cString: strerror(errno)))")
        }
        listenFD = fd

        var addr = sockaddr_un()
        addr.sun_family = sa_family_t(AF_UNIX)
        let pathBytes = Array(path.utf8)
        let maxLen = MemoryLayout.size(ofValue: addr.sun_path) - 1
        guard pathBytes.count <= maxLen else {
            fatalError("socket path too long: \(path)")
        }
        withUnsafeMutablePointer(to: &addr.sun_path) { ptr in
            ptr.withMemoryRebound(to: CChar.self, capacity: maxLen + 1) { cPtr in
                for (i, b) in pathBytes.enumerated() {
                    cPtr[i] = CChar(bitPattern: b)
                }
                cPtr[pathBytes.count] = 0
            }
        }
        let addrLen = socklen_t(MemoryLayout<sockaddr_un>.size)
        let bindResult = withUnsafePointer(to: &addr) { ptr -> Int32 in
            ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sa in
                Darwin.bind(fd, sa, addrLen)
            }
        }
        guard bindResult == 0 else {
            fatalError("bind(): \(String(cString: strerror(errno)))")
        }
        guard Darwin.listen(fd, 1) == 0 else {
            fatalError("listen(): \(String(cString: strerror(errno)))")
        }

        queue.async { [weak self] in
            self?.acceptLoop()
        }
    }

    private func acceptLoop() {
        let client = Darwin.accept(listenFD, nil, nil)
        guard client >= 0 else {
            FileHandle.standardError.write("cly-notifier: accept() failed\n".data(using: .utf8)!)
            return
        }
        clientFD = client
        DispatchQueue.main.async { [weak self] in
            self?.onConnect?()
        }
        readLoop()
    }

    private func readLoop() {
        var buf = [UInt8](repeating: 0, count: 4096)
        while true {
            let n = Darwin.read(clientFD, &buf, buf.count)
            if n <= 0 {
                Darwin.close(clientFD)
                clientFD = -1
                DispatchQueue.main.async { [weak self] in
                    self?.onDisconnect?()
                }
                return
            }
            readBuffer.append(buf, count: n)
            drainLines()
        }
    }

    private func drainLines() {
        while let nlIdx = readBuffer.firstIndex(of: 0x0A) {
            let lineData = readBuffer.subdata(in: 0..<nlIdx)
            readBuffer.removeSubrange(0...nlIdx)
            if let line = String(data: lineData, encoding: .utf8), !line.isEmpty {
                DispatchQueue.main.async { [weak self] in
                    self?.onLine?(line)
                }
            }
        }
    }

    // send marshals dict to JSON + newline and writes it to the client.
    // Writes synchronously — the dispatch queue is serial and blocked by
    // readLoop, so async dispatch would never run.
    func send(_ dict: [String: Any]) {
        guard clientFD >= 0 else { return }
        guard let data = try? JSONSerialization.data(withJSONObject: dict, options: []) else { return }
        var payload = Data(data)
        payload.append(0x0A)
        let fd = clientFD
        payload.withUnsafeBytes { raw in
            _ = Darwin.write(fd, raw.baseAddress, payload.count)
        }
    }
}
