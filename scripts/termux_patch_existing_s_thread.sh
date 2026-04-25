#!/data/data/com.termux/files/usr/bin/sh

set -eu

old_thread_id='019db3d5-3e00-7fd0-b723-0b48a2977b35'
new_thread_id='019dbcf2-d632-7770-9cbc-ae1f963eb2e0'
target_path="${PREFIX:-/data/data/com.termux/files/usr}/bin/s"

if [ ! -f "$target_path" ]; then
  echo "missing: $target_path" >&2
  exit 1
fi

sed -i "s/$old_thread_id/$new_thread_id/g" "$target_path"
hash -r

echo "patched $target_path"
grep 'thread-id' "$target_path"
