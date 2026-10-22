// @vitest-environment jsdom
import JSZip from 'jszip'
import { describe, expect, it } from 'vitest'
import { canPreviewAttachment, parseOfficePreview } from './attachmentPreview'

async function archive(files: Record<string, string>): Promise<Blob> {
  const zip = new JSZip()
  Object.entries(files).forEach(([path, content]) => zip.file(path, content))
  const bytes = await zip.generateAsync({ type: 'uint8array' })
  return new Blob([bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer])
}

describe('attachment preview formats', () => {
  it('enables common image, PDF, and modern Office formats while leaving legacy Office files downloadable', () => {
    expect(canPreviewAttachment('receipt.PDF', 'application/pdf')).toBe(true)
    expect(canPreviewAttachment('photo.webp', 'image/webp')).toBe(true)
    expect(canPreviewAttachment('report.docx', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document')).toBe(true)
    expect(canPreviewAttachment('ledger.xlsx', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')).toBe(true)
    expect(canPreviewAttachment('deck.pptx', 'application/vnd.openxmlformats-officedocument.presentationml.presentation')).toBe(true)
    expect(canPreviewAttachment('old.doc', 'application/msword')).toBe(false)
    expect(canPreviewAttachment('old.xls', 'application/vnd.ms-excel')).toBe(false)
    expect(canPreviewAttachment('photo.heic', 'image/heic')).toBe(false)
  })

  it('extracts DOCX paragraphs as text', async () => {
    const file = await archive({
      'word/document.xml': '<w:document xmlns:w="word"><w:body><w:p><w:r><w:t>Hello &amp; welcome</w:t></w:r></w:p><w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p></w:body></w:document>',
    })
    await expect(parseOfficePreview(file, 'note.docx')).resolves.toEqual({ kind: 'document', paragraphs: ['Hello & welcome', 'Second paragraph'] })
  })

  it('reads XLSX sheets and cached cell values without generating HTML', async () => {
    const file = await archive({
      'xl/workbook.xml': '<workbook xmlns:r="relationships"><sheets><sheet name="Budget" r:id="rId1"/></sheets></workbook>',
      'xl/_rels/workbook.xml.rels': '<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>',
      'xl/sharedStrings.xml': '<sst><si><t>Month</t></si><si><t>Amount</t></si></sst>',
      'xl/worksheets/sheet1.xml': '<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row><row r="2"><c r="A2" t="inlineStr"><is><t>Jan</t></is></c><c r="B2"><v>125</v></c></row></sheetData></worksheet>',
    })
    await expect(parseOfficePreview(file, 'budget.xlsx')).resolves.toEqual({
      kind: 'spreadsheet', sheets: [{ name: 'Budget', rows: [['Month', 'Amount'], ['Jan', '125']] }], truncated: false,
    })
  })

  it('extracts text slide by slide from PPTX', async () => {
    const file = await archive({
      'ppt/slides/slide1.xml': '<p:sld xmlns:p="p" xmlns:a="a"><a:p><a:r><a:t>Title</a:t></a:r></a:p><a:p><a:r><a:t>Details</a:t></a:r></a:p></p:sld>',
    })
    await expect(parseOfficePreview(file, 'brief.pptx')).resolves.toEqual({ kind: 'slides', slides: [{ title: 'Title', paragraphs: ['Title', 'Details'] }] })
  })

  it('rejects invalid archives', async () => {
    await expect(parseOfficePreview(new Blob(['not a zip']), 'bad.docx')).rejects.toThrow()
  })
})
