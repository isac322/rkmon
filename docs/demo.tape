Output docs/demo.gif

Set FontSize 14
Set Width 1280
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Set PlaybackSpeed 1.0

Hide
Type "ssh -tt your-rk3588-host '~/.local/bin/rkmon --refresh=400ms --tiers=all'"
Enter
Sleep 3s
Show

# Baseline view
Sleep 4s

# Cycle through help tabs
Type "?"
Sleep 3s
Tab
Sleep 3s
Tab
Sleep 3s
Escape
Sleep 1s

# Demonstrate tier toggles
Type "k"
Sleep 1s
Type "s"
Sleep 1s
Type "i"
Sleep 2s
Type "i"
Sleep 1s
Type "s"
Sleep 1s
Type "k"
Sleep 2s

# Demonstrate scrolling
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
PageDown
Sleep 1s
PageUp
Sleep 1s

# Quit cleanly
Hide
Type "q"
Sleep 500ms
