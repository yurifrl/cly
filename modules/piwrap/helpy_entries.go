// helpy_entries.go registers piwrap's flag descriptions with the
// global helpy registry. Loaded automatically via package init.
//
// Adding a new piwrap flag without registering it here means it
// won't appear in `cly pi --helpy` — review-catchable.
package piwrap

import "github.com/yurifrl/cly/pkg/helpy"

func init() {
	helpy.Register(helpy.Entry{
		Section:     "Naming",
		Flags:       []string{"-n", "--name"},
		Value:       "<name>",
		Description: "Set session name. Renames cmux tab, exports $CLY_SESSION_NAME,\nand pins a pi session file at ~/.pi/agent/sessions/<encoded-cwd>/\n<prefix>-<kebab-name>.jsonl.",
		ConfigKeys:  []string{"modules.piwrap.session_file_name_prefix"},
		EnvVars:     []string{"CLY_SESSION_NAME"},
		Examples: []string{
			"cly pi -n refactor-auth",
			`cly pi --name "My Work" -p "summarize"`,
		},
		Order: 1,
	})

	helpy.Register(helpy.Entry{
		Section:     "Session Import",
		Flags:       []string{"--sety session_import.id", "-y session_import.id"},
		Value:       "<UUID|prefix|path>",
		Description: "Fork an existing pi session into this cwd under -n's name.\nSource resolved from the current cwd's session dir, or all dirs\nwhen modules.piwrap.session_import.search_scope=all.",
		Requires:    []string{"-n"},
		ConfigKeys: []string{
			"modules.piwrap.session_import.search_scope",
			"modules.piwrap.session_import.quarantine_dir",
		},
		Errors: []string{
			"SETY_NAME_REQUIRED",
			"SETY_IMPORT_ID_TOO_SHORT",
			"SETY_IMPORT_NOT_FOUND",
			"SETY_IMPORT_AMBIGUOUS",
			"SETY_IMPORT_FAILED",
		},
		Examples: []string{
			"cly pi -n bug-repro --sety session_import.id=019e5057",
			`cly pi -n bug-repro --sety session_import.id="$UUID" --sety session_import.override=true`,
			`cly pi -n bug-repro --sety session_import.id="$UUID" --dry-run`,
		},
		Order: 2,
	})

	helpy.Register(helpy.Entry{
		Section:     "Session Import",
		Flags:       []string{"--sety session_import.override"},
		Value:       "true|false",
		Description: "On filename conflict at the target, move the existing file\nto the quarantine dir and proceed (true), or fail with\nSETY_IMPORT_CONFLICT (false). Defaults to\nmodules.piwrap.session_import.override (false).",
		ConfigKeys:  []string{"modules.piwrap.session_import.override"},
		Errors:      []string{"SETY_IMPORT_CONFLICT", "SETY_PARSE"},
		Order:       3,
	})

	helpy.Register(helpy.Entry{
		Section:     "Dry Run",
		Flags:       []string{"--dry-run"},
		Description: "Validate piwrap-side actions and print the planned operation\nas JSON. No filesystem writes, no pi exec.",
		Order:       4,
	})

	helpy.Register(helpy.Entry{
		Section:     "Help",
		Flags:       []string{"--helpy"},
		Description: "Print this cheat sheet (text). Add `-o json` for machine output.\nFlags listed above are recognised by piwrap; everything else is\nforwarded to the underlying pi binary unchanged.",
		Examples: []string{
			"cly pi --helpy",
			"cly pi --helpy -o json",
		},
		Order: 5,
	})
}
