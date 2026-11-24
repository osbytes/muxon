// Print the CHANGELOG.md section for one version, as the GitHub release body.
// Usage: node scripts/release-notes.mjs 1.2.3
//
// changesets/changelog-github appends "Thanks @login!" to every entry. Maintainers' credits are
// dropped (they wrote most of it); outside contributors' stay.
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const MAINTAINERS = new Set(['adavila0703'])

const version = process.argv[2]
if (!version) {
  console.error('usage: node scripts/release-notes.mjs <version>')
  process.exit(1)
}

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const lines = readFileSync(join(root, 'CHANGELOG.md'), 'utf8').split(/\r?\n/)
const start = lines.findIndex((l) => l.trim() === `## ${version}`)
if (start === -1) {
  console.error(`no "## ${version}" section in CHANGELOG.md`)
  process.exit(1)
}
let end = lines.findIndex((l, i) => i > start && /^## /.test(l))
if (end === -1) end = lines.length

const body = lines
  .slice(start + 1, end)
  .map((l) =>
    l.replace(/ Thanks (\[@([\w-]+)\]\([^)]*\)(, )?)+!/g, (credit) => {
      const logins = [...credit.matchAll(/\[@([\w-]+)\]\(([^)]*)\)/g)]
      const kept = logins.filter(([, login]) => !MAINTAINERS.has(login))
      return kept.length ? ` Thanks ${kept.map(([m]) => m).join(', ')}!` : ''
    })
  )
  .join('\n')
  .trim()

console.log(body)
