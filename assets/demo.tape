# converge demo
# Regenerate: vhs assets/demo.tape (from repo root)

Output assets/demo.gif

Set FontSize 16
Set Width 1280
Set Height 720
Set Padding 24
Set Margin 28
Set MarginFill "#181825"
Set BorderRadius 14
Set WindowBar Colorful
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms
Set CursorBlink false

Hide
Type "cd ~/projects/code/github/converge"
Enter
Sleep 300ms
Type "go run ./assets/vhs-demo.go plan baseline > /dev/null 2>&1"
Enter
Sleep 5s
Type "alias converge='go run ./assets/vhs-demo.go'"
Enter
Sleep 300ms
Type "clear"
Enter
Sleep 300ms
Show

# Plan: show what would change
Type "converge plan baseline"
Sleep 500ms
Enter
Sleep 6s

Sleep 2s

# Apply: converge and exit
Type "converge serve baseline --timeout 1s"
Sleep 500ms
Enter
Sleep 8s

Sleep 2s
