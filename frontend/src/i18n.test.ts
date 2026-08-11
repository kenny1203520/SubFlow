import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { en } from './locales/en'
import { zhTW } from './locales/zh-TW'

function sourceFiles(directory:string):string[]{return readdirSync(directory,{withFileTypes:true}).flatMap(entry=>{const path=join(directory,entry.name);if(entry.isDirectory())return entry.name==='locales'?[]:sourceFiles(path);return /\.(vue|ts)$/.test(entry.name)&&!entry.name.endsWith('.test.ts')?[path]:[]})}

describe('i18n coverage',()=>{
  it('keeps locale dictionaries in sync',()=>{expect(Object.keys(en).sort()).toEqual(Object.keys(zhTW).sort())})
  it('does not leave Chinese interface copy outside locale files',()=>{const offenders=sourceFiles(join(process.cwd(),'src')).filter(path=>/[\u3400-\u9fff]/u.test(readFileSync(path,'utf8')));expect(offenders).toEqual([])})
})
