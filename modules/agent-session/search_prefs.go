package agentsession

import "github.com/yurifrl/cly/pkg/config"

// Search filter preferences persist under this config prefix so the folder,
// role, sort, and AI toggles survive across invocations.
const (
	prefRole        = "modules.agent_session.search.role"
	prefSort        = "modules.agent_session.search.sort"
	prefFolderScope = "modules.agent_session.search.folder_current"
	prefAI          = "modules.agent_session.search.ai"
)

// searchPrefs holds the persisted filter state applied at model startup.
type searchPrefs struct {
	role string
	sort SortMode
}

func loadSearchPrefs() searchPrefs {
	role := config.GetString(prefRole)
	if role != roleUser && role != roleAssistant {
		role = roleAll
	}
	return searchPrefs{role: role, sort: sortFromLabel(config.GetString(prefSort))}
}

func nextRole(r string) string {
	switch r {
	case roleAll:
		return roleUser
	case roleUser:
		return roleAssistant
	default:
		return roleAll
	}
}

func sortFromLabel(label string) SortMode {
	switch label {
	case "hits":
		return SortByHits
	case "name":
		return SortByName
	case "path":
		return SortByPath
	default:
		return SortByDate
	}
}

func saveRole(r string)        { _ = config.Set(prefRole, r) }
func saveSort(s SortMode)      { _ = config.Set(prefSort, s.Label()) }
func saveFolderScope(cur bool) { _ = config.Set(prefFolderScope, cur) }
func saveAI(on bool)           { _ = config.Set(prefAI, on) }
