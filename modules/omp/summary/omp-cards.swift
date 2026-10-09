// omp-cards renders one summary card for the focused workspace. `cly omp
// summary` stores a positional, pipe-joined card in the workspace description:
//   0 title | 1 stamp | 2 ask | 3 goal | 4 response | 5 action | 6 compaction | 7+ timeline
// Empty hero slots are "~". cmux's sidebar interpreter mangles multibyte
// string surgery (hasPrefix/dropFirst mis-split the old two-rune glyph tags),
// so parsing here uses split + position + count only — no prefix tests, no
// string subscripts.

func sectionLabel(_ text: String) -> some View {
	return Text(text)
		.font(.system(size: 8, design: .monospaced))
		.fontWeight(.semibold)
		.foregroundColor(.tertiary)
}

ScrollView {
	VStack(alignment: .leading, spacing: 0) {
		ForEach(workspaces) { w in
			if w.selected {
				let description = w.description == nil ? "" : String(w.description)
				let lines = Array(description.split(separator: "|", omittingEmptySubsequences: true).prefix(20))
				VStack(alignment: .leading, spacing: 0) {
					VStack(alignment: .leading, spacing: 2) {
						if lines.count < 1 {
							Text(w.title).font(.system(size: 13)).bold().foregroundColor("#F97316").frame(maxWidth: .infinity, alignment: .leading)
						} else {
							Text(String(lines[0])).font(.system(size: 13)).bold().foregroundColor("#F97316").frame(maxWidth: .infinity, alignment: .leading)
						}
						if lines.count > 1 {
							if String(lines[1]).count > 1 {
								Text(String(lines[1])).font(.system(size: 9, design: .monospaced)).lineLimit(1).foregroundColor(.tertiary)
							} else {
								EmptyView()
							}
						}
					}
					.padding(.vertical, 2)

					if lines.count < 6 {
						Divider().opacity(0.35)
						Text("No OMP session data yet").font(.system(size: 11)).foregroundColor(.tertiary).padding(.vertical, 9)
					} else {
						ForEach(Array(lines.enumerated()), id: \.offset) { i, line in
							let text = String(line)
							if i == 2 && text.count > 1 {
								VStack(alignment: .leading, spacing: 4) {
									Divider().opacity(0.35)
									sectionLabel("LAST REQUEST")
									Text(text).font(.system(size: 12)).bold().lineLimit(2).foregroundColor(.primary).frame(maxWidth: .infinity, alignment: .leading)
								}.padding(.vertical, 8)
							} else if i == 3 && text.count > 1 {
								VStack(alignment: .leading, spacing: 4) {
									Divider().opacity(0.35)
									sectionLabel("GOAL")
									Text(text).font(.system(size: 11)).lineLimit(2).foregroundColor(.secondary).frame(maxWidth: .infinity, alignment: .leading)
								}.padding(.vertical, 8)
							} else if i == 4 && text.count > 1 {
								VStack(alignment: .leading, spacing: 4) {
									Divider().opacity(0.35)
									sectionLabel("RESPONSE")
									Text(text).font(.system(size: 11)).lineLimit(2).foregroundColor(.secondary).frame(maxWidth: .infinity, alignment: .leading)
								}.padding(.vertical, 8)
							} else if i == 5 && text.count > 1 {
								VStack(alignment: .leading, spacing: 4) {
									Divider().opacity(0.35)
									sectionLabel("NEEDS ACTION")
									Text(text).font(.system(size: 11)).fontWeight(.semibold).lineLimit(2).foregroundColor(.primary).frame(maxWidth: .infinity, alignment: .leading)
								}.padding(.vertical, 8)
							} else if i == 6 && text.count > 1 {
								VStack(alignment: .leading, spacing: 4) {
									Divider().opacity(0.35)
									sectionLabel("COMPACTED")
									Text(text).font(.system(size: 11, design: .monospaced)).lineLimit(3).foregroundColor(.tertiary).frame(maxWidth: .infinity, alignment: .leading)
								}.padding(.vertical, 8)
							} else {
								EmptyView()
							}
						}

						if lines.count > 7 {
							VStack(alignment: .leading, spacing: 6) {
								sectionLabel("TIMELINE")
								ForEach(Array(lines.enumerated()), id: \.offset) { i, line in
									let text = String(line)
									if i >= 7 && text.count > 1 {
										Text("• " + text).font(.system(size: 10, design: .monospaced)).frame(maxWidth: .infinity, alignment: .leading).foregroundColor(.primary)
									} else {
										EmptyView()
									}
								}
							}
							.padding(8)
							.background { RoundedRectangle(cornerRadius: 7).fill("#141416") }
							.overlay { RoundedRectangle(cornerRadius: 7).stroke("#343438", lineWidth: 1) }
							.padding(.top, 8)
						}
					}
				}
				.padding(10)
				.frame(maxWidth: .infinity, alignment: .leading)
				.background { RoundedRectangle(cornerRadius: 10).fill("#19191B") }
				.overlay { RoundedRectangle(cornerRadius: 10).stroke("#333337", lineWidth: 1) }
			}
		}
	}
	.padding(10)
}
