#!/usr/bin/env bash
# Drives bb as several real users on a fake pubnix. Runs as root inside the smoke container.
set -u

pass=0
fail=0
ok() { echo "PASS  $1"; pass=$((pass + 1)); }
bad() {
	echo "FAIL  $1"
	[ -n "${2:-}" ] && printf '%s\n' "$2" | tail -25 | sed 's/^/      | /'
	fail=$((fail + 1))
}

# run USER CMD [STDIN] - run CMD as USER with STDIN (printf %b escapes) and put the output in $OUT.
# A crash or a hang counts as a failure on its own.
run() {
	local u=$1 cmd=$2 in=${3:-}
	OUT=$(printf '%b' "$in" | timeout 30 su -s /bin/bash "$u" -c "$cmd" 2>&1)
	local code=$?
	if [ $code -eq 124 ]; then
		bad "$u: '$cmd' hung" "$OUT"
	elif grep -q '^panic:' <<<"$OUT"; then
		bad "$u: '$cmd' crashed" "$OUT"
	fi
}
has() { if grep -qF -- "$3" <<<"$2"; then ok "$1"; else bad "$1 (expected: $3)" "$2"; fi; }
hasnt() { if grep -qF -- "$3" <<<"$2"; then bad "$1 (unexpected: $3)" "$2"; else ok "$1"; fi; }
count() { grep -cF -- "$2" <<<"$1"; }
index_of() { run alice "bb -p"; grep -o "[0-9]*) $1 |" <<<"$OUT" | head -1 | cut -d')' -f1; }

today=$(date +%F)

echo "== setup"
run alice "bb -p"
has "bb refuses to run before the admin has initialised it" "$OUT" "has not yet ran / initiated bb"

run admin "bb init"
has "admin can initialise bb" "$OUT" "bb is initiated"
for f in bban0n.json bbp1n.json; do
	mode=$(stat -c %a "/home/admin/.bbmod/$f" 2>/dev/null)
	if [ "$mode" = 666 ]; then ok "shared $f is writable by everyone ($mode)"; else bad "shared $f mode is '$mode', want 666"; fi
done

echo "== boards and posting"
run alice "bb -c general"
has "alice creates a board" "$OUT" "saved general"
[ -f /home/alice/.bb/general.json ] && ok "board file lands in alice's home" || bad "no /home/alice/.bb/general.json"

run admin "bb -p" # admin's snapshot, for 'bb u' later
has "admin sees alice's board" "$OUT" "0) general | author: alice"

run alice "bb -l 0 && bb -a 'hello from alice' && bb -a lol"
run bob "bb -l 0 && bb -a lol"
run alice "bb -a 'see gemini://127.0.0.1:1/'" # unreachable gemini host used to crash bb
run alice "bb -l 0"
has "alice's post shows" "$OUT" "<alice> hello from alice"
has "bob's post merges onto alice's board" "$OUT" "<bob> lol"
if [ "$(count "$OUT" "> lol")" = 2 ]; then ok "same text from two users is kept twice"; else bad "'lol' from alice and bob should both show" "$OUT"; fi
has "posting an unreachable gemini link works" "$OUT" "see gemini://127.0.0.1:1/"
[ -f /home/bob/.bb/general.json ] && ok "bob's post lands in bob's home, not alice's" || bad "no /home/bob/.bb/general.json"

run admin "bb u"
has "'bb u' reports new posts" "$OUT" "Board updates on: general"

echo "== interactive commands"
run alice "bb" "0\nfile is broken\nreview this please\n\nq\n"
run alice "bb -l 0"
has "'file...' posts instead of filtering" "$OUT" "<alice> file is broken"
has "'review...' posts as typed" "$OUT" "<alice> review this please"
run alice "bb" "0\nrev olleh\n" # ends with EOF (ctrl-d) instead of q
run alice "bb -l 0"
has "'rev ' still reverses, ctrl-d exits" "$OUT" "<alice> hello"

echo "== anon posts from two live sessions"
mkfifo /tmp/bob.in && chmod 666 /tmp/bob.in
su -s /bin/bash bob -c "bb < /tmp/bob.in > /tmp/bob.out 2>&1" &
bobpid=$!
exec 3>/tmp/bob.in
printf '0\n' >&3 # bob opens the board and sits there
sleep 1
run alice "bb" "0\nanon from alice\n\nq\n" # alice posts anonymously meanwhile
printf 'r\nanon from bob\n\nq\n' >&3        # bob refreshes (used to overwrite alice's post) and posts
exec 3>&-
timeout 30 tail --pid=$bobpid -f /dev/null || bad "bob's session did not exit"
run alice "bb -l 0"
has "alice's anon post survives bob's session" "$OUT" "<???> from alice"
has "bob's anon post shows" "$OUT" "<???> from bob"

echo "== pins"
run admin "bb" "pin+ 0\nq\n"
run alice "bb -p"
has "admin can pin a board" "$OUT" "general | author: alice | $today **PINNED**"
run alice "bb" "pin- 0\nq\n"
run alice "bb -p"
has "non-moderators can't unpin from bb" "$OUT" "**PINNED**"

echo "== hostile board file"
cat >/home/bob/.bb/general.json <<EOF
{"Title":"general","Owner":"alice","Date":"$today","Contents":[[],["one field"],
 ["$today 23:59:59","\u001b]0;pwned\u0007\u001b[2Jgotcha","bob"]]}
EOF
run alice "bb -l 0"
has "malformed rows don't crash bb" "$OUT" "<bob> gotcha"
hasnt "escape codes from other users are stripped" "$OUT" "pwned"

echo "== deleting"
run alice "bb -c scratch"
idx=$(index_of scratch)
run alice "bb" "del $idx\nq\n"
[ ! -f /home/alice/.bb/scratch.json ] && ok "owner can delete an untouched board" || bad "scratch.json still exists"
run alice "bb -p"
hasnt "deleted board is gone from the index" "$OUT" ") scratch |"

idx=$(index_of general)
run bob "bb" "del $idx\nq\n"
has "bob can't delete alice's board" "$OUT" "you are not owner of topic"
[ -f /home/alice/.bb/general.json ] && ok "alice's board file is untouched" || bad "alice's general.json was deleted"

echo "== moderation"
run admin "bb mod + bob"
run bob "bb mod"
has "admin can make bob a moderator" "$OUT" "you are a bb moderator"
run alice "bb mod"
has "alice is not a moderator" "$OUT" "you are not a moderator"
run alice "bb mod del $idx"
run alice "bb -p"
has "non-moderators can't archive" "$OUT" "general | author: alice"
run bob "bb mod del $idx"
run alice "bb -p"
hasnt "a moderator can archive a board" "$OUT" "general | author: alice"

echo
echo "== $pass passed, $fail failed"
[ "$fail" -eq 0 ]
