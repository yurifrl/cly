// omp-cards: renders each workspace's omp summary card (set via
// `cmux workspace-action --action set-description` by `cly omp summary`).
// The card is pipe-joined tagged segments (cmux squashes newlines out of
// workspace descriptions before the binding sees them):
//   ▸ last command (exit N) | ? user ask | ◆ goal
//   ● ai status | ✦ model | ⏱ stamp
// Each segment gets its own color; workspaces without a card show a plain row.

func lineTint(_ l) -> String {
  if l.hasPrefix("? ") { return "primary" }
  if l.hasPrefix("◆ ") { return "cyan" }
  if l.hasPrefix("● ") { return "#FF8800" }
  if l.hasPrefix("✦ ") { return "indigo" }
  if l.hasPrefix("⏱ ") { return "tertiary" }
  return "secondary"
}

func hasCard(_ w) -> Bool {
  let d = w.description == nil ? "" : String(w.description)
  return d != ""
}

func card(_ w) -> some View {
  let d = w.description == nil ? "" : String(w.description)
  let lines = d.split(separator: "|", omittingEmptySubsequences: true).prefix(7)
  return VStack(alignment: .leading, spacing: 2) {
    ForEach(Array(lines.enumerated()), id: \.offset) { _, l in
      Text(String(l)).font(.system(size: 10, design: .monospaced)).lineLimit(1).foregroundColor(lineTint(String(l)))
    }
  }
}

func unreadBadge(_ w) -> some View {
  return Text("\(w.unread)")
    .font(.system(size: 9, design: .monospaced))
    .foregroundColor(.orange)
    .padding(3)
    .background { Capsule().foregroundColor(.orange).opacity(0.2) }
}

func row(_ w) -> some View {
  return Button(action: { cmux("workspace.select", workspace_id: w.id) }) {
    VStack(alignment: .leading, spacing: 4) {
      HStack(spacing: 5) {
        Text(w.selected ? "●" : "○")
          .font(.system(size: 9))
          .foregroundColor(w.selected ? "#FF8800" : .secondary)
        Text(w.title).font(.system(size: 11)).lineLimit(1)
        Spacer()
        if w.unread > 0 {
          unreadBadge(w)
        }
      }
      if hasCard(w) {
        card(w)
      }
    }
    .padding(6)
    .background { RoundedRectangle(cornerRadius: 8).foregroundColor(w.selected ? "#2A2A2E" : "#1E1E20") }
  }
}

VStack(alignment: .leading, spacing: 8) {
  HStack(spacing: 6) {
    Image(systemName: "sparkles").foregroundColor("#FF8800")
    Text("omp").font(.headline)
    Spacer()
    Text(clock.time).font(.system(size: 10, design: .monospaced)).foregroundColor(.secondary)
  }
  .padding(4)
  Divider()
  Reorderable(workspaces, move: "workspace.reorder") { w in
    row(w)
  }
}
