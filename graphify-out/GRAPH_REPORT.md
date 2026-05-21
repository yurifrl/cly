# Graph Report - cly  (2026-05-21)

## Corpus Check
- 464 files · ~225,012 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2965 nodes · 6910 edges · 38 communities detected
- Extraction: 57% EXTRACTED · 43% INFERRED · 0% AMBIGUOUS · INFERRED: 2958 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 39|Community 39]]

## God Nodes (most connected - your core abstractions)
1. `run()` - 297 edges
2. `Register()` - 197 edges
3. `render()` - 158 edges
4. `contains()` - 127 edges
5. `New()` - 105 edges
6. `WriteFile()` - 83 edges
7. `command` - 80 edges
8. `MkdirAll()` - 68 edges
9. `Dir` - 47 edges
10. `toLower()` - 44 edges

## Surprising Connections (you probably didn't know these)
- `Execute()` --calls--> `Execute()`  [INFERRED]
  cmd/root.go → modules/git-commits/executor.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/tui-daemon-combo/tui-daemon-combo.go → pkg/store/store.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/progress-animated/progress-animated.go → pkg/store/store.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/send-msg/send-msg.go → pkg/store/store.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/textinput/textinput.go → pkg/store/store.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (127): model, model, newCard(), reverse(), run(), model, run(), initialModel() (+119 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (124): ParseArgs(), TestParseArgs(), buildClaudeArgs(), runCapture(), TestBuildClaudeArgs(), TestRun_capture_routing(), ParsedArgs, activeLLMSummary() (+116 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (142): applyExtraFields(), GetAdapter(), TestGetAdapterPi(), TestPiAdapterGetConfigPath(), TestPiAdapterReadWriteConfig(), Config, DaemonStatus, IDEDef (+134 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (155): operationType, syncStats, BuildBatches(), buildFileAnalysis(), makeBatch(), OpenBrowser(), RustBundler, GetChangeset() (+147 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (75): editField, editModel, TransformKind, classifyBdErrAsCreate(), classifyBdErrAsList(), CreateBead(), isBdMissing(), isNoDB() (+67 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (134): ReconcileResult, backupBaseDir(), BackupExisting(), backupRootDir(), backupRootPath(), isCrossDeviceErr(), PlanBackupTarget(), resetBackupForTest() (+126 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (109): APIError, createBead(), getFileDiff(), getHealth(), j(), listLabels(), initialModel(), newFakeBd() (+101 more)

### Community 7 - "Community 7"
Cohesion: 0.03
Nodes (127): editResult, Entry, lsRow, Provider, Sessions, tuiDelegate, tuiItem, tuiMode (+119 more)

### Community 8 - "Community 8"
Cohesion: 0.04
Nodes (104): AliasEntry, FormatFish(), FormatFishCompletions(), GenerateAliases(), buildTestRoot(), TestFormatFish(), TestFormatFishCompletions(), TestFormatFishCompletionsSkipsOverrides() (+96 more)

### Community 9 - "Community 9"
Cohesion: 0.03
Nodes (103): Complete(), model, ApplyJsoncMapping(), updateLockJsoncEntry(), applyDiff(), executeCommand(), getConfigPath(), printJsoncResult() (+95 more)

### Community 10 - "Community 10"
Cohesion: 0.03
Nodes (68): Watcher, getRepos(), gotReposErrMsg, gotReposSuccessMsg, keymap, repo, extractTaps(), filterMasLines() (+60 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (78): addCalledFrom(), buildPiArgs(), findAllSessions(), findSession(), generateWords(), launchPi(), loadSession(), newSession() (+70 more)

### Community 12 - "Community 12"
Cohesion: 0.03
Nodes (30): init(), runAliExpress(), Scraper, GetAllExtractors(), Controller, init(), launchBrowser(), Options (+22 more)

### Community 13 - "Community 13"
Cohesion: 0.05
Nodes (60): pickerItem, pickerModel, simpleDelegate, SortOrder, runInside(), runOutside(), configureRuntime(), debugf() (+52 more)

### Community 14 - "Community 14"
Cohesion: 0.04
Nodes (44): run(), tick(), exitMsg, model, drawEllipse(), initialModel(), model, tickCmd() (+36 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (56): TestBeeepNotifier_Available(), classify(), defaultConfig(), detectProcOutliers(), dispatch(), expand(), formatTop(), loadConfig() (+48 more)

### Community 16 - "Community 16"
Cohesion: 0.05
Nodes (49): aiCandidate, aiRerankRequest, aiRerankResponse, candidate, indexedSession, jsonlFile, liveCache, liveResult (+41 more)

### Community 17 - "Community 17"
Cohesion: 0.05
Nodes (25): newAnthropicClient(), fieldID, initialModel(), keymap, loadTypesCmd(), model, picker, quickSelectExpireMsg (+17 more)

### Community 18 - "Community 18"
Cohesion: 0.06
Nodes (53): GetConfig(), ParseFormat(), readInput(), RenderCost(), RenderCustom(), RenderModel(), RenderStatusline(), runContext() (+45 more)

### Community 19 - "Community 19"
Cohesion: 0.06
Nodes (44): applyOverrideBlock(), applyProviderBlock(), CompleteFor(), descendMap(), errDisabled, HasAPIKey(), HasAPIKeyFor(), LoadConfig() (+36 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (16): add(), onKey(), remove(), TestSwitchContextPi(), defaultPrefills(), getSessionId(), getSummaryName(), readIdFromSessionFile() (+8 more)

### Community 21 - "Community 21"
Cohesion: 0.13
Nodes (35): JobApplyOptions, jobPaths, JobRetryConfig, JobStatus, onceState, TestApplyJobs_ForceRerunsOnce(), TestApplyJobs_OnceRunsOnlyOnce(), TestApplyJobs_StartupSkipsUnloadWhenNotLoaded() (+27 more)

### Community 22 - "Community 22"
Cohesion: 0.08
Nodes (12): initialModel(), item, model, getGradientColor(), interpolateColors(), model, run(), tick() (+4 more)

### Community 23 - "Community 23"
Cohesion: 0.21
Nodes (5): animate(), cellbuffer, drawEllipse(), frameMsg, model

### Community 24 - "Community 24"
Cohesion: 0.12
Nodes (15): Currency, DeliveryInfo, Price, ProductData, Quantity, RatingsData, Review, Shipping (+7 more)

### Community 25 - "Community 25"
Cohesion: 0.17
Nodes (5): initialModel(), item, itemDelegate, model, newStyles()

### Community 26 - "Community 26"
Cohesion: 0.22
Nodes (6): Execute(), init(), printVersion(), versionString(), GenerateBuildName(), TestGenerateBuildName()

### Community 27 - "Community 27"
Cohesion: 0.24
Nodes (6): initialModel(), model, processFinishedMsg, randomEmoji(), result, runPretendProcess()

### Community 28 - "Community 28"
Cohesion: 0.22
Nodes (4): checkServer(), errMsg, model, statusMsg

### Community 29 - "Community 29"
Cohesion: 0.22
Nodes (4): initialModel(), keyMap, model, newModel()

### Community 30 - "Community 30"
Cohesion: 0.38
Nodes (2): initialModel(), model

### Community 31 - "Community 31"
Cohesion: 0.67
Nodes (2): operationCompleteMsg, uiPreferencesSavedMsg

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (1): Extractor

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (1): Bundler

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (1): AddMCPOptions

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (1): contextSwitchedMsg

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (1): MCP

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (1): Notification

## Knowledge Gaps
- **230 isolated node(s):** `page`, `demoEntry`, `styles`, `result`, `tickMsg` (+225 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 30`** (7 nodes): `spinners.go`, `initialModel()`, `model`, `.Init()`, `.resetSpinner()`, `.Update()`, `.View()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (3 nodes): `operationCompleteMsg`, `uiPreferencesSavedMsg`, `operations.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (2 nodes): `Extractor`, `extractor.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (2 nodes): `Bundler`, `bundler.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (2 nodes): `AddMCPOptions`, `add.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (2 nodes): `contextSwitchedMsg`, `context_switch.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (2 nodes): `MCP`, `mcp.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (2 nodes): `Notification`, `types.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `run()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 19`, `Community 22`, `Community 26`?**
  _High betweenness centrality (0.261) - this node is a cross-community bridge._
- **Why does `render()` connect `Community 9` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 10`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 16`, `Community 17`, `Community 18`, `Community 20`, `Community 21`, `Community 22`, `Community 25`, `Community 27`, `Community 29`?**
  _High betweenness centrality (0.110) - this node is a cross-community bridge._
- **Why does `New()` connect `Community 6` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 21`, `Community 22`, `Community 25`, `Community 26`, `Community 27`, `Community 29`, `Community 30`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Are the 194 inferred relationships involving `run()` (e.g. with `TestGenerateBuildName()` and `main()`) actually correct?**
  _`run()` has 194 INFERRED edges - model-reasoned connections that need verification._
- **Are the 53 inferred relationships involving `Register()` (e.g. with `init()` and `renderGlowList()`) actually correct?**
  _`Register()` has 53 INFERRED edges - model-reasoned connections that need verification._
- **Are the 153 inferred relationships involving `render()` (e.g. with `.viewPalette()` and `.viewMenu()`) actually correct?**
  _`render()` has 153 INFERRED edges - model-reasoned connections that need verification._
- **Are the 124 inferred relationships involving `contains()` (e.g. with `TestGenerateBuildName()` and `.filterPalette()`) actually correct?**
  _`contains()` has 124 INFERRED edges - model-reasoned connections that need verification._