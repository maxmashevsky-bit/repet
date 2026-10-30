#!/bin/sh
set -eu

if [ "$#" -gt 1 ]; then
  printf 'Usage: %s [progress-file]\n' "$0" >&2
  exit 2
fi
repet_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
repet_tracker=${1:-"$repet_root/docs/CODEX_PROGRESS.md"}
if [ ! -r "$repet_tracker" ]; then
  printf 'Не удалось прочитать трекер: %s\n' "$repet_tracker" >&2
  exit 1
fi

awk '
function trim(value) { sub(/^[[:space:]]+/, "", value); sub(/[[:space:]]+$/, "", value); return value }
function fail(message) { print "Ошибка трекера: " message > "/dev/stderr"; invalid = 1 }
BEGIN { FS = "|" }
{
  id = trim($2)
  if (id !~ /^[0-9][0-9]$/) next
  if (NF != 8) { fail("неверное число столбцов, ID " id); next }
  if (id + 0 > 19) { fail("ID вне диапазона 00–19: " id); next }
  if (seen[id]++) { fail("повтор ID " id); next }
  title = trim($3); status = trim($4); checked = trim($5); result = trim($6); blocker = trim($7)
  if (status != "TODO" && status != "IN_PROGRESS" && status != "DONE" && status != "BLOCKED") fail("неизвестный статус у " id ": " status)
  if (checked !~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$/) fail("неверная дата у " id)
  if (title == "" || result == "" || blocker == "") fail("пустая обязательная ячейка у " id)
  rows[id] = id "  " status "  " title "  [" checked "]"
  if (status == "BLOCKED") rows[id] = rows[id] "\n    Блокер: " blocker
  counts[status]++; total++
}
END {
  for (i = 0; i < 20; i++) { key = sprintf("%02d", i); if (!(key in seen)) fail("отсутствует ID " key) }
  if (invalid) exit 1
  print "Репет 2.0 — прогресс разработки"
  printf "Задач: %d; DONE: %d; IN_PROGRESS: %d; TODO: %d; BLOCKED: %d\n\n", total, counts["DONE"], counts["IN_PROGRESS"], counts["TODO"], counts["BLOCKED"]
  for (i = 0; i < 20; i++) print rows[sprintf("%02d", i)]
  print "\nDONE отражает критерии этапов, а не готовность всего продукта."
}' "$repet_tracker"
