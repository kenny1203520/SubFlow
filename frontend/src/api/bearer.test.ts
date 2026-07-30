import { describe,expect,it } from 'vitest'
import { bearer } from './client'

describe('bearer',()=>{it('normalizes a raw PocketBase token without double-prefixing',()=>{expect(bearer('jwt')).toBe('Bearer jwt');expect(bearer('Bearer jwt')).toBe('Bearer jwt')})})
