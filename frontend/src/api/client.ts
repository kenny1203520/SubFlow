import type { ApiFailure, Envelope } from './types'

export class ApiError extends Error {
  status:number;code:string;fields?:Record<string,string>
  constructor(status:number,code:string,message:string,fields?:Record<string,string>){super(message);this.status=status;this.code=code;this.fields=fields}
}
export function bearer(token:string){return token&& !token.toLowerCase().startsWith('bearer ')?`Bearer ${token}`:token}
export class ApiClient {
  private token:()=>string;private unauthorized:()=>void
  constructor(token:()=>string,unauthorized:()=>void){this.token=token;this.unauthorized=unauthorized}
  async request<T>(path:string,options:RequestInit={}):Promise<Envelope<T>>{
    const headers=new Headers(options.headers);headers.set('Accept','application/json');const token=this.token();if(token)headers.set('Authorization',bearer(token));if(options.body&&!headers.has('Content-Type'))headers.set('Content-Type','application/json')
    let response:Response
    try{response=await fetch(`/api/subflow/v1${path}`,{...options,headers})}catch{throw new ApiError(0,'network_error','網路連線失敗，請稍後重試')}
    if(response.status===401)this.unauthorized()
    const body=await response.json().catch(()=>({error:{code:'invalid_response',message:'伺服器回應格式錯誤'}})) as Envelope<T>|ApiFailure
    if(!response.ok){const failure=body as ApiFailure;throw new ApiError(response.status,failure.error?.code??'request_failed',failure.error?.message??'請求失敗',failure.error?.fields)}
    return body as Envelope<T>
  }
  get<T>(path:string){return this.request<T>(path)}
  post<T>(path:string,body?:unknown){return this.request<T>(path,{method:'POST',body:body===undefined?undefined:JSON.stringify(body)})}
  patch<T>(path:string,body:unknown){return this.request<T>(path,{method:'PATCH',body:JSON.stringify(body)})}
  delete<T>(path:string){return this.request<T>(path,{method:'DELETE'})}
}
