import JSZip from 'jszip'

export type OfficePreview =
  | { kind: 'document'; paragraphs: string[] }
  | { kind: 'spreadsheet'; sheets: Array<{ name: string; rows: string[][] }>; truncated: boolean }
  | { kind: 'slides'; slides: Array<{ title: string; paragraphs: string[] }> }

const officeExtensions = new Set(['.docx', '.xlsx', '.pptx'])
const imageExtensions = new Set(['.png', '.jpg', '.jpeg', '.webp'])
const MAX_EXPANDED_SIZE = 40 * 1024 * 1024
const MAX_XML_SIZE = 10 * 1024 * 1024

export function canPreviewAttachment(fileName: string, mediaType: string): boolean {
  const extension = extensionOf(fileName)
  return mediaType === 'application/pdf' || imageExtensions.has(extension) || officeExtensions.has(extension)
}

export async function parseOfficePreview(file: Blob, fileName: string): Promise<OfficePreview> {
  const zip = await JSZip.loadAsync(file, { checkCRC32: false })
  const entries = Object.values(zip.files).filter((entry) => !entry.dir)
  const expandedSize = entries.reduce((total, entry) => total + ((entry as JSZip.JSZipObject & { _data?: { uncompressedSize?: number } })._data?.uncompressedSize ?? 0), 0)
  if (expandedSize > MAX_EXPANDED_SIZE) throw new Error('preview_too_large')

  const extension = extensionOf(fileName)
  if (extension === '.docx') return { kind: 'document', paragraphs: await readWord(zip) }
  if (extension === '.xlsx') return readWorkbook(zip)
  if (extension === '.pptx') return { kind: 'slides', slides: await readSlides(zip) }
  throw new Error('preview_unsupported')
}

function extensionOf(name: string): string {
  const index = name.lastIndexOf('.')
  return index < 0 ? '' : name.slice(index).toLowerCase()
}

async function xml(zip: JSZip, path: string): Promise<Document> {
  const entry = zip.file(path)
  if (!entry || ((entry as JSZip.JSZipObject & { _data?: { uncompressedSize?: number } })._data?.uncompressedSize ?? 0) > MAX_XML_SIZE) {
    throw new Error('preview_invalid')
  }
  const text = await entry.async('text')
  const parsed = new DOMParser().parseFromString(text, 'application/xml')
  if (parsed.querySelector('parsererror')) throw new Error('preview_invalid')
  return parsed
}

async function readWord(zip: JSZip): Promise<string[]> {
  const doc = await xml(zip, 'word/document.xml')
  return Array.from(doc.getElementsByTagName('w:p'))
    .map((paragraph) => Array.from(paragraph.getElementsByTagName('w:t')).map((node) => node.textContent ?? '').join(''))
    .filter((text) => text.length > 0)
}

async function readWorkbook(zip: JSZip): Promise<OfficePreview & { kind: 'spreadsheet' }> {
  const workbook = await xml(zip, 'xl/workbook.xml')
  const relations = await xml(zip, 'xl/_rels/workbook.xml.rels')
  const targets = new Map(Array.from(relations.getElementsByTagName('Relationship')).map((relation) => [
    relation.getAttribute('Id') ?? '', relation.getAttribute('Target') ?? '',
  ]))
  const sharedStrings = zip.file('xl/sharedStrings.xml') ? await xml(zip, 'xl/sharedStrings.xml') : undefined
  const strings = sharedStrings ? Array.from(sharedStrings.getElementsByTagName('si')).map((item) =>
    Array.from(item.getElementsByTagName('t')).map((node) => node.textContent ?? '').join('')) : []
  const sheets: Array<{ name: string; rows: string[][] }> = []
  let truncated = false

  for (const sheet of Array.from(workbook.getElementsByTagName('sheet'))) {
    const relationId = sheet.getAttribute('r:id') ?? ''
    const target = targets.get(relationId)
    if (!target) continue
    const path = target.startsWith('/') ? target.slice(1) : `xl/${target.replace(/^\.\//, '')}`
    const sheetXml = await xml(zip, path)
    const rows: string[][] = []
    for (const row of Array.from(sheetXml.getElementsByTagName('row'))) {
      if (rows.length >= 100) { truncated = true; break }
      const cells: string[] = []
      for (const cell of Array.from(row.getElementsByTagName('c'))) {
        const column = columnIndex(cell.getAttribute('r') ?? '')
        if (column >= 30) { truncated = true; continue }
        const type = cell.getAttribute('t')
        const value = cell.getElementsByTagName('v')[0]?.textContent ?? ''
        const inline = Array.from(cell.getElementsByTagName('t')).map((node) => node.textContent ?? '').join('')
        cells[column] = type === 's' ? (strings[Number(value)] ?? '') : type === 'inlineStr' ? inline : value
      }
      rows.push(cells.map((cell) => cell ?? ''))
    }
    sheets.push({ name: sheet.getAttribute('name') ?? '', rows })
  }
  return { kind: 'spreadsheet', sheets, truncated }
}

function columnIndex(reference: string): number {
  const letters = reference.match(/^[A-Z]+/i)?.[0]?.toUpperCase() ?? 'A'
  return [...letters].reduce((value, letter) => value * 26 + letter.charCodeAt(0) - 64, 0) - 1
}

async function readSlides(zip: JSZip): Promise<Array<{ title: string; paragraphs: string[] }>> {
  const paths = Object.keys(zip.files).filter((path) => /^ppt\/slides\/slide\d+\.xml$/.test(path))
    .sort((left, right) => Number(left.match(/slide(\d+)/)?.[1]) - Number(right.match(/slide(\d+)/)?.[1]))
  const slides = []
  for (const path of paths) {
    const doc = await xml(zip, path)
    const paragraphs = Array.from(doc.getElementsByTagName('a:p')).map((paragraph) =>
      Array.from(paragraph.getElementsByTagName('a:t')).map((node) => node.textContent ?? '').join(''))
      .filter((text) => text.trim().length > 0)
    slides.push({ title: paragraphs[0] ?? '', paragraphs })
  }
  return slides
}
