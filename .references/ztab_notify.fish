function ztab_notify -d "Add emoji/text to Zellij tab name"
    # Check if we're in a Zellij session
    if not set -q ZELLIJ
        return 1
    end

    # Check if arguments were provided
    if test (count $argv) -eq 0
        return 1
    end

    set emoji ""
    set ring_bell 0
    set clear_mode 0

    # Parse arguments
    for arg in $argv
        switch $arg
            case '--done' '-d'
                set emoji " ✅"
            case '--error' '-e'
                set emoji " ❌"
            case '--warning' '-w'
                set emoji " ⚠️"
            case '--running' '-r'
                set emoji " ⚡"
            case '--bell' '-b'
                set ring_bell 1
            case '--clear' '-c'
                set clear_mode 1
            case '--*' '-*'
                # Unknown flag, do nothing
                return 1
            case '*'
                set emoji " $arg"
        end
    end

    # If no emoji and not clear mode, do nothing
    if test -z "$emoji" -a $clear_mode -eq 0
        return 1
    end

    # Get current tab name using workaround
    set tabs_before (zellij action query-tab-names)
    set temp_marker "__TEMP_MARKER_$$__"
    zellij action rename-tab "$temp_marker"
    sleep 0.05
    set tabs_after (zellij action query-tab-names)

    # Find original name
    set original_name ""
    for tab in (string split ' ' $tabs_before)
        if not string match -q "*$tab*" $tabs_after
            set original_name $tab
            break
        end
    end

    if test -z "$original_name"
        set original_name "Tab"
    end

    # Remove existing status emojis
    set cleaned_name (string replace -r ' [🔴✅❌⚠️⚡💼🎉]+$' '' $original_name)
    set cleaned_name (string trim -r $cleaned_name)

    # Apply action
    if test $clear_mode -eq 1
        zellij action rename-tab "$cleaned_name"
        echo "Cleared: '$original_name' → '$cleaned_name'"
    else
        set new_name "$cleaned_name$emoji"
        zellij action rename-tab "$new_name"
        echo "Updated: '$original_name' → '$new_name'"
    end

    # Ring bell if requested
    if test $ring_bell -eq 1
        printf '\a'
    end
end
