import { afterEach,describe,expect,it,vi } from 'vitest'
import { ApiClient,ApiError } from './client'

afterEach(()=>vi.unstubAllGlobals())
describe('ApiClient',()=>{
  it('attaches the PocketBase bearer token',async()=>{const fetchMock=vi.fn().mockResolvedValue(new Response(JSON.stringify({data:{ok:true}}),{status:200,headers:{'Content-Type':'application/json'}}));vi.stubGlobal('fetch',fetchMock);const client=new ApiClient(()=> 'Bearer token',()=>{});await client.get('/groups');expect((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).toBeInstanceOf(Headers);expect(((fetchMock.mock.calls[0]?.[1] as RequestInit).headers as Headers).get('Authorization')).toBe('Bearer token')})
  it('clears auth on 401',async()=>{const unauthorized=vi.fn();vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify({error:{code:'unauthorized',message:'?餃憭望?'}}),{status:401})));const client=new ApiClient(()=>'',unauthorized);await expect(client.get('/groups')).rejects.toBeInstanceOf(ApiError);expect(unauthorized).toHaveBeenCalledOnce()})
  it('preserves the screen with a network error',async()=>{vi.stubGlobal('fetch',vi.fn().mockRejectedValue(new Error('offline')));const client=new ApiClient(()=>'',()=>{});await expect(client.get('/groups')).rejects.toMatchObject({status:0,code:'network_error'})})
})

