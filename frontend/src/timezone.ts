export function timezoneOffset(timeZone:string,date:Date|string=new Date()){
  try{
    const value=new Intl.DateTimeFormat('en-US',{timeZone,timeZoneName:'longOffset'}).formatToParts(new Date(date)).find(part=>part.type==='timeZoneName')?.value||'GMT'
    if(value==='GMT'||value==='UTC')return{label:'UTC+00:00',minutes:0}
    const match=value.match(/(?:GMT|UTC)([+-])(\d{1,2})(?::?(\d{2}))?/)
    if(!match)return{label:'UTC+00:00',minutes:0}
    const hours=String(Number(match[2])).padStart(2,'0'),minutesPart=String(Number(match[3]||0)).padStart(2,'0')
    const minutes=(Number(hours)*60+Number(minutesPart))*(match[1]==='-'?-1:1)
    return{label:`UTC${match[1]}${hours}:${minutesPart}`,minutes}
  }catch{return{label:'UTC+00:00',minutes:0}}
}
export function timezoneLabel(timeZone:string,date:Date|string=new Date(),locale=typeof document==='undefined'?'zh-TW':document.documentElement.lang||'zh-TW'){const name=timeZone||'UTC';let localized=name;try{localized=new Intl.DateTimeFormat(locale,{timeZone:name,timeZoneName:'longGeneric'}).formatToParts(typeof date==='string'?new Date(date):date).find(part=>part.type==='timeZoneName')?.value||name}catch{}return`${localized} · ${name} (${timezoneOffset(name,date).label})`}
