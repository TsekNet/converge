# converge demo
# Regenerate: vhs assets/demo.tape (from repo root)

Output assets/demo.gif

Set FontSize 16
Set Width 1280
Set Height 680
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
Type "go run ./assets/vhs-demo.go > /dev/null 2>&1"
Enter
Sleep 5s
Type "alias converge='go run ./assets/vhs-demo.go'"
Enter
Sleep 300ms
Type "clear"
Enter
Sleep 300ms
Show

Type "converge plan workstation"
Sleep 500ms
Enter
Sleep 8s
