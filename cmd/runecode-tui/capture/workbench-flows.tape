Output "workbench-flows.gif"

Set Shell "bash"
Set FontSize 16
Set Width 2200
Set Height 1400
Set Theme "Dracula"

Require runecode-tui

Type "runecode-tui"
Enter
Sleep 1s

# quick route scan
Type "2"
Sleep 700ms
Type "3"
Sleep 700ms
Type "4"
Sleep 700ms
Type "5"
Sleep 700ms

# workbench command palette + object aware flow
Type ":"
Sleep 700ms
Type "inspect run"
Sleep 700ms
Enter
Sleep 900ms

# inspector/detail mode transitions
Type "v"
Sleep 700ms
Type "v"
Sleep 700ms

# selection-mode affordance
Type "\u0014"
Sleep 800ms
Type "\u0014"
Sleep 800ms

# shell pane controls
Type "s"
Sleep 700ms
Type "s"
Sleep 700ms
Type "Y"
Sleep 700ms

Type "q"
Sleep 300ms
