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
    try{response=await fetch(`/api/subflow/v1${path}`,{...options,headers})}catch{throw new ApiError(0,'network_error','Network request failed')}
    if(response.status===401)this.unauthorized()
    const body=await response.json().catch(()=>({error:{code:'invalid_response',message:'Invalid server response'}})) as Envelope<T>|ApiFailure
    if(!response.ok){const failure=body as ApiFailure;throw new ApiError(response.status,failure.error?.code??'request_failed',failure.error?.message??'Request failed',failure.error?.fields)}
    return body as Envelope<T>
  }
  get<T>(path:string){return this.request<T>(path)}
  upload<T>(path:string, body:FormData, onProgress:(percent:number)=>void):Promise<Envelope<T>> {
    return new Promise((resolve,reject)=>{
      const xhr=new XMLHttpRequest()
      xhr.open('POST',`/api/subflow/v1${path}`)
      const token=this.token();if(token)xhr.setRequestHeader('Authorization',bearer(token))
      xhr.setRequestHeader('Accept','application/json')
      xhr.upload.onprogress=event=>{if(event.lengthComputable)onProgress(Math.round(event.loaded*100/event.total))}
      xhr.onerror=()=>reject(new ApiError(0,'network_error','Network request failed'))
      xhr.onabort=()=>reject(new ApiError(0,'upload_cancelled','Upload cancelled'))
      xhr.onload=()=>{
        if(xhr.status===401)this.unauthorized()
        let body:Envelope<T>|ApiFailure
        try{body=JSON.parse(xhr.responseText)}catch{reject(new ApiError(xhr.status,'invalid_response','Invalid server response'));return}
        if(xhr.status>=200&&xhr.status<300){resolve(body as Envelope<T>);return}
        const failure=body as ApiFailure
        const fallbackCode=xhr.status===401?'unauthorized':xhr.status===403?'forbidden':'request_failed'
        reject(new ApiError(xhr.status,failure.error?.code??fallbackCode,failure.error?.message??'Request failed'))
      }
      xhr.send(body)
    })
  }
  async getBlob(path:string):Promise<{blob:Blob;filename:string}>{
    const headers=new Headers();headers.set('Accept','*/*');const token=this.token();if(token)headers.set('Authorization',bearer(token))
    let response:Response
    try{response=await fetch(`/api/subflow/v1${path}`,{headers})}catch{throw new ApiError(0,'network_error','Network request failed')}
    if(response.status===401)this.unauthorized()
    if(!response.ok){const body=await response.json().catch(()=>({error:{code:'invalid_response',message:'Invalid server response'}})) as ApiFailure;throw new ApiError(response.status,body.error?.code??'request_failed',body.error?.message??'Request failed',body.error?.fields)}
    const disposition=response.headers.get('Content-Disposition')||''
    const match=/filename="?([^"]+)"?/.exec(disposition)
    return {blob:await response.blob(),filename:match?.[1]||'export.csv'}
  }
  post<T>(path:string,body?:unknown){return this.request<T>(path,{method:'POST',body:body===undefined?undefined:JSON.stringify(body)})}
  patch<T>(path:string,body:unknown){return this.request<T>(path,{method:'PATCH',body:JSON.stringify(body)})}
  delete<T>(path:string){return this.request<T>(path,{method:'DELETE'})}
}
