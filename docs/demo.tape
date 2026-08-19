# Regenerate docs/demo.gif:
#
#   vhs docs/demo.tape
#
# The tape seeds its own world in a hidden step, because agents go stale after 90
# seconds and a recording made against a world seeded even a minute earlier shows a
# screen full of dimmed, dead-looking agents.
#
# The world it records is seeded by scripts/demo.sh, so this file and that script have
# to stay in step. Both live in the repo because a recording has to be remade every
# time these views change, and the last one was left two releases behind.

Output docs/demo.gif

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 14
Set Width 1180
Set Height 560
Set Padding 20
Set Theme "Builtin Dark"
Set TypingSpeed 45ms

Hide
Type "PETS_SANDBOX=/tmp/pets-demo ./scripts/demo.sh --seed >/dev/null"
Enter
Sleep 12s
Type "export HOME=/tmp/pets-demo/home XDG_CONFIG_HOME=/tmp/pets-demo/home/.config XDG_STATE_HOME=/tmp/pets-demo/home/.local/state PATH=/tmp/pets-demo:$PATH PS1='$ '"
Enter
Type "clear"
Enter
Show

Sleep 700ms
Type "pets party"
Enter
Sleep 5s

Type "clear"
Enter
Sleep 300ms
Type "pets den"
Enter
Sleep 5s
