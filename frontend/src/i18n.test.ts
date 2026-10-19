import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { en } from './locales/en'
import { zhTW } from './locales/zh-TW'
import { incomeAttachmentAudit } from './locales/incomeAttachments'

function sourceFiles(directory:string):string[]{return readdirSync(directory,{withFileTypes:true}).flatMap(entry=>{const path=join(directory,entry.name);if(entry.isDirectory())return entry.name==='locales'?[]:sourceFiles(path);return /\.(vue|ts)$/.test(entry.name)&&!entry.name.endsWith('.test.ts')?[path]:[]})}

describe('i18n coverage',()=>{
  it('keeps locale dictionaries in sync',()=>{expect(Object.keys(en).sort()).toEqual(Object.keys(zhTW).sort())})
  it('covers every backend group permission in the editor and both languages',()=>{
    const backend=readFileSync(join(process.cwd(),'../backend/internal/application/rbac.go'),'utf8').split('var groupRolePermissions =')[1]!.split('\n}')[0]!
    const editor=readFileSync(join(process.cwd(),'src/views/GroupRolesView.vue'),'utf8')
    for(const match of backend.matchAll(/"([\w.]+)"/g)){
      const permission=match[1]!
      expect(editor).toContain(`'${permission}'`)
      for(const dictionary of [en,zhTW])for(const prefix of ['permission_','permissionDesc_'])expect((dictionary as Record<string,string>)[prefix+permission.replaceAll('.','_')],permission).toBeTruthy()
    }
  })
  it('covers income and attachment audit events emitted by backend code',()=>{
    const dir=join(process.cwd(),'../backend/internal/application')
    const files=readdirSync(dir).filter(n=>!n.endsWith('_test.go')&&n.endsWith('.go'))
    const actions=new Set(files.flatMap(n=>[...readFileSync(join(dir,n),'utf8').matchAll(/"((?:income|income_occurrence|recurring_income|attachment)\.[a-z_]+)"/g)].map(m=>m[1]!)))
    for(const action of actions)for(const lang of ['en','zh-TW'] as const)expect(incomeAttachmentAudit[lang].action[action],`${lang} ${action}`).toBeTruthy()
    for(const section of ['action','resource','field','reason'] as const)expect(Object.keys(incomeAttachmentAudit.en[section]).sort()).toEqual(Object.keys(incomeAttachmentAudit['zh-TW'][section]).sort())
  })
  it('does not leave Chinese interface copy outside locale files',()=>{const offenders=sourceFiles(join(process.cwd(),'src')).filter(path=>/[\u3400-\u9fff]/u.test(readFileSync(path,'utf8')));expect(offenders).toEqual([])})
})
