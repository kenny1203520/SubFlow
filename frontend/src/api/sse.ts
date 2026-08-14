import type { SubFlowEvent } from './types'
import { bearer } from './client'

export class SSEClient {
  private stopped=false
  private controller?:AbortController
  private token:()=>string
  private onEvent:(event:SubFlowEvent)=>void
  private onUnauthorized:()=>void
  private onReconnect?:()=>void
  // Only reconnects that follow an actual drop are interesting — the very
  // first connect on page load has nothing to resync yet.
  constructor(token:()=>string,onEvent:(event:SubFlowEvent)=>void,onUnauthorized:()=>void,onReconnect?:()=>void){this.token=token;this.onEvent=onEvent;this.onUnauthorized=onUnauthorized;this.onReconnect=onReconnect}
  stop(){this.stopped=true;this.controller?.abort()}
  async start(groupId=''){
    this.stop();this.stopped=false;let delay=1000;let hadFailure=false
    while(!this.stopped){
      this.controller=new AbortController()
      try{
        const query=groupId?`?groupId=${encodeURIComponent(groupId)}`:''
        const response=await fetch(`/api/subflow/v1/events${query}`,{headers:{Accept:'text/event-stream',Authorization:bearer(this.token())},signal:this.controller.signal})
        if(response.status===401){this.onUnauthorized();return}
        if(!response.ok||!response.body)throw new Error('SSE failed')
        delay=1000
        if(hadFailure){hadFailure=false;this.onReconnect?.()}
        const reader=response.body.getReader();const decoder=new TextDecoder();let buffer=''
        while(!this.stopped){const {done,value}=await reader.read();if(done)break;buffer+=decoder.decode(value,{stream:true});const frames=buffer.split('\n\n');buffer=frames.pop()??'';for(const frame of frames){const line=frame.split('\n').find(value=>value.startsWith('data: '));if(line)this.onEvent(JSON.parse(line.slice(6)) as SubFlowEvent)}}
      }catch(error){if(this.stopped||error instanceof DOMException&&error.name==='AbortError')return}
      hadFailure=true
      await new Promise(resolve=>setTimeout(resolve,delay));delay=Math.min(delay*2,30000)
    }
  }
}

