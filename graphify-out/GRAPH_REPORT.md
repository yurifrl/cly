# Graph Report - cly  (2026-05-23)

## Corpus Check
- 494 files · ~241,337 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3212 nodes · 7763 edges · 51 communities detected
- Extraction: 54% EXTRACTED · 46% INFERRED · 0% AMBIGUOUS · INFERRED: 3534 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 52|Community 52]]

## God Nodes (most connected - your core abstractions)
1. `run()` - 303 edges
2. `Register()` - 200 edges
3. `render()` - 165 edges
4. `contains()` - 134 edges
5. `New()` - 113 edges
6. `WriteFile()` - 92 edges
7. `command` - 84 edges
8. `MkdirAll()` - 78 edges
9. `Dir` - 50 edges
10. `toLower()` - 47 edges

## Surprising Connections (you probably didn't know these)
- `Execute()` --calls--> `Execute()`  [INFERRED]
  cmd/root.go → modules/git-commits/executor.go
- `run()` --calls--> `WithTaskListID()`  [INFERRED]
  modules/llm-chat/cmd.go → pkg/session/session.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/progress-animated/progress-animated.go → pkg/store/store.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/send-msg/send-msg.go → pkg/store/store.go
- `initialModel()` --calls--> `New()`  [INFERRED]
  modules/demo/progress-static/progress-static.go → pkg/store/store.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (130): editField, editModel, model, model, run(), run(), run(), model (+122 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (136): buildClaudeArgs(), runCapture(), TestBuildClaudeArgs(), TestRun_capture_routing(), activeLLMSummary(), addFlags(), BuildExtraCompletions(), bumpSemver() (+128 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (181): TestPiAdapterGetConfigPath(), ReconcileResult, backupBaseDir(), BackupExisting(), backupRootDir(), backupRootPath(), isCrossDeviceErr(), PlanBackupTarget() (+173 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (142): applyExtraFields(), GetAdapter(), TestGetAdapterPi(), TestPiAdapterReadWriteConfig(), Config, DaemonStatus, IDEDef, logField (+134 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (146): editResult, Entry, lsRow, Provider, Sessions, tuiDelegate, tuiItem, tuiMode (+138 more)

### Community 5 - "Community 5"
Cohesion: 0.02
Nodes (93): getRepos(), gotReposErrMsg, gotReposSuccessMsg, initialModel(), keymap, repo, parallelModel, pkgResultMsg (+85 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (145): operationType, syncStats, BuildBatches(), buildFileAnalysis(), makeBatch(), OpenBrowser(), GetChangeset(), gitExec() (+137 more)

### Community 7 - "Community 7"
Cohesion: 0.03
Nodes (99): ClaudeVerbose(), createConfigCmd(), createHookCmd(), createSoundCmd(), CmuxSurfaceID(), CmuxTabID(), CmuxWorkspaceID(), InCmux() (+91 more)

### Community 8 - "Community 8"
Cohesion: 0.04
Nodes (107): AliasEntry, FormatFish(), FormatFishCompletions(), GenerateAliases(), buildTestRoot(), TestFormatFish(), TestFormatFishCompletions(), TestFormatFishCompletionsSkipsOverrides() (+99 more)

### Community 9 - "Community 9"
Cohesion: 0.03
Nodes (109): ApplyJsoncMapping(), updateLockJsoncEntry(), TestDiff(), TestDiffByBaseName(), expandTilde(), hasJobName(), isValidJobName(), parseJobLine() (+101 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (55): init(), runAliExpress(), Scraper, GetAllExtractors(), Controller, init(), launchBrowser(), Options (+47 more)

### Community 11 - "Community 11"
Cohesion: 0.03
Nodes (49): run(), tick(), animate(), cellbuffer, drawEllipse(), frameMsg, model, exitMsg (+41 more)

### Community 12 - "Community 12"
Cohesion: 0.05
Nodes (70): APIError, createBead(), getFileDiff(), getHealth(), j(), listLabels(), newFakeBd(), TestCreateBead_BdMissing() (+62 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (46): newAnthropicClient(), fieldID, initialModel(), keymap, loadTypesCmd(), model, picker, quickSelectExpireMsg (+38 more)

### Community 14 - "Community 14"
Cohesion: 0.05
Nodes (63): pickerItem, pickerModel, simpleDelegate, SortOrder, runInside(), runOutside(), configureRuntime(), debugf() (+55 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (50): aiCandidate, aiRerankRequest, aiRerankResponse, candidate, indexedSession, jsonlFile, liveCache, liveResult (+42 more)

### Community 16 - "Community 16"
Cohesion: 0.06
Nodes (41): NewBrewBundler(), baseBundler, checkCmd(), cleanupCmd(), getBundleFile(), getEditor(), JsBundler, openInEditor() (+33 more)

### Community 17 - "Community 17"
Cohesion: 0.05
Nodes (52): TestBeeepNotifier_Available(), classify(), defaultConfig(), detectProcOutliers(), dispatch(), expand(), formatTop(), loadConfig() (+44 more)

### Community 18 - "Community 18"
Cohesion: 0.06
Nodes (53): GetConfig(), ParseFormat(), readInput(), RenderCost(), RenderCustom(), RenderModel(), RenderStatusline(), runContext() (+45 more)

### Community 19 - "Community 19"
Cohesion: 0.05
Nodes (44): newSetyError(), Entry, JSONOutput, JSONSection, buildSessionPath(), encodeCwd(), errInvalidBoolErr, extractName() (+36 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (43): applyOverrideBlock(), applyProviderBlock(), Complete(), CompleteFor(), descendMap(), errDisabled, HasAPIKey(), HasAPIKeyFor() (+35 more)

### Community 21 - "Community 21"
Cohesion: 0.06
Nodes (17): add(), onKey(), remove(), TestSwitchContextPi(), defaultPrefills(), getSessionId(), getSummaryName(), piClyExtension() (+9 more)

### Community 22 - "Community 22"
Cohesion: 0.1
Nodes (26): addCalledFrom(), buildPiArgs(), findAllSessions(), findSession(), generateWords(), launchPi(), loadSession(), newSession() (+18 more)

### Community 23 - "Community 23"
Cohesion: 0.12
Nodes (15): Currency, DeliveryInfo, Price, ProductData, Quantity, RatingsData, Review, Shipping (+7 more)

### Community 24 - "Community 24"
Cohesion: 0.22
Nodes (13): TransformKind, hasNoInterpolation(), interpolateEnv(), removeTrailingCommas(), StripAllowedTools(), stripJSONCComments(), TestHasNoInterpolation(), TestInterpolateEnv() (+5 more)

### Community 25 - "Community 25"
Cohesion: 0.2
Nodes (1): Watcher

### Community 26 - "Community 26"
Cohesion: 0.26
Nodes (12): parseJSON(), parseJSONC(), ParseMCPFile(), parseYAML(), stripJSONComments(), stripTrailingCommas(), TestParseMCPFile_FlatFormat(), TestParseMCPFile_JSONC_McpServersWrapper() (+4 more)

### Community 27 - "Community 27"
Cohesion: 0.24
Nodes (11): classifyBdErrAsCreate(), classifyBdErrAsList(), CreateBead(), isNoDB(), ListLabels(), normalizePriority(), TestNormalizePriority(), BdRunner (+3 more)

### Community 28 - "Community 28"
Cohesion: 0.39
Nodes (10): createClaudeCmd(), createInstallCmd(), createRemoveCmd(), createVerifyCmd(), getHooksFromConfig(), getSettingsPath(), installHooks(), readSettings() (+2 more)

### Community 29 - "Community 29"
Cohesion: 0.25
Nodes (6): getGradientColor(), interpolateColors(), model, run(), tick(), tickMsg

### Community 30 - "Community 30"
Cohesion: 0.25
Nodes (5): model, tickMsg, initialModel(), run(), tick()

### Community 31 - "Community 31"
Cohesion: 0.22
Nodes (6): Execute(), init(), printVersion(), versionString(), GenerateBuildName(), TestGenerateBuildName()

### Community 32 - "Community 32"
Cohesion: 0.31
Nodes (4): model, styles, initialModel(), run()

### Community 33 - "Community 33"
Cohesion: 0.29
Nodes (3): initialModel(), model, tickMsg

### Community 34 - "Community 34"
Cohesion: 0.33
Nodes (2): model, SleepPrintln()

### Community 35 - "Community 35"
Cohesion: 0.33
Nodes (2): model, run()

### Community 36 - "Community 36"
Cohesion: 0.33
Nodes (2): model, run()

### Community 37 - "Community 37"
Cohesion: 0.33
Nodes (1): model

### Community 38 - "Community 38"
Cohesion: 0.33
Nodes (1): model

### Community 39 - "Community 39"
Cohesion: 0.33
Nodes (1): model

### Community 40 - "Community 40"
Cohesion: 0.33
Nodes (1): model

### Community 41 - "Community 41"
Cohesion: 0.33
Nodes (1): model

### Community 42 - "Community 42"
Cohesion: 0.33
Nodes (3): formatMCPCount(), TestFormatMCPCount(), contextOption

### Community 43 - "Community 43"
Cohesion: 0.6
Nodes (4): demoEntry, getDemos(), newOverlayModel(), runOverlay()

### Community 44 - "Community 44"
Cohesion: 0.67
Nodes (2): operationCompleteMsg, uiPreferencesSavedMsg

### Community 45 - "Community 45"
Cohesion: 1.0
Nodes (1): Extractor

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (1): Bundler

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (1): AddMCPOptions

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (1): contextSwitchedMsg

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (1): MCP

### Community 52 - "Community 52"
Cohesion: 1.0
Nodes (1): Notification

## Knowledge Gaps
- **245 isolated node(s):** `page`, `demoEntry`, `styles`, `result`, `tickMsg` (+240 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 25`** (15 nodes): `Watcher`, `.Add()`, `.AddTargetDir()`, `.debounce()`, `.debounceReverse()`, `.fire()`, `.fireReverse()`, `.isSuppressed()`, `.isTargetPath()`, `.Run()`, `.SetReverseSync()`, `.stopTimers()`, `.SuppressBriefly()`, `watcher.go`, `NewWatcher()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (7 nodes): `sequence.go`, `initialModel()`, `model`, `.Init()`, `.Update()`, `.View()`, `SleepPrintln()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (7 nodes): `model`, `.describeCursor()`, `.Init()`, `.Update()`, `.View()`, `run()`, `cursorstyle.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (6 nodes): `model`, `.Init()`, `.Update()`, `.View()`, `run()`, `colorprofile.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (6 nodes): `set-window-title.go`, `initialModel()`, `model`, `.Init()`, `.Update()`, `.View()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (6 nodes): `initialModel()`, `model`, `.Init()`, `.Update()`, `.View()`, `focus-blur.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (6 nodes): `suspend.go`, `initialModel()`, `model`, `.Init()`, `.Update()`, `.View()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (6 nodes): `mouse.go`, `initialModel()`, `model`, `.Init()`, `.Update()`, `.View()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (6 nodes): `window-size.go`, `initialModel()`, `model`, `.Init()`, `.Update()`, `.View()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (3 nodes): `operationCompleteMsg`, `uiPreferencesSavedMsg`, `operations.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (2 nodes): `Extractor`, `extractor.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (2 nodes): `Bundler`, `bundler.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (2 nodes): `AddMCPOptions`, `add.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (2 nodes): `contextSwitchedMsg`, `context_switch.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (2 nodes): `MCP`, `mcp.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (2 nodes): `Notification`, `types.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `run()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 20`, `Community 22`, `Community 24`, `Community 27`, `Community 29`, `Community 30`, `Community 31`, `Community 32`, `Community 35`, `Community 36`, `Community 42`, `Community 43`?**
  _High betweenness centrality (0.321) - this node is a cross-community bridge._
- **Why does `New()` connect `Community 5` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 20`, `Community 22`, `Community 31`?**
  _High betweenness centrality (0.124) - this node is a cross-community bridge._
- **Why does `render()` connect `Community 0` to `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 18`, `Community 19`, `Community 21`, `Community 22`, `Community 28`, `Community 29`, `Community 30`?**
  _High betweenness centrality (0.081) - this node is a cross-community bridge._
- **Are the 200 inferred relationships involving `run()` (e.g. with `TestGenerateBuildName()` and `main()`) actually correct?**
  _`run()` has 200 INFERRED edges - model-reasoned connections that need verification._
- **Are the 55 inferred relationships involving `Register()` (e.g. with `renderGlowList()` and `renderGroceryList()`) actually correct?**
  _`Register()` has 55 INFERRED edges - model-reasoned connections that need verification._
- **Are the 160 inferred relationships involving `render()` (e.g. with `.viewPalette()` and `.viewMenu()`) actually correct?**
  _`render()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **Are the 131 inferred relationships involving `contains()` (e.g. with `TestGenerateBuildName()` and `.filterPalette()`) actually correct?**
  _`contains()` has 131 INFERRED edges - model-reasoned connections that need verification._