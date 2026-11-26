#!/usr/bin/env bash
# Interactive demo (make demo): fills the fake pubnix with a couple of boards and posts,
# then opens a root shell. The container is thrown away on exit.
set -e
as() { su -s /bin/bash "$1" -c "$2" >/dev/null; }

as admin "bb init"
as alice "bb -c welcome && bb -l 0 && bb -a 'hello! this is the demo board' && bb -a 'say @bob in a message and it is highlighted for bob'"
as bob "bb -l 0 && bb -a 'hi alice' && bb -a 'rev sdrawkcab si siht'"
printf '0\nanon nobody knows who posted this\n\nq\n' | as alice "bb"
as admin "bb -c announcements"
printf 'pin+ 0\nq\n' | as admin "bb" # announcements sorts first: admin's home is read before alice's

cat <<'EOF'

  bb demo pubnix - users: admin (bb admin + moderator), alice, bob

    su - alice     become alice, then run: bb   (h for help)
    tmux           split panes with ctrl-b % to be two users at once
                   (each new pane starts as root - su - bob in it first)
    exit           leave a user / the demo (everything is thrown away)

EOF
exec bash
