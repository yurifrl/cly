package gitcommits

import "testing"

func seedFile(path string, status FileStatus) FileChange {
	return FileChange{Path: path, Status: status}
}

func TestBuildSeedsRenameAtomic(t *testing.T) {
	files := []FileChange{
		{Path: "pkg/ai/decisions.go", Status: StatusAdded},
		{Path: "pkg/ai/ai.go", Status: StatusModified},
		{Path: "pkg/new_name.go", Status: StatusRenamed, OldPath: "pkg/old_name.go"},
	}
	seeds := buildSeeds(files)
	if len(seeds) != 2 {
		t.Fatalf("seeds = %d, want 2", len(seeds))
	}
	for _, s := range seeds {
		if s.Area != "pkg" {
			continue
		}
		if len(s.Files) != 1 || s.Files[0].Path != "pkg/new_name.go" || s.Files[0].OldPath != "pkg/old_name.go" {
			t.Errorf("rename seed files = %+v", s.Files)
		}
	}
}

func TestBuildSeedsAreaClustering(t *testing.T) {
	files := []FileChange{
		seedFile("pkg/ai/client.go", StatusModified),
		seedFile("pkg/ai/client_test.go", StatusAdded),
		seedFile("modules/git-commits/planner.go", StatusModified),
		seedFile("README.md", StatusModified),
	}
	seeds := buildSeeds(files)
	if len(seeds) != 3 {
		t.Fatalf("seeds = %d, want 3", len(seeds))
	}
	areas := map[string]bool{}
	for _, s := range seeds {
		areas[s.Area] = true
	}
	for _, want := range []string{"pkg/ai", "modules/git-commits", "root"} {
		if !areas[want] {
			t.Errorf("missing area %q (got %v)", want, areas)
		}
	}
}

func TestBuildSeedsTypeInference(t *testing.T) {
	cases := []struct {
		name  string
		files []FileChange
		want  string
	}{
		{"test only", []FileChange{seedFile("pkg/a/x_test.go", StatusAdded), seedFile("pkg/a/y_test.go", StatusModified)}, "test"},
		{"feat has additions", []FileChange{seedFile("pkg/a/new.go", StatusAdded), seedFile("pkg/a/old.go", StatusModified)}, "feat"},
		{"fix otherwise", []FileChange{seedFile("pkg/a/old.go", StatusModified), seedFile("pkg/a/old2.go", StatusDeleted)}, "fix"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seeds := buildSeeds(tc.files)
			if len(seeds) != 1 {
				t.Fatalf("seeds = %d, want 1", len(seeds))
			}
			if seeds[0].Type != tc.want {
				t.Errorf("type = %q, want %q", seeds[0].Type, tc.want)
			}
		})
	}
}

func TestBuildSeedsTypeInferenceSplitAreas(t *testing.T) {
	docs := buildSeeds([]FileChange{seedFile("README.md", StatusModified), seedFile("docs/guide.md", StatusAdded)})
	if len(docs) != 2 {
		t.Fatalf("docs seeds = %d, want 2", len(docs))
	}
	for _, s := range docs {
		if s.Type != "docs" {
			t.Errorf("seed %q type = %q, want docs", s.Area, s.Type)
		}
	}
	build := buildSeeds([]FileChange{seedFile(".github/workflows/ci.yml", StatusModified), seedFile("Makefile", StatusAdded)})
	if len(build) != 2 {
		t.Fatalf("build seeds = %d, want 2", len(build))
	}
	for _, s := range build {
		if s.Type != "chore" {
			t.Errorf("seed %q type = %q, want chore", s.Area, s.Type)
		}
	}
}

func TestBuildSeedsDeterministicOrder(t *testing.T) {
	files := []FileChange{
		seedFile("pkg/ai/client.go", StatusModified),
		seedFile("modules/git-commits/planner.go", StatusModified),
		seedFile("pkg/ai/client_test.go", StatusModified),
	}
	a := buildSeeds(files)
	b := buildSeeds(files)
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Area != b[i].Area {
			t.Fatalf("order differs: %v vs %v", a, b)
		}
	}
	if a[0].Area != "pkg/ai" {
		t.Errorf("first seed area = %q", a[0].Area)
	}
}

func TestBuildSeedsLabels(t *testing.T) {
	files := []FileChange{
		seedFile("pkg/ai/client.go", StatusModified),
		seedFile("pkg/ai/client_test.go", StatusAdded),
	}
	seeds := buildSeeds(files)
	if len(seeds) != 1 {
		t.Fatalf("seeds = %d", len(seeds))
	}
	want := "feat changes in pkg/ai: client.go(M), client_test.go(A)"
	if seeds[0].Label != want {
		t.Errorf("label = %q, want %q", seeds[0].Label, want)
	}
}
