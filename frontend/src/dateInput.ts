// Formats/parses <input type="date"|"datetime-local"> values against a given
// IANA timezone (the viewer's own timezone for record dates — see AGENTS.md
// convention: input/display always use the viewer's timezone, only the
// backend's calculations use the group timezone). Never string-slice an ISO
// instant to get a local date: an ISO string's calendar day is UTC's, not the
// viewer's, and slicing silently produces the wrong day whenever the viewer's
// offset pushes local midnight to the other side of the UTC day boundary.

function parts(iso: string, timeZone: string) {
  const formatter = new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })
  const found: Record<string, string> = {}
  for (const part of formatter.formatToParts(new Date(iso))) if (part.type !== 'literal') found[part.type] = part.value
  return found
}

/** ISO instant -> "YYYY-MM-DD" in timeZone, for <input type="date">. */
export function toDateInput(iso: string, timeZone: string): string {
  if (!iso) return ''
  const p = parts(iso, timeZone)
  return `${p.year}-${p.month}-${p.day}`
}

/** ISO instant -> "YYYY-MM-DDTHH:mm" in timeZone, for <input type="datetime-local">. */
export function toDateTimeInput(iso: string, timeZone: string): string {
  if (!iso) return ''
  const p = parts(iso, timeZone)
  return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`
}

/** "YYYY-MM-DD" in timeZone -> ISO instant at local midnight. */
export function fromDateInput(value: string, timeZone: string): string {
  return fromDateTimeInput(`${value}T00:00`, timeZone)
}

/** "YYYY-MM-DDTHH:mm" in timeZone -> ISO instant. */
export function fromDateTimeInput(value: string, timeZone: string): string {
  if (!value) return ''
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/.exec(value)
  if (!match) return new Date(value).toISOString()
  const [, year, month, day, hour, minute] = match
  // Find the UTC instant whose wall-clock time in `timeZone` equals the input.
  // A first guess assuming UTC, corrected once using the timezone's actual
  // offset at that instant (handles all real-world offsets, including DST).
  const guess = Date.UTC(+year, +month - 1, +day, +hour, +minute)
  const guessed = parts(new Date(guess).toISOString(), timeZone)
  const guessedUtc = Date.UTC(+guessed.year, +guessed.month - 1, +guessed.day, +guessed.hour, +guessed.minute)
  return new Date(guess - (guessedUtc - guess)).toISOString()
}

/** Today's date as "YYYY-MM-DD" in timeZone, for a form default. */
export function todayInput(timeZone: string): string {
  return toDateInput(new Date().toISOString(), timeZone)
}
