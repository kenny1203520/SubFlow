import type { ApiFailure, Envelope } from './types'

import { isMaintenanceBlocked, signalClientUpdateRequired, signalMaintenance } from '../runtime/maintenance'

export class ApiError extends Error {
  status:number;code:string;fields?:Record<string,string>;details?:string;requestId?:string
  constructor(status:number,code:string,message:string,fields?:Record<string,string>,details?:string,requestId?:string){super(message);this.status=status;this.code=code;this.fields=fields;this.details=details;this.requestId=requestId}
}
export function bearer(token:string){return token&& !token.toLowerCase().startsWith('bearer ')?`Bearer ${token}`:token}
function maintenanceError(){return new ApiError(503,'maintenance','SubFlow is in maintenance mode')}
function transportError(){
  if(isMaintenanceBlocked())return maintenanceError()
  return new ApiError(0,'network_error','Network request failed')
}
export class ApiClient {
  private token:()=>string;private unauthorized:()=>void
  constructor(token:()=>string,unauthorized:()=>void){this.token=token;this.unauthorized=unauthorized}
  async request<T>(path:string,options:RequestInit={}):Promise<Envelope<T>>{
    if(isMaintenanceBlocked())throw maintenanceError()
    const headers=new Headers(options.headers);headers.set('Accept','application/json');headers.set('X-SubFlow-Protocol','1');const token=this.token();if(token)headers.set('Authorization',bearer(token));if(options.body&&!headers.has('Content-Type'))headers.set('Content-Type','application/json')
    let response:Response
    try{response=await fetch(`/api/subflow/v1${path}`,{...options,headers})}catch{throw transportError()}
    if(response.status===503){signalMaintenance();throw maintenanceError()}
    if(response.status===426)signalClientUpdateRequired()
    if(response.status===401)this.unauthorized()
    const body=await response.json().catch(()=>({error:{code:'invalid_response',message:'Invalid server response'}})) as Envelope<T>|ApiFailure
    if(!response.ok){const failure=body as ApiFailure;if(failure.error?.code==='client_update_required')signalClientUpdateRequired();throw new ApiError(response.status,failure.error?.code??'request_failed',failure.error?.message??'Request failed',failure.error?.fields,failure.error?.details,failure.error?.requestId)}
    return body as Envelope<T>
  }
  get<T>(path:string){return this.request<T>(path)}
  upload<T>(path:string, body:FormData, onProgress:(percent:number)=>void):Promise<Envelope<T>> {
    return new Promise((resolve,reject)=>{
      if(isMaintenanceBlocked()){reject(maintenanceError());return}
      const xhr=new XMLHttpRequest()
      xhr.open('POST',`/api/subflow/v1${path}`)
      const token=this.token();if(token)xhr.setRequestHeader('Authorization',bearer(token))
      xhr.setRequestHeader('Accept','application/json')
      xhr.setRequestHeader('X-SubFlow-Protocol','1')
      xhr.upload.onprogress=event=>{if(event.lengthComputable)onProgress(Math.round(event.loaded*100/event.total))}
      xhr.onerror=()=>reject(transportError())
      xhr.onabort=()=>reject(new ApiError(0,'upload_cancelled','Upload cancelled'))
      xhr.onload=()=>{
        if(xhr.status===503){signalMaintenance();reject(maintenanceError());return}
        if(xhr.status===426)signalClientUpdateRequired()
        if(xhr.status===401)this.unauthorized()
        let body:Envelope<T>|ApiFailure
        try{body=JSON.parse(xhr.responseText)}catch{reject(new ApiError(xhr.status,'invalid_response','Invalid server response'));return}
        if(xhr.status>=200&&xhr.status<300){resolve(body as Envelope<T>);return}
        const failure=body as ApiFailure
        if(failure.error?.code==='client_update_required')signalClientUpdateRequired()
        const fallbackCode=xhr.status===401?'unauthorized':xhr.status===403?'forbidden':'request_failed'
        reject(new ApiError(xhr.status,failure.error?.code??fallbackCode,failure.error?.message??'Request failed',failure.error?.fields,failure.error?.details,failure.error?.requestId))
      }
      xhr.send(body)
    })
  }
  async getBlob(path:string):Promise<{blob:Blob;filename:string}>{
    if(isMaintenanceBlocked())throw maintenanceError()
    const headers=new Headers();headers.set('Accept','*/*');headers.set('X-SubFlow-Protocol','1');const token=this.token();if(token)headers.set('Authorization',bearer(token))
    let response:Response
    try{response=await fetch(`/api/subflow/v1${path}`,{headers})}catch{throw transportError()}
    if(response.status===503){signalMaintenance();throw maintenanceError()}
    if(response.status===426)signalClientUpdateRequired()
    if(response.status===401)this.unauthorized()
    if(!response.ok){const body=await response.json().catch(()=>({error:{code:'invalid_response',message:'Invalid server response'}})) as ApiFailure;throw new ApiError(response.status,body.error?.code??'request_failed',body.error?.message??'Request failed',body.error?.fields,body.error?.details,body.error?.requestId)}
    const disposition=response.headers.get('Content-Disposition')||''
    const match=/filename="?([^"]+)"?/.exec(disposition)
    return {blob:await response.blob(),filename:match?.[1]||'export.csv'}
  }
  post<T>(path:string,body?:unknown){return this.request<T>(path,{method:'POST',body:body===undefined?undefined:JSON.stringify(body)})}
  patch<T>(path:string,body:unknown){return this.request<T>(path,{method:'PATCH',body:JSON.stringify(body)})}
  delete<T>(path:string){return this.request<T>(path,{method:'DELETE'})}
}
