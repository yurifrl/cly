u will work in the folder research u will document the features of  ~/.local/bin/ai

  - make a list of the features
  - Think of a arquiteture
  - Arqtuiteture must be heavly configurable
  - amongst the features some are here some are not:
    - auto sync, be bidirectional
    - Keep a change history, just like git
    - make run as daemon with autosync (will need a way to handle conflics)

  - the goal of this script, is getting something that is in .ai and replicate that to .claude .opencode, so you can have a single config, and
  this takes care of syncing
  - agents, commands and skills are easy, there's some convertion but they are basicaly the same
  - we need a interpretation layer that is heavyly decopled, so we can plug nother ides, basicaly get the generic .ai things and convert, a
  convertion layer, i want to have 1 single file per ide easy to maitain easy to test
  - it saves to a bkp, i think we can use git as a depency to do that, remeber we will need to keep track of it in multiple projects
  - config, very important ~/.ai.json is the core thing, there i can config the paths for the global configs, i can set the default behavior to
  parse jsonc, jsonc can be evaluated so it accepts ${VARS} u can have a tag at the top to disable, it also is converted to json
  - today a .ai can becase a .opencode, in the cureture cursor codex etc, but it is only file to file, .AGENTS becase .CLAUDE.md inside a
  .claude, but lets say i want a CLAUDE.md in my root, synced from a file in .ai  maybe a .AGENTS but where do i put it? AGENTS in .ai will
  became .claude/CLAUDE.md, where do i put it? the CLAUDE.md in root dir?
  - about pattern, lets use claude patters for skills agents commands, and crush patter instead of CLAUDE.md i think is AGENTS.md plural, need
  cheking
  - tehres a bit of a hack, there is this folder ides/* is copied as is (unless jsonc) but this architeture is kinda ugly think a bit here


  now, make re
