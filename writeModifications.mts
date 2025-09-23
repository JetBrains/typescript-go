// MODIFIED IN THIS FORK: 2025-09-23 Aleksei Berezkin cde8066b5b710554e803ad957200451cea7a3053 Script to write modification notice
import assert from 'node:assert'
import child_process from 'node:child_process'
import fs from 'fs'
import readline from 'node:readline'

const ourContributors = ['Piotr Tomiak', 'Andrey Vorobev', 'Aleksei Berezkin', 'Vladislav Minaev', 'Konstantin Ulitin']

const child = child_process.exec(
  'git log dfd337df3e64d6fb2bc18c46bf4de05bcbea5a64~1..HEAD --pretty=format:"%H|%an|%ad|%s" --date=short --name-status --find-renames',
  {
    encoding: 'utf-8',
    maxBuffer: Infinity,
  }
)

type Commit = {
  hash: string
  author: string
  date: string
  message: string
  addedOrModified: string[]
  deleted: string[]
  renamed: {from: string, to: string}[]
}

const commits: Commit[] = []

readline.createInterface({input: child.stdout!}).on('line', line => {
  if (!line.trim()) return

  if (line.includes('|')) {
    const [hash, author, date, message] = line.split('|')
    commits.push({hash, author, date, message, addedOrModified: [], deleted: [], renamed: []})
  } else {
    const currentCommit = commits.at(-1)
    assert(currentCommit)

    if (ourContributors.includes(currentCommit.author)) {
      const addedOrModified = /[MA]\s+(.+)/.exec(line)
      if (addedOrModified)
        currentCommit.addedOrModified.push(addedOrModified[1].trim())
    }

    const deleted = /D\s+(.+)/.exec(line)
    if (deleted)
      currentCommit.deleted.push(deleted[1].trim())

    const renamed = /R\d+\s+(?<from>[^\s]+)\s+(?<to>[^\s]+)/.exec(line)
    if (renamed) {
      const {from, to} = renamed.groups!
      currentCommit.renamed.push({from, to})
    }

  }
})

type FileModification = {
  latestName: string
  deleted: boolean
  modifications: Modification[]
}

type Modification = {
  hash: string
  author: string
  date: string
  message: string
}

child.on('close', (code) => {
  if (code) console.log(`git log exited with code ${code}`)

  const allFileModifications = new Map<string, FileModification>()
  for (const commit of commits) {
    for (const addedOrModified of commit.addedOrModified) {
      const fileModifications = allFileModifications.get(addedOrModified)
      const {hash, author, date, message} = commit
      const modification = {hash, author, date, message}
      if (fileModifications) {
        fileModifications.modifications.push(modification)
      } else {
        allFileModifications.set(addedOrModified, {latestName: addedOrModified, deleted: false, modifications: [modification]})
      }
    }
    for (const deleted of commit.deleted) {
      const fileModifications = allFileModifications.get(deleted)
      if (fileModifications) {
        fileModifications.deleted = true
      } else {
        allFileModifications.set(deleted, {latestName: deleted, deleted: true, modifications: []})
      }
    }
    for (const {from, to} of commit.renamed) {
      const fileModifications = allFileModifications.get(to)
      if (fileModifications) {
        allFileModifications.delete(to)
        allFileModifications.set(from, fileModifications)
      } else {
        allFileModifications.set(from, {latestName: to, deleted: false, modifications: []})
      }
    }
  }

  const effectiveModifications = [...allFileModifications.entries()]
    .filter(([latestName]) => !latestName.includes('.idea/'))
    .filter(([, modifications]) => modifications.modifications.length && !modifications.deleted)
    .map(([, {latestName, modifications}]) => ({latestName, modifications}))


  const fileNames = effectiveModifications.map(({latestName}) => latestName)
  console.log(fileNames.sort())
  console.log(`Type 'y' to write to ${fileNames.length} files`)

  process.stdin.once('data', data => {
    process.stdin.pause()

    const answer = String(data).trim()
    if (answer.toLowerCase() === 'y') {
      for (const {latestName, modifications} of effectiveModifications) {
        patch(latestName, modifications)
      }
      console.log('Done')
    } else {
      console.log('Skipped')
    }
  })
})

const label = 'MODIFIED IN THIS FORK: '
const labelJson = 'modified-in-this-fork'
function patch(fileName: string, modifications: Modification[]) {
  const prefix = fileName.endsWith('.go') || fileName.endsWith('.mts') ? `// ${label}`
    : fileName.endsWith('.md') ? `\`${label}`
    : fileName === '.gitignore' ? `# ${label}`
    : fileName.endsWith('.json') ? `    "#${labelJson}": "`
    : undefined
  if (!prefix) throw new Error(`Unsupported file type: ${fileName}`)
  const suffixImpl = fileName.endsWith('.md') ? '`'
    : fileName.endsWith('.json') ? '",'
    : ''
  
  const content = fs.readFileSync(fileName, 'utf-8')
    .split('\n')
    .filter(line => !line.startsWith(prefix))

  const modificationsNotice = modifications
    .map(({hash, author, date, message}) => `${prefix}${date} ${author} ${hash} ${message}${suffixImpl}`)

  const newContent = fileName.endsWith('.json')
    ? [
      content[0],
      ...modificationsNotice,
      ...content.slice(1),
    ]
    : [
      ...modificationsNotice,
      ...content,
    ]

  fs.writeFileSync(fileName, newContent.join('\n'))
}
