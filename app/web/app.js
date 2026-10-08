import {progressAt,controlsByID,compatible,tone,meterValue,meterBallistics,gainPosition,display,gridMove,numericValue,sceneRoom,relativeTime,timeValue,stale,connectionTone,connectionText} from "./model.mjs";
import {icon} from "./icons.mjs";

const $=s=>document.querySelector(s);
const root=$("#content");
let state={Controls:[],Plugins:{},Enabled:{}}, controls=new Map(), screen="audio", signature="", pending=null;
let localError="", dismissedNotice="";
const widgets=[], updaters=[];
const titles={audio:["AUDIO","Audio"],routing:["AUDIO","Routing"],soundboard:["LIBRARY","Soundboard"],lights:["LIGHTING","Lights"],appaudio:["MIXER","App audio"],meetings:["CALLS","Meetings"],media:["NOW PLAYING","Media"],deck:["CONTROL SURFACE","Stream Deck"],plugins:["SYSTEM","Plugins"],apps:["SYSTEM","Third-party apps"],diagnostics:["SYSTEM","Diagnostics"]};

// Utility groups reused across screens. They are plain Tailwind utility
// literals (no component classes), kept complete so the CSS build finds them.
const ui={
 nav:"flex items-center gap-[13px] border-transparent bg-transparent px-3 py-[11px] text-left whitespace-nowrap text-muted hover:bg-hover aria-[current=page]:border-[#214a5a] aria-[current=page]:bg-[#163341] aria-[current=page]:text-active max-[1120px]:gap-[9px] max-[1120px]:px-2",
 panel:"min-w-0 rounded-xl border border-seam bg-panel p-5 max-[1120px]:p-4",
 panelHead:"mb-[18px] flex items-center justify-between gap-2.5",
 h2:"text-base font-semibold",
 h3:"mb-2.5 text-[13px] font-bold",
 grid:"grid grid-cols-2 gap-[18px] max-[1120px]:gap-3.5 max-[850px]:grid-cols-1",
 actions:"flex flex-wrap items-center gap-2",
 toolbar:"mb-[18px] flex flex-wrap items-center gap-2",
 empty:"rounded-xl border border-dashed border-line px-6 py-11 text-center text-muted",
 note:"-mt-1 mb-[15px] text-xs text-attention",
 small:"px-[9px] py-[5px]",
 primary:"border-transparent bg-active font-semibold text-[#06232b] hover:bg-active",
 danger:"text-critical",
 toggle:"min-w-[65px] aria-pressed:border-active-edge aria-pressed:bg-active-bg aria-pressed:text-active data-[tone=critical]:aria-pressed:border-critical-edge data-[tone=critical]:aria-pressed:bg-critical-bg data-[tone=critical]:aria-pressed:text-critical",
 strip:"rounded-[9px] border border-seam bg-sidebar p-[18px] max-[850px]:p-3",
 stripTitle:"mb-3 text-xs font-bold text-muted",
 readout:"text-[27px] tracking-[-.5px] tabular-nums max-[850px]:text-[23px]",
 meterLabel:"mt-[5px] text-[10px] text-muted tabular-nums",
 // Gradient stops follow the dBFS scale: green below -12, amber to -3, red above.
 meterGradient:"absolute inset-0 bg-[linear-gradient(90deg,var(--color-meter-green)_0%,var(--color-meter-green)_70%,var(--color-meter-amber)_80%,var(--color-meter-amber)_90%,var(--color-meter-red)_96%)]",
 gainButtons:"mt-3.5 flex gap-[7px] *:flex-1",
 clipGrid:"grid grid-cols-[repeat(auto-fill,minmax(145px,1fr))] gap-3.5 min-[1500px]:grid-cols-[repeat(auto-fill,minmax(170px,1fr))]",
 sceneGrid:"mt-[18px] grid grid-cols-[repeat(auto-fill,minmax(118px,1fr))] gap-2.5",
 clip:"relative flex min-h-[168px] flex-col items-start rounded-xl bg-panel p-4 text-left data-[state=playing]:border-active data-[state=playing]:bg-[#12303a] data-[state=wait]:border-attention data-[state=error]:border-critical",
 scene:"relative flex min-h-[112px] flex-col items-start rounded-xl bg-panel p-3 text-left data-[state=playing]:border-active data-[state=playing]:bg-[#12303a] data-[state=wait]:border-attention data-[state=error]:border-critical",
 clipArt:"mb-3 grid h-[70px] w-full place-items-center text-active",
 sceneArt:"mb-1.5 grid h-10 w-full place-items-center text-active",
 badge:"inline-block rounded-full border border-line px-[9px] py-1 text-[11px] text-muted data-[tone=active]:border-[#23505c] data-[tone=active]:bg-[#11313b] data-[tone=active]:text-active data-[tone=attention]:border-attention-edge data-[tone=attention]:text-attention data-[tone=critical]:border-critical-edge data-[tone=critical]:text-critical",
};
const toneText={active:"text-active",attention:"text-attention",critical:"text-critical",muted:"text-muted"};
function tc(t){return toneText[t]||"";}

function el(tag,className="",text="") {
 const node=document.createElement(tag);
 if(className) node.className=className;
 if(text!==undefined && text!=="") node.textContent=text;
 return node;
}
function button(text,handler,className="") {
 const node=el("button",className,text); node.type="button"; node.onclick=handler; return node;
}
function panel(title,parent,className="") {
 const node=el("section",ui.panel+" "+className), head=el("div",ui.panelHead);
 head.dataset.part="panel-head";
 head.append(el("h2",ui.h2,title)); node.append(head); parent.append(node); return node;
}
function badge(text,t=""){const b=el("span",ui.badge,text);b.dataset.tone=t;return b;}
function withIcon(node,name){node.prepend(icon(name,"size-4 shrink-0"));node.classList.add("inline-flex","items-center","gap-1.5");return node;}
function empty(parent,text){parent.append(el("div",ui.empty,text));}
function c(id){return controls.get(id);}
function showError(error){localError=String(error); refreshNotice();}
function refreshNotice(){
 const message=localError || (state.Notice!==dismissedNotice ? state.Notice : "");
 const node=$("#notice"); node.hidden=!message; node.replaceChildren();
 if(message){node.append(button("Dismiss",()=>{localError="";dismissedNotice=state.Notice;refreshNotice();},ui.small+" float-right ml-3"),document.createTextNode(message));}
}
async function send(action) {
 if(pending) return;
 localError="";
 pending={id:action.Request?.ID,revision:action.Request?.Revision,notice:state.Notice,until:Date.now()+2500};
 update();
 try { await window.go.app.Desktop.Send(action); }
 catch(error){pending=null;showError(error);update();}
}
function request(control,operation,value="",delta=0,revision=control?.Revision) {
 if(!control || !control.Available || !(control.Operations||[]).includes(operation)) return;
 send({Request:{ID:control.ID,Revision:revision,Operation:operation,Value:value,Delta:delta}});
}
function press(id){request(c(id),"press");}
// iconButton is an icon-only button; label is its accessible name and tooltip.
function iconButton(name,label,handler){
 const b=button("",handler,"flex items-center justify-center");b.append(icon(name,"size-4 shrink-0"));b.setAttribute("aria-label",label);b.title=label;return b;
}
function command(id,label,parent,className="",iconName="") {
 if(!c(id)) return null;
 const b=button(label,()=>press(id),className);if(iconName)withIcon(b,iconName);parent.append(b);
 updaters.push(()=>{b.disabled=!!pending || !c(id)?.Available;});
 return b;
}
// control renders one semantic control. layout "row" puts the label beside the
// input, "stack" above it (narrow panels), and "bare" drops the row separators;
// hideLabel keeps the label for assistive technology only.
function control(id,parent,label,{layout="row",hideLabel=false}={}) {
 const initial=c(id);if(!initial)return;
 const rowClass=layout==="stack"?"block py-3 border-t border-[#25323e] first:border-t-0 first:pt-0"
  :layout==="bare"?"flex items-center justify-between gap-[18px]"
  :"flex items-center justify-between gap-[18px] border-t border-[#25323e] py-3 first:border-t-0 first:pt-0 last:pb-0 max-[850px]:gap-2.5";
 const row=el("div",rowClass), caption=el("div",layout==="stack"?"mb-[7px]":"min-w-0 flex-1"), title=el("label",hideLabel?"sr-only":layout==="stack"?"block":"",label||initial.Label), note=el("small","mt-1 block text-[11px] break-words");
 row.dataset.part="row";
 caption.append(title,note); row.append(caption);
 const wide=layout==="stack";
 let input;
 if(initial.Kind==="toggle") {
  input=button("",()=>request(c(id),"press"),ui.toggle);input.setAttribute("aria-label",label||initial.Label);
 } else if(initial.Kind==="selection") {
  input=el("select",wide?"w-full":hideLabel?"min-w-40":"max-w-[58%] min-w-[110px]"); input.id="field-"+id;title.htmlFor=input.id;
  for(const value of initial.Options||[]) {const option=el("option","",initial.OptionLabels?.[value]??(value||"None"));option.value=value;input.append(option);}
  input.onfocus=()=>{input.editRevision=c(id)?.Revision;};
  input.onchange=()=>{request(c(id),"set",input.value,0,input.editRevision);input.blur();};
 } else if(initial.Kind==="text") {
  input=el("input","min-w-0 flex-1");input.type="text";input.id="field-"+id;title.htmlFor=input.id;
  input.onfocus=()=>{input.editRevision=c(id)?.Revision;};
  input.onkeydown=e=>{if(e.key==="Enter"){request(c(id),"set",input.value,0,input.editRevision);input.blur();}if(e.key==="Escape"){input.value=c(id)?.Value||"";input.blur();}};
  input.title="Enter to apply";
 } else if(initial.Kind==="command") {
  input=button(label||initial.Label,()=>press(id));title.textContent=initial.Value||"";
 } else if(initial.Kind==="numeric") {
  input=el("div",ui.actions);
  const value=el("span","max-w-[65%] text-right break-words");
  input.append(iconButton("minus","Decrease "+(label||initial.Label),()=>request(c(id),"adjust","",-1)),value,iconButton("plus","Increase "+(label||initial.Label),()=>request(c(id),"adjust","",1)));
  updaters.push(()=>{value.textContent=display(c(id)||initial);for(const b of input.querySelectorAll("button"))b.disabled=!!pending||!c(id)?.Available;});
 } else {
  input=el("span");
 }
 if(initial.Kind==="text"){
  const editor=el("div",wide?"flex min-w-0 gap-[7px]":"flex min-w-0 max-w-[65%] gap-[7px]"),apply=button(id==="streamdeck.add"?"Add":"Apply",()=>{
   request(c(id),"set",input.value,0,input.editRevision??c(id)?.Revision);input.blur();
  },ui.small);
  editor.append(input,apply);row.append(editor);
  const enable=()=>{apply.disabled=!!pending||!c(id)?.Available||input.value===(c(id)?.Value||"");};
  input.addEventListener("input",enable);updaters.push(enable);
 }else row.append(input);
 parent.append(row);
 widgets.push({id,row,title,note,input,initial});
}
// meterBar is a level meter: a dimmed full gradient as the unlit track, the
// lit level clipped over it with a short transition, a held peak marker and a
// dBFS scale. update returns the reading, or null when unknown.
function meterBar(){
 const bar=el("div","relative mt-3 h-2.5 rounded-[3px] bg-[#283542]"),unlit=el("div",ui.meterGradient+" rounded-[3px] opacity-20"),signal=el("div",ui.meterGradient+" rounded-[3px] transition-[clip-path] duration-[120ms] ease-out [clip-path:inset(0_100%_0_0)]"),peakMark=el("div","absolute -top-0.5 -bottom-0.5 w-0.5 -translate-x-1/2 bg-ink data-[tone=critical]:bg-meter-red");
 bar.dataset.part="meter";peakMark.dataset.part="peak";bar.append(unlit,signal,peakMark);
 const scale=el("div","relative mt-1 h-3 text-[10px] text-muted tabular-nums");
 for(const db of [-60,-30,-12,-3,0]){
  const mark=el("span","absolute "+(db===-60?"":db===0?"-translate-x-full":"-translate-x-1/2"),String(db));
  mark.style.left=((db+60)/60*100)+"%";scale.append(mark);
 }
 let meterState=null;
 return {nodes:[bar,scale],update(control){
  const db=meterValue(control?.Meter);
  meterState=meterBallistics(meterState,db,Date.now());
  const level=meterState.level;
  signal.style.clipPath="inset(0 "+(level===null?100:-(level/60)*100)+"% 0 0)";
  peakMark.hidden=level===null||meterState.peak<=-60;
  if(level!==null)peakMark.style.left=((meterState.peak+60)/60*100)+"%";
  peakMark.dataset.tone=meterState.peak>=-3?"critical":"";
  return db;
 }};
}
function mixer(id,parent,title) {
 if(!c(id))return;
 const node=el("div",ui.strip), value=el("div",ui.readout),text=el("div",ui.meterLabel),actions=el("div",ui.gainButtons);
 const device=el("div",ui.meterLabel);
 // Position track: where the gain sits in its range, with the 0 dB mark.
 const track=el("div","relative mt-2 h-1 rounded-full bg-[#283542]"),fillTrack=el("div","h-full rounded-full bg-active transition-[width] duration-[120ms]"),zero=el("div","absolute -top-1 h-3 w-0.5 -translate-x-1/2 bg-ink");
 track.dataset.part="position";track.append(fillTrack,zero);
 const meter=meterBar(),[bar,scale]=meter.nodes;
 node.dataset.part="strip";
 node.append(el("h3",ui.stripTitle,title),device,value,track,bar,scale,text,actions);
 actions.append(iconButton("minus",title+" down",()=>request(c(id),"adjust","",-1)),button("0 dB",()=>press(id),ui.toggle),iconButton("plus",title+" up",()=>request(c(id),"adjust","",1)));
 // Only soundboard's press means reset; audio gain press retains its existing mute behavior.
 if(id!=="soundboard.volume"){actions.children[1].textContent="Mute";actions.children[1].title="Toggle mute";}
 parent.append(node);
 updaters.push(()=>{
  const control=c(id);device.textContent=id==="audio.gain-playback"?(c("audio.playback-device")?.Value||""):id==="audio.gain-mic"?(c("audio.mic-device")?.Value||""):"";device.hidden=!device.textContent;value.textContent=control ? display(control) : "Unavailable";
  const db=meter.update(control);
  text.textContent=db===null?"LEVEL N/A":db.toFixed(1)+" dBFS";
  const position=control?.Available?gainPosition(control.Value):null;
  track.hidden=!position;
  if(position){fillTrack.style.width=(position.position*100)+"%";zero.hidden=position.zero===null;zero.style.left=((position.zero||0)*100)+"%";}
  const muteID={"audio.gain-mic":"audio.mic-mute","audio.gain-playback":"audio.speaker-mute"}[id];
  if(muteID){
   const muted=c(muteID)?.Value==="On";
   actions.children[1].dataset.tone=muted?"critical":"";
   actions.children[1].setAttribute("aria-pressed",String(muted));
  }
  for(const b of actions.children)b.disabled=!!pending||!control?.Available;
 });
}
function buildAudio() {
 if(![...controls.keys()].some(id=>id.startsWith("audio."))){empty(root,"Audio is disabled. Enable it in Plugins.");return;}
 const grid=el("div",ui.grid);root.append(grid);
 const live=panel("Live controls",grid,"col-span-full"), top=el("div","mb-5 grid grid-cols-3 gap-5"), strips=el("div","grid grid-cols-2 gap-5 max-[850px]:gap-[9px]");
 control("audio.health",live,"Engine");
 live.append(top);control("audio.mic-stack",top,"Mic stack",{layout:"bare"});control("audio.mic-mute",top,"Mic mute",{layout:"bare"});control("audio.speaker-mute",top,"Playback mute",{layout:"bare"});
 live.append(strips);
 mixer("audio.gain-mic",strips,"MICROPHONE");mixer("audio.gain-playback",strips,"PLAYBACK");control("audio.interface",live,"Interface");
 const used=new Set(["audio.health","audio.mic-stack","audio.mic-mute","audio.speaker-mute","audio.gain-mic","audio.gain-playback","audio.interface","audio.playback-device"]);
 for(const [title,prefix] of [["Normal","audio.normal-"],["VR","audio.vr-profile-"]]){
  if(!c(prefix+"source"))continue;
  const card=panel(title,grid), note=el("p",ui.note);card.append(note);
  updaters.push(()=>{note.textContent=c(prefix+"source")?.Subdued?"VR override · Normal remains editable":"";note.hidden=!note.textContent;});
  for(const [key,label] of [["source","Microphone"],["mode","Processing"],["monitor","Monitor"],["output","Playback"]]){control(prefix+key,card,label);used.add(prefix+key);}
 }
 const recording=[...controls.values()].filter(v=>v.Group==="Recording"&&!v.SurfaceOnly);
 if(recording.length){const card=panel("Recording",grid);for(const item of recording){control(item.ID,card);used.add(item.ID);}if(c("audio.tape-play"))transport(card);}
 if(c("aec.mode")){const card=panel("Echo cancellation",grid);card.dataset.part="echo";control("aec.status",card,"Status");control("aec.mode",card,"Mode");control("aec.engine",card,"Engine");control("aec.strength",card,"Strength");control("aec.retry",card,"Retry");control("aec.timing",card,"Timing");}
 const other=[...controls.values()].filter(v=>v.ID.startsWith("audio.")&&!v.SurfaceOnly&&!used.has(v.ID)&&v.Group!=="Routing");
 if(other.length||c("audio.engine-restart")){
  const card=panel("Routing & recovery",grid);for(const item of other)control(item.ID,card);
  const engine=el("div",ui.actions+" mt-3");card.append(engine);command("audio.engine-restart","Restart audio engine",engine,"","refresh-cw");
 }
}
// priorityEdit sends one change to a device priority list; Snoofer validates,
// saves and applies it.
function priorityEdit(list,op,index=0,value="",field=""){request(c("audio.priority-edit"),"set",JSON.stringify({list,op,index,value,field}));}
function editable(){return !pending&&!!c("audio.priority-edit")?.Available;}
// matchBadge states what an entry matches under the planner's rules.
function matchBadge(entry,missing="No match"){
 const matches=entry.Matches||[];
 const b=entry.InUse?badge("In use","active"):matches.length>1?badge("Ambiguous · "+matches.length,"attention"):badge(matches.length?"Ready":missing);
 b.dataset.part="match";if(matches.length)b.title=matches.join("\n");return b;
}
// patternField is a pattern input that applies on Enter or Apply.
function patternField(value,label,apply){
 const wrap=el("div","flex min-w-[min(100%,16rem)] flex-1 gap-[7px]"),input=el("input","min-w-0 flex-1 font-mono text-xs"),b=button("Apply",()=>apply(input.value),ui.small);
 input.type="text";input.value=value;input.setAttribute("aria-label",label);input.spellcheck=false;
 input.onkeydown=e=>{if(e.key==="Enter")b.click();if(e.key==="Escape"){input.value=value;input.blur();}};
 const sync=()=>{b.disabled=!editable()||input.value===value;input.disabled=!editable();};
 input.addEventListener("input",sync);updaters.push(sync);wrap.append(input,b);return wrap;
}
function entryButtons(list,index,count,name){
 const box=el("div","flex shrink-0 gap-1"),buttons=[iconButton("chevron-up","Move "+name+" up",()=>priorityEdit(list,"move",index,"up")),iconButton("chevron-down","Move "+name+" down",()=>priorityEdit(list,"move",index,"down")),iconButton("trash-2","Remove "+name,()=>priorityEdit(list,"remove",index))];
 buttons[2].classList.add(ui.danger);box.append(...buttons);
 updaters.push(()=>{for(const b of buttons)b.disabled=!editable();buttons[0].disabled||=index===0;buttons[1].disabled||=index===count-1;buttons[2].disabled||=count===1;});
 return box;
}
function entryRow(parent,index){
 const row=el("div","flex flex-wrap items-center gap-2.5 border-t border-[#25323e] py-3 first:border-t-0 first:pt-0");row.dataset.part="priority";
 row.append(el("span","w-5 shrink-0 text-right text-muted tabular-nums",String(index+1)));parent.append(row);return row;
}
// suggestionRows offers connected devices no entry matches, with patterns
// Snoofer generated.
function suggestionRows(parent,list,suggestions,driverLabel=true){
 if(!suggestions?.length)return;
 const box=el("div","mt-4 border-t border-[#25323e] pt-3");box.dataset.part="suggestions";box.append(el("h3",ui.stripTitle,"Suggestions"));parent.append(box);
 for(const s of suggestions){
  const row=el("div","flex flex-wrap items-center gap-2 py-1.5"),name=el("span","min-w-0 flex-1 break-words",s.Name);
  row.append(name);if(driverLabel)row.append(badge(s.Driver.toUpperCase()));
  // Add saves the device by identity; patterns are the advanced forms.
  const add=(entry,label,title,primary)=>{const b=withIcon(button(label,()=>priorityEdit(list,"add",0,JSON.stringify(entry)),ui.small+(primary?" "+ui.primary:"")),"plus");b.title=title;b.setAttribute("aria-label",label+" "+s.Name);row.append(b);updaters.push(()=>{b.disabled=!editable();});};
  if(s.ID)add({driver:s.Driver,id:s.ID,name:s.Name},"Add","This device, even if Windows renames it",true);
  add({driver:s.Driver,pattern:s.Exact},"Exact pattern",s.Exact,!s.ID);if(s.Device)add({driver:s.Driver,pattern:s.Device},"Partial pattern",s.Device,false);
  box.append(row);
 }
}
// deviceName shows an entry chosen by identity: the device's current name, or
// its stored label while disconnected.
function deviceName(label,matches){
 const node=el("span","flex min-w-[min(100%,16rem)] flex-1 items-center gap-2 break-words");node.dataset.part="device-entry";
 node.append(icon("speaker","size-4 shrink-0 text-muted"),document.createTextNode(matches?.length===1?matches[0]:label));
 if(!matches?.length)node.append(badge("Disconnected"));
 return node;
}
function candidateCard(parent,title,list,view){
 const card=panel(title,parent);card.dataset.part="list-"+list;
 candidateRows(card,title,list,view.Lists?.[list]||[],view.Suggestions?.[list],list==="playback");
}
// candidateRows lists device entries in priority order, then suggestions.
function candidateRows(card,title,list,entries,suggestions,driverLabel){
 entries.forEach((entry,n)=>{
  const row=entryRow(card,n);
  row.append(entry.DeviceID?deviceName(entry.DeviceName,entry.Matches):patternField(entry.Pattern,title+" "+(n+1)+" pattern",value=>priorityEdit(list,"set",n,value,"pattern")));
  if(list==="playback")row.append(badge(entry.Driver.toUpperCase()));
  row.append(matchBadge(entry),entryButtons(list,n,entries.length,title+" "+(n+1)));
  // An identity row already names its device.
  const names=entry.Matches||[];if(names.length&&!(entry.DeviceID&&names.length===1))row.append(el("small","basis-full pl-7 text-[11px] break-words text-muted",names.join(" · ")));
 });
 if(!entries.length)card.append(el("p","text-muted","No entries."));
 suggestionRows(card,list,suggestions,driverLabel);
}
function micName(view,id){return id==="off"?"Off":(view.Mics||[]).find(m=>m.ID===id)?.Name||id;}
// textField is a plain text input that applies on Enter or Apply.
function textField(value,label,max,apply){
 const wrap=el("div","flex min-w-[min(100%,12rem)] flex-1 gap-[7px]"),input=el("input","min-w-0 flex-1"),b=button("Apply",()=>apply(input.value.trim()),ui.small);
 input.type="text";input.value=value;input.maxLength=max;input.setAttribute("aria-label",label);
 input.onkeydown=e=>{if(e.key==="Enter")b.click();if(e.key==="Escape"){input.value=value;input.blur();}};
 const sync=()=>{b.disabled=!editable()||!input.value.trim()||input.value.trim()===value;input.disabled=!editable();};
 input.addEventListener("input",sync);updaters.push(sync);wrap.append(input,b);return wrap;
}
// micsCard defines the microphones; the number is the input each one uses.
// Device microphones list their devices in priority order.
function micsCard(parent,view){
 const card=panel("Microphones",parent,"col-span-full"),mics=view.Mics||[];card.dataset.part="mics";
 mics.forEach((m,n)=>{
  const row=entryRow(card,n);row.dataset.part="mic";
  const state=m.InUse?badge("In use","active"):m.Silent?badge("Silent","attention"):badge(m.Option?"Ready":"Unavailable");state.dataset.part="mic-state";
  row.append(textField(m.Name,"Microphone "+(n+1)+" name",16,v=>priorityEdit("mics","set",n,v,"name")),micSource(m,n,view),readyToggle(m,n),state,entryButtons("mics",n,mics.length,m.Name));
  if(m.Device){const box=el("div","basis-full pl-7");candidateRows(box,m.Name,"mic-devices:"+m.ID,view.Lists?.["mic-devices:"+m.ID]||[],view.Suggestions?.["mic-devices:"],false);row.append(box);}
 });
 if(mics.length<5){
  // A new microphone uses interface channels, or a connected input device.
  const add=el("div","mt-4 flex flex-wrap items-center gap-2 border-t border-[#25323e] pt-3"),name=el("input","w-40"),kind=el("select","min-w-0 flex-1");add.dataset.part="mic-add";
  name.type="text";name.placeholder="Microphone";name.maxLength=16;name.setAttribute("aria-label","New microphone name");kind.setAttribute("aria-label","New microphone source");
  kind.append(Object.assign(el("option","","Interface channels"),{value:""}));
  const devices=view.Suggestions?.["mic-devices:"]||[];
  devices.forEach((s,n)=>{const o=el("option","",s.Name);o.value=String(n);kind.append(o);});
  const added=()=>{const s=devices[Number(kind.value)];return kind.value===""?{name:name.value.trim()}:{name:name.value.trim(),devices:[s.ID?{driver:s.Driver,id:s.ID,name:s.Name}:{driver:s.Driver,pattern:s.Exact}]};};
  const b=withIcon(button("Add microphone",()=>{if(name.value.trim())priorityEdit("mics","add",0,JSON.stringify(added()));},ui.small),"plus");
  name.onkeydown=e=>{if(e.key==="Enter")b.click();};name.addEventListener("input",update);
  updaters.push(()=>{name.disabled=kind.disabled=!editable();b.disabled=!editable()||!name.value.trim();});
  add.append(name,kind,b);card.append(add);
 }
}
// micSource switches a microphone between interface channels and a
// connected input device; a device microphone keeps its current devices.
function micSource(m,n,view){
 const select=el("select","min-w-0 max-w-[16rem]"),devices=view.Suggestions?.["mic-devices:"]||[];select.setAttribute("aria-label",m.Name+" source");
 select.append(Object.assign(el("option","","Interface channels"),{value:"interface"}));
 if(m.Device)select.append(Object.assign(el("option","","Device"),{value:"current"}));
 devices.forEach((s,i)=>{const o=el("option","",s.Name);o.value=String(i);select.append(o);});
 select.value=m.Device?"current":"interface";
 select.onchange=()=>{const s=devices[Number(select.value)];priorityEdit("mics","set",n,select.value==="interface"?"interface":JSON.stringify(s.ID?{driver:s.Driver,id:s.ID,name:s.Name}:{driver:s.Driver,pattern:s.Exact}),"source");};
 updaters.push(()=>{select.disabled=!editable();});
 return select;
}
// readyToggle asserts a microphone is usable whenever present: Auto never
// skips it for silence.
function readyToggle(m,n){
 const b=button("Always ready",()=>priorityEdit("mics","set",n,String(!m.Ready),"ready"),ui.toggle+" "+ui.small);b.setAttribute("aria-pressed",String(!!m.Ready));b.setAttribute("aria-label",m.Name+" always ready");b.title="Auto never skips it for silence";
 updaters.push(()=>{b.disabled=!editable();});
 return b;
}
// channelFields sets one interface microphone's channels: Off, Mono (one
// channel) or Stereo (left and right).
function channelFields(entry,n,m){
 const field="input:"+m.ID,current=entry.Inputs?.[m.ID]||[],wrap=el("div","flex flex-wrap items-center gap-1.5");wrap.dataset.part="channels";
 const mode=el("select","w-24");mode.setAttribute("aria-label",m.Name+" on interface "+(n+1));
 for(const [value,label] of [["off","Off"],["mono","Mono"],["stereo","Stereo"]])mode.append(Object.assign(el("option","",label),{value}));
 mode.value=current.length===2?"stereo":current.length?"mono":"off";
 const number=(value,label)=>{const i=el("input","w-16 tabular-nums");i.type="number";i.min="1";i.max="64";i.value=String(value);i.setAttribute("aria-label",label);return i;};
 const left=current[0]||1,name=m.Name,where=" for interface "+(n+1);
 const inputs=mode.value==="stereo"?[number(left,name+" left channel"+where),number(current[1],name+" right channel"+where)]:mode.value==="mono"?[number(left,name+" channel"+where)]:[];
 const send=value=>priorityEdit("interfaces","set",n,value,field);
 mode.onchange=()=>send(mode.value==="off"?"":mode.value==="mono"?String(left):left+","+(current[1]||left+1));
 for(const i of inputs)i.onchange=()=>send(inputs.map(x=>x.value).join(","));
 updaters.push(()=>{for(const x of [mode,...inputs])x.disabled=!editable();});
 wrap.append(el("span","text-xs text-muted",name),mode,...inputs);return wrap;
}
// activityCard edits metering: how many leading options are metered (Off
// removes metering), the silence threshold and the delay before Auto skips.
function activityCard(parent,view){
 if(!view.Profiles)return;
 const card=panel("Activity",parent),a=view.Activity,fields=el("div","grid grid-cols-[max-content_minmax(0,1fr)] items-center gap-x-3 gap-y-2");card.dataset.part="activity";
 const check=el("select","min-w-0");check.setAttribute("aria-label","Meter");
 check.append(Object.assign(el("option","","Off"),{value:"0"}));
 for(let k=1;k<=4;k++)check.append(Object.assign(el("option","","Top "+k),{value:String(k)}));
 check.value=String(a?.Check||0);check.onchange=()=>priorityEdit("activity","set",0,check.value,"check");
 const number=(value,label,field,min,max)=>{const i=el("input","w-24 tabular-nums");i.type="number";i.min=String(min);i.max=String(max);i.value=a?String(value):"";i.setAttribute("aria-label",label);i.onchange=()=>priorityEdit("activity","set",0,i.value,field);return i;};
 const threshold=number(a?.SilenceDB,"Silence below (dBFS)","silence_db",-120,-20),delay=number(a?.SilentAfterS,"Silent after (s)","silent_after_s",2,600);
 fields.append(el("span","text-xs text-muted","Meter"),check,el("span","text-xs text-muted","Silence below (dBFS)"),threshold,el("span","text-xs text-muted","Silent after (s)"),delay);
 card.append(fields);
 updaters.push(()=>{check.disabled=!editable();threshold.disabled=delay.disabled=!editable()||!a;});
}
function interfaceCard(parent,view){
 const card=panel("Interfaces",parent,"col-span-full"),entries=view.Lists?.interfaces||[];card.dataset.part="list-interfaces";
 entries.forEach((entry,n)=>{
  const row=entryRow(card,n),fields=el("div","grid min-w-0 flex-1 grid-cols-[max-content_minmax(0,1fr)] items-center gap-x-3 gap-y-2");
  fields.append(el("span","text-xs text-muted","Driver"),entry.ASIOID?deviceName(entry.ASIOName,entry.Drivers):patternField(entry.ASIOPattern,"Interface "+(n+1)+" driver pattern",v=>priorityEdit("interfaces","set",n,v,"asio_pattern")));
  fields.append(el("span","text-xs text-muted","Presence"),entry.PresenceID?deviceName(entry.PresenceName,entry.Matches):patternField(entry.PresencePattern,"Interface "+(n+1)+" presence pattern",v=>priorityEdit("interfaces","set",n,v,"presence_pattern")));
  const channels=el("div","flex flex-wrap items-center gap-x-4 gap-y-2");
  for(const m of (view.Mics||[]).filter(m=>!m.Device))channels.append(channelFields(entry,n,m));
  fields.append(el("span","text-xs text-muted","Channels"),channels);
  row.append(fields,matchBadge(entry),entryButtons("interfaces",n,entries.length,"interface "+(n+1)));
  const drivers=entry.Drivers||[];row.append(el("small","basis-full pl-7 text-[11px] break-words text-muted",(entry.Matches||[]).join(" · ")+(drivers.length?" · driver "+drivers.join(", "):" · no driver")));
 });
 // A new interface pairs an installed driver with the WDM input that proves
 // its hardware is connected.
 if(view.Drivers?.length&&view.Inputs?.length){
  const add=el("div","mt-4 flex flex-wrap items-center gap-2 border-t border-[#25323e] pt-3"),driver=el("select","min-w-0 flex-1"),input=el("select","min-w-0 flex-1");add.dataset.part="interface-add";
  driver.setAttribute("aria-label","New interface driver");input.setAttribute("aria-label","New interface presence input");
  view.Drivers.forEach((s,n)=>{const o=el("option","",s.Name);o.value=String(n);driver.append(o);});
  view.Inputs.forEach((s,n)=>{const o=el("option","",s.Name);o.value=String(n);input.append(o);});
  // Both are saved by identity when Snoofer knows it, else as exact patterns.
  const added=()=>{const d=view.Drivers[Number(driver.value)],p=view.Inputs[Number(input.value)];return {...(d.ID?{asio_id:d.ID,asio_name:d.Name}:{asio_pattern:d.Exact}),...(p.ID?{presence_id:p.ID,presence_name:p.Name}:{presence_pattern:p.Exact}),inputs:{}};};
  const b=withIcon(button("Add",()=>priorityEdit("interfaces","add",0,JSON.stringify(added())),ui.small),"plus");
  updaters.push(()=>{b.disabled=driver.disabled=input.disabled=!editable();});
  add.append(el("h3",ui.stripTitle+" mb-0 basis-full","Add interface"),driver,input,b);card.append(add);
 }
}
function micPriorityCard(parent,view){
 const card=panel("Mic priority",parent),entries=view.Lists?.microphones||[];card.dataset.part="list-microphones";
 entries.forEach((entry,n)=>{
  const row=entryRow(card,n);row.append(el("span","min-w-0 flex-1",micName(view,entry.ID)));
  row.append(entry.InUse?badge("In use","active"):badge(entry.Option?"Ready":"Unavailable"),entryButtons("microphones",n,entries.length,micName(view,entry.ID)));
 });
 const missing=(view.Microphones||[]).filter(id=>!entries.some(e=>e.ID===id));
 if(missing.length){
  const add=el("div","mt-4 flex gap-2 border-t border-[#25323e] pt-3"),select=el("select","min-w-0 flex-1");select.setAttribute("aria-label","Microphone to add");
  for(const id of missing){const o=el("option","",micName(view,id));o.value=id;select.append(o);}
  const b=withIcon(button("Add",()=>priorityEdit("microphones","add",0,select.value),ui.small),"plus");
  updaters.push(()=>{b.disabled=select.disabled=!editable();});add.append(select,b);card.append(add);
 }
}
const sourceNames={"virtual:1":"Computer","virtual:2":"Virtual 2","virtual:3":"Virtual 3",monitor:"Monitor",soundboard:"Soundboard",tape:"Tape"};
const sourceIcons={"virtual:1":"monitor","virtual:2":"monitor","virtual:3":"monitor",monitor:"headphones",soundboard:"music",tape:"cassette-tape"};
const outputTones={"In use":"active","No output":"attention"};
// outputEdit adds, renames, re-assigns or removes an output slot by ID.
function outputEdit(op,id,value=""){request(c("audio.output-edit"),"set",JSON.stringify({op,id,value}));}
// routeCell is one source-to-destination switch; pressing it toggles (or
// cycles Monitor on Playback). Moving focus never dispatches.
function routeCell(id,label){
 const b=button("",()=>press(id),ui.toggle+" w-full min-w-0 px-2 py-1.5");b.dataset.part="route";b.setAttribute("aria-label",label);
 updaters.push(()=>{
  const item=c(id),on=!!item&&item.Value!=="Off"&&item.Value!=="off";
  b.textContent=item?display(item):"N/A";b.setAttribute("aria-pressed",String(on));b.disabled=!!pending||!item?.Available;
 });
 return b;
}
function outputsCard(parent,view){
 const card=panel("Outputs",parent,"col-span-full");card.dataset.part="outputs";
 const slots=[1,2,3].map(n=>"audio.slot-"+n).filter(id=>c(id)&&!c(id).Hidden);
 const editableOutputs=()=>!pending&&!!c("audio.output-edit")?.Available;
 const playbackSources=[...controls.keys()].filter(id=>id.startsWith("audio.playback:")).map(id=>id.slice("audio.playback:".length)).sort();
 const sources=[...playbackSources,"monitor"];
 for(const source of ["soundboard","tape"])if(slots.some(id=>c(id+":"+source)?.Available))sources.push(source);
 const scroll=el("div","overflow-x-auto"),table=el("table","w-full min-w-[520px] border-separate border-spacing-x-2 border-spacing-y-1.5 text-[13px]");scroll.append(table);card.append(scroll);
 const head=el("tr");table.append(el("thead"),el("tbody"));table.tHead.append(head);head.append(el("th","w-[150px]"));
 const column=(title,device,stateID)=>{
  const th=el("th","min-w-[150px] pb-2 text-left align-top font-normal"),name=el("div","flex items-center gap-1.5 font-semibold"),deviceText=el("div","mt-0.5 text-[11px] break-words text-muted"),state=badge("");
  name.append(icon("speaker","size-4 shrink-0 text-muted"),document.createTextNode(title));th.append(name,deviceText,state);head.append(th);
  updaters.push(()=>{deviceText.textContent=device()||"";const v=stateID?c(stateID)?.Value||"":"";state.textContent=v;state.dataset.tone=outputTones[v]||"";state.hidden=!v;});
  return th;
 };
 column("Playback",()=>c("audio.playback-device")?.Value,"").dataset.part="output-playback";
 for(const id of slots){
  const slot=c(id),slotID=slot.ViewData?.ID||"",th=column(slot.Label,()=>c(id)?.Status,id);th.dataset.part="output-slot";
  // Slot management: device, rename, remove.
  const tools=el("div","mt-2 flex flex-col gap-1.5"),device=el("select","w-full min-w-0 text-xs"),rename=el("input","min-w-0 flex-1 text-xs"),renameRow=el("div","flex gap-1.5");
  device.setAttribute("aria-label",slot.Label+" device");
  device.append(Object.assign(el("option","","No device"),{value:""}));
  // The current device keeps its saved form; others are saved by identity.
  if(slot.Status){const o=el("option","",slot.Status);o.value="current";device.append(o);}
  const choices=(view.OutputDevices||[]).filter(d=>d.Name!==slot.Status);
  choices.forEach((d,n)=>{const o=el("option","",d.Name);o.value=String(n);device.append(o);});
  device.value=slot.Status?"current":"";
  device.onchange=()=>{const d=choices[Number(device.value)];outputEdit("device",slotID,device.value===""?"":d.ID?JSON.stringify({ID:d.ID,Name:d.Name}):d.Name);};
  rename.type="text";rename.value=slot.Label;rename.maxLength=16;rename.setAttribute("aria-label","Rename "+slot.Label);
  const apply=button("Rename",()=>outputEdit("rename",slotID,rename.value.trim()),ui.small),remove=iconButton("trash-2","Remove "+slot.Label,()=>outputEdit("remove",slotID));remove.classList.add(ui.danger);
  rename.onkeydown=e=>{if(e.key==="Enter")apply.click();};
  renameRow.append(rename,apply,remove);tools.append(device,renameRow);th.append(tools);
  updaters.push(()=>{for(const n of [device,rename,remove])n.disabled=!editableOutputs();apply.disabled=!editableOutputs()||!rename.value.trim()||rename.value.trim()===slot.Label;});
 }
 for(const source of sources){
  const tr=el("tr"),th=el("th","text-left font-normal text-muted"),label=el("span","inline-flex items-center gap-2");
  label.append(icon(sourceIcons[source]||"audio-lines","size-4 shrink-0"),document.createTextNode(sourceNames[source]||source));th.append(label);tr.append(th);
  const playback=el("td");playback.dataset.part="cell";
  if(source==="monitor")playback.append(routeCell("audio.monitor","Playback Monitor"));
  else if(source==="tape")playback.append(el("span","text-xs text-muted","Always"));
  else if(c("audio.playback:"+source))playback.append(routeCell("audio.playback:"+source,"Playback "+(sourceNames[source]||source)));
  tr.append(playback);
  for(const id of slots){
   const td=el("td");td.dataset.part="cell";
   if(c(id+":"+source)?.Available)td.append(routeCell(id+":"+source,c(id).Label+" "+(sourceNames[source]||source)));
   tr.append(td);
  }
  table.tBodies[0].append(tr);
 }
 // A new output may start without a device; one can be assigned later from
 // connected devices that are neither Playback nor another slot.
 if(slots.length<3){
  const add=el("div","mt-4 flex flex-wrap items-center gap-2 border-t border-[#25323e] pt-3"),name=el("input","w-40"),device=el("select","min-w-0 flex-1");add.dataset.part="output-add";
  name.type="text";name.placeholder="Monitor output";name.maxLength=16;name.setAttribute("aria-label","New output name");device.setAttribute("aria-label","New output device");
  device.append(Object.assign(el("option","","No device"),{value:""}));
  const devices=view.OutputDevices||[];
  devices.forEach((d,n)=>{const o=el("option","",d.Name);o.value=String(n);device.append(o);});
  const added=()=>{const d=devices[Number(device.value)];return d?{name:name.value.trim(),device:d.Name,...(d.ID?{device_id:d.ID}:{})}:{name:name.value.trim()};};
  const b=withIcon(button("Add output",()=>{if(name.value.trim())outputEdit("add","",JSON.stringify(added()));},ui.small),"plus");
  name.onkeydown=e=>{if(e.key==="Enter")b.click();};
  updaters.push(()=>{name.disabled=device.disabled=!editableOutputs();b.disabled=!editableOutputs()||!name.value.trim();});
  name.addEventListener("input",update);add.append(name,device,b);card.append(add);
 }
}
function buildRouting(){
 const status=c("audio.priorities");
 if(!status){empty(root,"Audio is disabled. Enable it in Plugins.");return;}
 const view=status.ViewData||{},error=el("p","mb-[18px] text-critical");error.dataset.part="priority-error";root.append(error);
 updaters.push(()=>{error.textContent=c("audio.priority-edit")?.Status||"";error.hidden=!error.textContent;});
 const grid=el("div",ui.grid);root.append(grid);
 outputsCard(grid,view);micsCard(grid,view);interfaceCard(grid,view);candidateCard(grid,"Playback","playback",view);micPriorityCard(grid,view);activityCard(grid,view);
}
// transport is the Recording card's recorder row: Record, then playback of
// the loaded file. Labels follow the recorder state; availability comes from
// the controls, so nothing here can act on a recording.
function transport(card){
 const row=el("div",ui.actions+" mt-3");row.dataset.part="tape";card.append(row);
 const record=command("audio.record-toggle","Record",row,"","circle-dot"),play=command("audio.tape-play","Play",row,"","play");
 command("audio.tape-stop","Stop",row,"","square");
 for(const [id,name,label] of [["audio.tape-rew","rewind","Rewind"],["audio.tape-ff","fast-forward","Fast-forward"]]){
  if(!c(id))continue;
  const b=iconButton(name,label,()=>press(id));row.append(b);
  updaters.push(()=>{b.disabled=!!pending||!c(id)?.Available;});
 }
 const relabel=(b,text,name)=>{if(b&&b.dataset.label!==text){b.dataset.label=text;b.replaceChildren(icon(name,"size-4 shrink-0"),text);}};
 updaters.push(()=>{
  relabel(record,c("audio.record-toggle")?.Value==="Recording"?"Stop rec":"Record",c("audio.record-toggle")?.Value==="Recording"?"square":"circle-dot");
  relabel(play,c("audio.tape-play")?.Value==="Playing"?"Pause":"Play",c("audio.tape-play")?.Value==="Playing"?"pause":"play");
 });
}
function setArt(art,artwork,fallback,size,animated){
 if(art.dataset.art===(artwork||""))return;
 art.dataset.art=artwork||"";art.replaceChildren();
 if(artwork){const image=el("img",size+" object-contain");image.src="data:image/png;base64,"+artwork;image.alt="";art.append(image);if(animated)animate(art,image,animated,artwork);}
 else art.append(fallback());
}
// animate plays a control's artwork frames, fetched once per artwork; frames
// for older artwork are ignored, and playback stops when the image is replaced.
async function animate(art,image,id,artwork){
 const desktop=window.go?.app?.Desktop;if(!desktop?.Animation)return;
 let frames;try{frames=await desktop.Animation(id);}catch{return;}
 if(!frames||frames.length<2||frames[0].Artwork!==artwork)return;
 let n=0;
 const step=()=>{
  if(!image.isConnected||art.dataset.art!==artwork)return;
  n=(n+1)%frames.length;image.src="data:image/png;base64,"+frames[n].Artwork;
  setTimeout(step,frames[n].Delay/1e6);
 };
 setTimeout(step,frames[0].Delay/1e6);
}
function clipState(item,active){return item.Value===active?"playing":item.Value==="Wait"||item.Status==="Pending"?"wait":tone(item)==="critical"?"error":"";}
function buildSoundboard(){
 const clips=[...controls.values()].filter(v=>v.ID.startsWith("soundboard.clip-"));
 const toolbar=el("div",ui.toolbar),search=el("input","min-w-[170px] flex-1");search.type="search";search.placeholder="Search clips";search.setAttribute("aria-label","Search clips");toolbar.append(icon("search","size-5 shrink-0 text-muted"),search);
 command("soundboard.stop","Stop",toolbar,"","square");root.append(toolbar);
 if(c("soundboard.volume")){const card=panel("Playback",root,"mb-[18px]");control("soundboard.status",card,"Library");control("soundboard.overlap",card,"Overlap clips");mixer("soundboard.volume",card,"VOLUME");}
 if(!clips.length){empty(root,"No clips available. Check the soundboard plugin and library folder.");return;}
 const grid=el("div",ui.clipGrid);root.append(grid);
 for(const initial of clips){
  const b=button("",()=>press(initial.ID),ui.clip),art=el("div",ui.clipArt),name=el("strong","text-[13px] break-words",initial.Label),status=el("small","mt-[5px] text-[11px] text-muted");
  b.dataset.part="clip";b.append(art,name,status);b.title=initial.Label;grid.append(b);
  updaters.push(()=>{
   const item=c(initial.ID);if(!item)return;
   b.disabled=!!pending||!item.Available;b.dataset.state=clipState(item,"Playing");
   status.textContent=item.Status||display(item);status.className="mt-[5px] text-[11px] "+(tc(tone(item))||"text-muted");
   setArt(art,item.Artwork,()=>icon("play","size-9"),"size-16",item.ID);
   b.hidden=!item.Label.toLocaleLowerCase().includes(search.value.toLocaleLowerCase());
  });
 }
 search.oninput=update;
}
function lightsSetup(parent){
 const card=el("section",ui.panel+" mb-[18px] border-[#345365]"),title=el("h2",ui.h2),text=el("p","mt-2 mb-3.5 text-muted"),actions=el("div",ui.actions);
 card.dataset.part="setup";
 const pair=button("Pair",()=>press("hue.pair"),ui.primary);actions.append(pair);card.append(title,text,actions);parent.append(card);
 updaters.push(()=>{
  const status=c("hue.status"),p=c("hue.pair"),info=status?.ViewData||{},value=status?.Value||"";
  let heading="",body="",accent="",canPair=false;
  if(p?.Value==="Press button"){heading="Press the link button";body="Press the round button on top of the Hue Bridge"+(info.bridge?" at "+info.bridge:"")+" within 30 seconds.";accent="attention";}
  else if(p?.Status){heading="Pairing failed";body=p.Status;accent="critical";canPair=true;}
  else if(value==="Not paired"){heading=info.bridge?"Bridge found at "+info.bridge:"Bridge found";body="Pair once so Snoofer can control your lights.";canPair=true;}
  else if(value==="No bridge"){heading="Searching for the Hue Bridge";body="No bridge has answered yet. If it is on another network, set hue.address in snoofer.json to its IP address.";}
  else if(value==="Multiple bridges"){heading="Multiple bridges found";body=status.Status||"";}
  else if(value==="Error"){heading="Bridge needs pairing";body=status.Status||"";accent="critical";canPair=true;}
  else if(value==="Disconnected"){heading="Connecting to the Hue Bridge";body=status.Status||"";}
  card.hidden=!heading;title.textContent=heading;title.className=ui.h2+" "+tc(accent);text.textContent=body;
  pair.hidden=!canPair;pair.textContent=value==="Error"?"Pair again":"Pair";pair.disabled=!!pending||!p?.Available;
 });
}
function lightDial(id,parent,title,pressLabel,step,min,max){
 if(!c(id))return;
 const node=el("div",ui.strip),value=el("div",ui.readout),slider=el("input","mt-3 w-full border-0 bg-transparent p-0 accent-active"),note=el("div",ui.meterLabel),actions=el("div",ui.gainButtons);
 node.dataset.part="strip";
 slider.type="range";slider.min=min;slider.max=max;slider.step="any";slider.setAttribute("aria-label",title);
 node.append(el("h3",ui.stripTitle,title),value,slider,note,actions);parent.append(node);
 actions.append(iconButton("minus",title+" down",()=>request(c(id),"adjust","",-1)),button(pressLabel,()=>press(id)),iconButton("plus",title+" up",()=>request(c(id),"adjust","",1)));
 // The plugin owns absolute values; the slider sends the equivalent relative ticks on release.
 slider.onchange=()=>{const current=numericValue(c(id)?.Value);if(current===null)return;const ticks=Math.round((Number(slider.value)-current)/step);if(ticks)request(c(id),"adjust","",ticks);};
 updaters.push(()=>{
  const control=c(id),number=numericValue(control?.Value);
  value.textContent=control?display(control):"Unavailable";value.className=ui.readout+(control?.Subdued?" text-attention":"");
  if(document.activeElement!==slider&&number!==null)slider.value=number;
  slider.disabled=!!pending||!control?.Available||number===null;
  note.textContent=control?.Status||"";note.className=ui.meterLabel+" "+tc(tone(control));
  for(const b of actions.children)b.disabled=!!pending||!control?.Available;
 });
}
function segmented(id,title,parent){
 const initial=c(id);if(!initial)return;
 const row=el("div","mb-4"),group=el("div","flex");group.setAttribute("role","group");group.setAttribute("aria-label",title);
 row.append(el("h3",ui.stripTitle,title),group);parent.append(row);
 updaters.push(()=>{row.hidden=!!c(id)?.Hidden;});
 for(const option of initial.Options||[]){
  const b=button(initial.OptionLabels?.[option]??option,()=>request(c(id),"set",option),"-ml-px min-w-0 flex-1 rounded-none px-1 py-2 first:ml-0 first:rounded-l-[7px] last:rounded-r-[7px] aria-pressed:relative aria-pressed:border-active-edge aria-pressed:bg-active-bg aria-pressed:text-active");group.append(b);
  updaters.push(()=>{const item=c(id);b.setAttribute("aria-pressed",String(item?.Value===option));b.disabled=!!pending||!item?.Available;});
 }
}
function sceneCard(id,parent){
 const b=button("",()=>press(id),ui.scene),art=el("div",ui.sceneArt),name=el("strong","text-[13px] break-words"),status=el("small","mt-[5px] text-[11px] text-muted");
 b.dataset.part="scene";b.append(art,name,status);parent.append(b);
 updaters.push(()=>{
  const item=c(id);if(!item)return;
  name.textContent=item.ShortLabel||item.Label;b.title=item.Label;
  b.disabled=!!pending||!item.Available;b.dataset.state=clipState(item,"Active");
  status.textContent=item.Status==="Pending"?"Wait":item.Status||display(item);status.className="mt-[5px] text-[11px] "+(tc(tone(item))||"text-muted");
  setArt(art,item.Artwork,()=>icon("lightbulb","size-7"),"size-10");
 });
}
// roomsCard chooses the rooms and zones Snoofer controls, here and on the deck.
function roomsCard(parent){
 const item=c("hue.rooms");if(!item?.ViewData?.length)return;
 const card=panel("Rooms",parent,"col-span-full");card.dataset.part="rooms";
 card.append(el("p","text-[13px] text-muted","Only checked rooms appear here and on the deck."));
 const list=el("div","mt-3 grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-2"),error=el("p","mt-2 text-critical");card.append(list,error);
 const boxes=[];
 for(const room of item.ViewData){
  const label=el("label","flex items-center gap-2.5"),box=el("input","size-4 accent-active");box.type="checkbox";box.value=room.ID;
  label.append(box,el("span","",room.Name+(room.Kind==="zone"?" (zone)":"")));list.append(label);boxes.push(box);
  box.onchange=()=>request(c("hue.rooms"),"set",boxes.filter(b=>b.checked).map(b=>b.value).join(","));
 }
 updaters.push(()=>{
  const current=c("hue.rooms"),chosen=new Map((current?.ViewData||[]).map(r=>[r.ID,r.Chosen]));
  for(const b of boxes)b.checked=!!chosen.get(b.value);
  const count=boxes.filter(b=>b.checked).length;
  // The last chosen room stays chosen.
  for(const b of boxes)b.disabled=!!pending||!current?.Available||(b.checked&&count===1);
  error.textContent=current?.Status||"";error.hidden=!error.textContent;
 });
}
function buildLights(){
 if(!(state.Plugins||{}).hue){empty(root,"Hue is not part of this build.");return;}
 if(!state.Enabled?.hue){
  const card=panel("Hue is off",root,"mb-[18px] border-[#345365]");card.dataset.part="setup";card.append(el("p","mt-2 mb-3.5 text-muted","Control Hue scenes, room brightness and Hue Sync."));
  const actions=el("div",ui.actions);actions.append(button("Enable Hue",()=>send({Kind:"selection",Plugin:"hue",Enable:true}),ui.primary));card.append(actions);return;
 }
 if(!c("hue.status")){empty(root,"Hue: "+(state.Plugins.hue||"Starting"));return;}
 lightsSetup(root);
 const grid=el("div","grid grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)] items-start gap-[18px] max-[1120px]:grid-cols-1");grid.dataset.part="lights";root.append(grid);
 const room=panel("Room",grid),roomNote=el("p",ui.note);room.dataset.part="room";room.append(roomNote);
 control("hue.group",room.querySelector("[data-part=panel-head]"),"Room",{layout:"bare",hideLabel:true});
 const dials=el("div","mt-4 grid grid-cols-1 gap-5");room.append(dials);
 lightDial("hue.brightness",dials,"BRIGHTNESS","On/Off",2,1,100);
 // Motion sensors appear only while the room has some.
 const motion=el("div","mt-4");motion.dataset.part="motion";room.append(motion);control("hue.motion",motion,"Motion sensors");
 updaters.push(()=>{const item=c("hue.motion");motion.hidden=!item||item.Hidden;});
 updaters.push(()=>{roomNote.textContent=c("hue.sync")?.Value==="On"?"Hue Sync is driving the lights · Brightness adjusts the sync":"";roomNote.hidden=!roomNote.textContent;});
 const groupControl=c("hue.group"),roomName=(groupControl?.OptionLabels?.[groupControl.Value]||"").replace(/ \(zone\)$/,"");
 const scenes=[...controls.values()].filter(v=>v.ID.startsWith("hue.scene-"));
 const own=scenes.filter(v=>roomName&&sceneRoom(v)===roomName).sort((a,b)=>a.ShortLabel.localeCompare(b.ShortLabel));
 const sceneGrid=el("div",ui.sceneGrid);sceneGrid.dataset.part="room-scenes";room.append(sceneGrid);
 if(own.length)for(const scene of own)sceneCard(scene.ID,sceneGrid);
 else sceneGrid.append(el("p","text-muted",roomName?"No scenes in this room.":"Choose a room to see its scenes."));
 const others=scenes.filter(v=>!own.includes(v));
 if(others.length){
  const details=el("details","mt-4"),summary=el("summary","","Other rooms");details.append(summary);room.append(details);
  for(const name of [...new Set(others.map(sceneRoom))].sort()){
   const heading=el("h3","mt-4 text-xs font-bold text-muted",name||"Other");heading.dataset.part="room-heading";
   details.append(heading);const g=el("div",ui.sceneGrid);details.append(g);
   for(const scene of others.filter(v=>sceneRoom(v)===name).sort((a,b)=>a.ShortLabel.localeCompare(b.ShortLabel)))sceneCard(scene.ID,g);
  }
 }
 const sync=panel("Hue Sync",grid),syncState=el("p","-mt-1.5 mb-3 text-[13px]"),toggle=button("",()=>press("hue.sync"),ui.toggle+" mb-[18px] w-full p-3.5 text-base"),help=el("p","text-muted");
 sync.append(syncState,toggle);segmented("hue.sync-mode","Mode",sync);segmented("hue.sync-intensity","Intensity",sync);sync.append(help);
 updaters.push(()=>{
  const status=c("hue.sync-status"),item=c("hue.sync"),on=item?.Value==="On",failed=!!item?.Status&&item.Status!=="Pending";
  syncState.textContent=status?.Value==="N/A"?"Not connected":status?.Value||"";syncState.className="-mt-1.5 mb-3 text-[13px] "+(on?"text-active":tc(tone(status)));
  toggle.textContent=item?.Status==="Pending"?"Wait":on?"Stop sync":"Start sync";toggle.setAttribute("aria-pressed",String(on));
  toggle.dataset.tone=failed?"critical":"";toggle.disabled=!!pending||!item?.Available;
  help.textContent=status?.Value==="N/A"?"Open Hue Sync and turn on Settings → Third-party control.":status?.Value==="No bridge"?"Hue Sync has no bridge connection.":failed?item.Status:"";
  help.className=failed?"text-critical":"text-muted";help.hidden=!help.textContent;
 });
 roomsCard(grid);
}
// appEdit sends a pick or rule change for an app.
function appEdit(op,app,value=""){request(c("appaudio.edit"),"set",JSON.stringify({op,app,value}));}
function appStrip(id,parent){
 const initial=c(id),node=el("div",ui.strip+" flex flex-col"),head=el("div","mb-3 flex items-center gap-3"),art=el("div","grid size-10 shrink-0 place-items-center text-active"),name=el("h3","min-w-0 flex-1 truncate text-sm font-semibold",initial.Label);
 node.dataset.part="app";parent.append(node);
 const info=()=>(c("appaudio.status")?.ViewData?.Apps||[]).find(a=>a.ID===id)||{};
 const pin=iconButton("pin","Pin "+initial.Label,()=>appEdit(info().Picked?"unpick":"pick",initial.Label));pin.dataset.part="pin";pin.className+=" "+ui.toggle+" min-w-0 px-2";
 const earlier=iconButton("chevron-left","Move "+initial.Label+" earlier",()=>appEdit("move",initial.Label,"up")),later=iconButton("chevron-right","Move "+initial.Label+" later",()=>appEdit("move",initial.Label,"down"));
 head.append(art,name,earlier,later,pin);
 const value=el("div",ui.readout),status=el("div",ui.meterLabel);
 const slider=el("input","mt-3 w-full accent-active");slider.type="range";slider.min="0";slider.max="100";slider.step="1";slider.setAttribute("aria-label",initial.Label+" volume");slider.dataset.part="volume";
 slider.onchange=()=>request(c(id),"set",slider.value);
 const meter=meterBar(),mute=button("Mute",()=>press(id),ui.toggle+" mt-3 self-start");mute.dataset.part="mute";
 const more=el("details","mt-3 text-[13px]"),summary=el("summary","cursor-pointer text-muted","More");more.append(summary);
 const rename=el("input","min-w-0 flex-1");rename.type="text";rename.placeholder="New name";rename.setAttribute("aria-label","Rename "+initial.Label);
 const renameRow=el("div","mt-2 flex gap-2"),renameApply=button("Rename",()=>{if(rename.value.trim())appEdit("rename",initial.Label,rename.value.trim());},ui.small);renameRow.append(rename,renameApply);
 const combine=el("select","min-w-0 flex-1");combine.setAttribute("aria-label","Combine "+initial.Label+" into");
 const combineRow=el("div","mt-2 flex gap-2"),combineApply=button("Combine",()=>{if(combine.value)appEdit("combine",initial.Label,combine.value);},ui.small);combineRow.append(combine,combineApply);
 for(const other of (c("appaudio.status")?.ViewData?.Apps||[]).filter(a=>!a.Hidden&&a.ID!==id)){const option=el("option","","Into "+other.Name);option.value=other.Name;combine.append(option);}
 const hide=button("Hide",()=>appEdit("hide",initial.Label),ui.small+" "+ui.danger);hide.dataset.part="hide";
 const details=el("dl","mt-3 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-3 gap-y-1 text-[12px] text-muted");
 more.append(renameRow,combineRow,el("div",ui.actions+" mt-2"),details);more.children[3].append(hide);
 // Rules naming programs into this app (its rename, combined programs);
 // Separate undoes one.
 const naming=info().Naming||[],separates=[];
 if(naming.length){
  const list=el("div","mt-3 border-t border-[#25323e] pt-2");list.dataset.part="naming";list.append(el("h4","text-[11px] font-bold text-muted","Named here"));
  for(const rule of naming){
   const row=el("div","flex items-center justify-between gap-2 py-1"),programs=(rule.Programs||[]).join(", ");
   const b=button("Separate",()=>appEdit("separate",initial.Label,rule.Match),ui.small);b.setAttribute("aria-label","Separate "+programs);separates.push(b);
   row.append(el("span","min-w-0 break-all text-ink",programs),b);list.append(row);
  }
  more.insertBefore(list,details);
 }
 node.append(head,value,status,slider,...meter.nodes,mute,more);
 updaters.push(()=>{
  const item=c(id);if(!item)return;
  const a=info(),closed=item.Value==="Closed",percent=numericValue(item.Value);
  setArt(art,item.Artwork,()=>icon("app-window","size-7"),"size-10");
  value.textContent=display(item);status.textContent=item.Status||(closed?"Not running":"");status.className=ui.meterLabel+" "+tc(tone(item));
  if(document.activeElement!==slider&&percent!==null)slider.value=String(percent);
  meter.update(item);
  const muted=item.Value==="Muted";mute.textContent=muted?"Muted":"Mute";mute.setAttribute("aria-pressed",String(muted));mute.dataset.tone=muted?"critical":"";
  pin.setAttribute("aria-pressed",String(!!a.Picked));pin.title=a.Picked?"Unpin":"Pin";
  earlier.hidden=later.hidden=!a.Picked;
  for(const b of [slider,mute])b.disabled=!!pending||!item.Available||closed;
  for(const n of [slider,mute,...meter.nodes])n.hidden=closed;
  for(const b of [pin,earlier,later,renameApply,combineApply,hide,rename,combine,...separates])b.disabled=!!pending||!c("appaudio.edit")?.Available;
  for(const b of [renameApply,combineApply,hide,rename,combine])b.disabled||=closed;
  details.replaceChildren();
  for(const [label,text] of [["Programs",(a.Executables||[]).join("\n")],["Devices",(a.Devices||[]).join("\n")],["Processes",(a.PIDs||[]).join(", ")],["Sessions",String(a.Sessions||0)],["Rule",a.Rule||"None"],["Heard",a.LastHeard?relativeTime(a.LastHeard):"Not yet"]]){
   details.append(el("dt","",label),el("dd","whitespace-pre-wrap break-all text-ink",text||"—"));
  }
 });
}
function buildAppAudio(){
 if(!(state.Plugins||{}).appaudio){empty(root,"App audio is not part of this build.");return;}
 if(!state.Enabled?.appaudio){
  const card=panel("App audio is off",root,"mb-[18px] border-[#345365]");card.dataset.part="setup";card.append(el("p","mt-2 mb-3.5 text-muted","Control the volume and mute of individual apps."));
  const actions=el("div",ui.actions);actions.append(button("Enable App audio",()=>send({Kind:"selection",Plugin:"appaudio",Enable:true}),ui.primary));card.append(actions);return;
 }
 // The recent window is edited inline; it saves on Enter or when focus leaves.
 const intro=el("div","mb-2 flex flex-wrap items-center gap-2 text-[13px] text-muted"),minutes=el("input","w-20 tabular-nums");
 intro.dataset.part="appaudio-note";
 minutes.type="number";minutes.min="1";minutes.max="1440";minutes.step="1";minutes.setAttribute("aria-label","Recent window in minutes");minutes.dataset.part="recent";
 minutes.onchange=()=>{
  const value=Number(minutes.value),current=c("appaudio.status")?.ViewData?.RecentMinutes;
  if(Number.isInteger(value)&&value>=1&&value<=1440&&value!==current)appEdit("recent","recent",String(value));
  else minutes.value=String(current||5);
 };
 minutes.onkeydown=e=>{if(e.key==="Enter")minutes.blur();};
 intro.append(el("span","","Pinned apps first, then apps heard in the last"),minutes,el("span","","minutes."));
 const note=el("p","mb-[18px] text-[13px]");root.append(intro,note);
 updaters.push(()=>{
  const s=c("appaudio.status");
  if(document.activeElement!==minutes)minutes.value=String(s?.ViewData?.RecentMinutes||5);
  minutes.disabled=!!pending||!c("appaudio.edit")?.Available;
  note.textContent=s?.Status||(s?.Value==="Preview"?"Preview: volumes are read, never changed.":"");
  note.hidden=!note.textContent;
  note.className="mb-[18px] text-[13px] "+(s?.Status?"text-critical":"text-muted");
 });
 const apps=[...controls.values()].filter(v=>v.Collection==="appaudio.apps").sort((a,b)=>(a.Order||0)-(b.Order||0));
 if(!apps.length)empty(root,"No app has played sound recently. Play something, or pin an app to keep it here.");
 else{const grid=el("div","grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-[18px]");root.append(grid);for(const item of apps)appStrip(item.ID,grid);}
 excludedCard(c("appaudio.status")?.ViewData||{});
}
// excludedCard lists excluded program patterns with the running apps each
// hides, adds patterns by file name, and lists apps hidden by JSON rules.
function excludedCard(view){
 const card=panel("Excluded",root,"mt-[18px]");card.dataset.part="excluded";
 card.append(el("p","-mt-2 mb-3 text-[13px] text-muted","Programs never shown here or on the deck. Use the file name; * matches anything."));
 const apps=view.Apps||[],editable=()=>!pending&&!!c("appaudio.edit")?.Available,buttons=[];
 const row=(title,detail,action,label,aria)=>{
  const node=el("div","flex items-center justify-between gap-3 border-t border-[#25323e] py-2.5"),text=el("div","min-w-0");
  text.append(el("div","break-all",title));if(detail)text.append(el("small","block break-all text-[11px] text-muted",detail));
  const b=button(label,action,ui.small);b.setAttribute("aria-label",aria);buttons.push(b);node.append(text,b);card.append(node);
 };
 for(const pattern of view.Exclude||[]){
  const hiding=apps.filter(a=>a.Hidden&&a.Open&&a.Rule==="Excluded ("+pattern+")").map(a=>a.Name);
  row(pattern,hiding.length?"Hiding "+hiding.join(", "):"",()=>appEdit("include",pattern,pattern),"Remove","Remove "+pattern);
 }
 if(!(view.Exclude||[]).length)card.append(el("p","text-[13px] text-muted","Nothing is excluded."));
 for(const a of apps.filter(a=>a.Hidden&&a.Open&&!(a.Rule||"").startsWith("Excluded"))){
  row(a.Name,a.Rule,()=>appEdit("unhide",a.Name),"Unhide","Unhide "+a.Name);
 }
 const add=el("div","mt-3 flex gap-2 border-t border-[#25323e] pt-3"),input=el("input","min-w-0 flex-1");
 input.type="text";input.placeholder="game.exe";input.setAttribute("aria-label","Program to exclude");
 const apply=button("Exclude",()=>{const v=input.value.trim();if(v){appEdit("exclude",v,v);input.value="";}},ui.small);apply.dataset.part="exclude";
 input.onkeydown=e=>{if(e.key==="Enter")apply.click();};
 add.append(input,apply);card.append(add);
 updaters.push(()=>{for(const b of buttons)b.disabled=!editable();apply.disabled=!editable()||!input.value.trim();});
 input.addEventListener("input",update);
}
// mediaCard shows one session: artwork, title, source, progress with a seek
// slider and play/pause, which also focuses it; the focused card adds Previous,
// Next and Mute.
function mediaCard(id,parent){
 const info=()=>(c("nowplaying.status")?.ViewData?.Sessions||[]).find(s=>s.ID===id)||{};
 const card=el("article",ui.strip+" flex flex-col gap-2 data-[focused=true]:border-active"),head=el("div","flex items-center gap-3"),art=el("div","grid size-16 shrink-0 place-items-center text-active"),text=el("div","min-w-0 flex-1");
 const title=el("h3","truncate text-sm font-semibold"),artist=el("div","truncate text-[13px]"),source=el("div","truncate text-[11px] text-muted");
 card.dataset.part="media-session";text.append(title,artist,source);head.append(art,text);
 const seek=el("input","w-full accent-active");seek.type="range";seek.min="0";seek.step="1000";seek.dataset.part="seek";
 const times=el("div","flex justify-between text-[11px] text-muted tabular-nums"),elapsed=el("span"),length=el("span");times.append(elapsed,length);
 // While dragged the slider keeps the pointer's position; afterwards it follows
 // the session again (focus alone would freeze it after one drag).
 let dragging=false;
 seek.onpointerdown=()=>{dragging=true;};
 seek.onpointerup=seek.onpointercancel=()=>{dragging=false;};
 seek.onchange=()=>{dragging=false;request(c(id),"set",seek.value);};
 const actions=el("div",ui.actions),status=el("small","text-[11px]");
 const prev=iconButton("skip-back","Previous",()=>press("nowplaying.prev")),play=iconButton("play","Play or pause",()=>press(id)),next=iconButton("skip-forward","Next",()=>press("nowplaying.next"));
 const mute=button("Mute",()=>press("nowplaying.mute"),ui.toggle+" "+ui.small);
 play.dataset.part="play";mute.dataset.part="mute";
 actions.append(prev,play,next,mute,status);card.append(head,seek,times,actions);parent.append(card);
 const clock=ms=>{const s=Math.max(0,Math.floor((ms||0)/1000));return s>=3600?Math.floor(s/3600)+":"+String(Math.floor(s/60)%60).padStart(2,"0")+":"+String(s%60).padStart(2,"0"):Math.floor(s/60)+":"+String(s%60).padStart(2,"0");};
 updaters.push(()=>{
  const item=c(id),s=info();if(!item)return;
  setArt(art,item.Artwork,()=>icon("disc-3","size-9"),"size-16");
  title.textContent=s.Title||s.App||item.Label;artist.textContent=[s.Artist,s.Album].filter(Boolean).join(" · ");artist.hidden=!artist.textContent;
  source.textContent=s.App||"";card.dataset.focused=String(!!s.Focused);
  play.replaceChildren(icon(s.Status==="Playing"?"pause":"play","size-4 shrink-0"));play.setAttribute("aria-label",(s.Status==="Playing"?"Pause ":"Play ")+(s.Title||s.App||""));
  seek.hidden=times.hidden=!s.DurationMs;seek.max=String(s.DurationMs||0);seek.setAttribute("aria-label","Seek "+(s.Title||s.App||""));
  const position=dragging?Number(seek.value):progressAt(s);
  if(!dragging)seek.value=String(position);
  elapsed.textContent=clock(position);length.textContent=clock(s.DurationMs);
  prev.hidden=!s.Focused||!s.CanPrev;next.hidden=!s.Focused||!s.CanNext;mute.hidden=!s.Focused||!s.CanMute;
  mute.textContent=s.Muted?"Muted":"Mute";mute.setAttribute("aria-pressed",String(!!s.Muted));mute.dataset.tone=s.Muted?"critical":"";
  status.textContent=s.Pending?"Wait":s.Failure||"";status.className="text-[11px] "+(s.Failure?"text-critical":"text-attention");
  seek.disabled=!!pending||!s.CanSeek;play.disabled=!!pending||!s.CanToggle;
  for(const b of [prev,next,mute])b.disabled=!!pending;
 });
}
// extensionCard explains the Snoofer Media extension, which reports each
// browser tab, and shows which browsers are connected.
function extensionCard(parent){
 const card=panel("Browser extension",parent,"mt-[18px]");card.dataset.part="extension";
 const intro=el("p","-mt-2 text-[13px] text-muted","Browsers show Windows only one session for all their tabs. The Snoofer Media extension reports each tab separately, so every video or song can be paused and sought on its own. It connects by itself; there is nothing to pair.");
 const browsers=el("div","mt-3 grid gap-1 text-[13px]"),problem=el("p","mt-2 text-[13px] text-critical");
 const steps=el("ol","mt-3 list-decimal space-y-1 pl-5 text-[13px] text-muted");
 for(const step of ["Install Snoofer Media from the Chrome Web Store (Chrome, Brave, Edge) or Firefox Add-ons.","In Firefox, click the extension's toolbar button and choose Allow on all sites.","Play something in a tab; it appears above."])steps.append(el("li","",step));
 const dev=el("p","mt-2 text-[12px] text-muted","Developing: run npm run build in the repository's extension folder, then load extension/dist/chrome unpacked (Developer mode) or extension/dist/firefox as a temporary add-on.");
 card.append(intro,browsers,problem,steps,dev);
 updaters.push(()=>{
  const v=c("nowplaying.status")?.ViewData||{};
  browsers.replaceChildren();
  for(const b of v.Browsers||[])browsers.append(el("div","text-active",b.Name+" connected · "+b.Sessions+" with media"+(b.Version?" · extension "+b.Version:"")));
  if(!(v.Browsers||[]).length)browsers.append(el("div","text-muted","No browser connected."));
  problem.textContent=[v.Bridge?.Error,v.Bridge?.Refused,v.Windows&&v.Windows!=="Connected"?"Windows media sessions: "+v.Windows:""].filter(Boolean).join(" · ");problem.hidden=!problem.textContent;
 });
}
function buildMedia(){
 if(!(state.Plugins||{}).nowplaying){empty(root,"Now playing is not part of this build.");return;}
 if(!state.Enabled?.nowplaying){
  const card=panel("Now playing is off",root,"mb-[18px] border-[#345365]");card.dataset.part="setup";card.append(el("p","mt-2 mb-3.5 text-muted","Show and control what is playing in each app and browser tab."));
  const actions=el("div",ui.actions);actions.append(button("Enable Now playing",()=>send({Kind:"selection",Plugin:"nowplaying",Enable:true}),ui.primary));card.append(actions);return;
 }
 const sessions=[...controls.values()].filter(v=>v.Collection==="nowplaying.sessions").sort((a,b)=>(a.Order||0)-(b.Order||0));
 if(!sessions.length)empty(root,"Nothing is playing. Start music or a video in any app or browser tab.");
 else{const grid=el("div","grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-[18px]");root.append(grid);for(const item of sessions)mediaCard(item.ID,grid);}
 extensionCard(root);
}
// pluginOff renders the card for a plugin that is missing or disabled, and
// reports whether it did.
function pluginOff(id,title,text,parent){
 if(!(state.Plugins||{})[id]){const card=panel(title,parent);card.append(el("p","text-muted",title+" is not part of this build."));return true;}
 if(state.Enabled?.[id])return false;
 const card=panel(title+" is off",parent,"border-[#345365]");card.dataset.part="setup";card.append(el("p","mt-2 mb-3.5 text-muted",text));
 const actions=el("div",ui.actions);actions.append(button("Enable "+title,()=>send({Kind:"selection",Plugin:id,Enable:true}),ui.primary));card.append(actions);return true;
}
// stateLine shows a status control's value under a card title, toned by its state.
function stateLine(id,parent){
 const line=el("p","-mt-1.5 mb-3 text-[13px]");line.dataset.part="state";parent.append(line);
 updaters.push(()=>{const item=c(id);line.textContent=item?display(item):"";line.className="-mt-1.5 mb-3 text-[13px] "+(item?tc(tone(item)):"");});
}
function buildMeetings(){
 const grid=el("div",ui.grid);grid.dataset.part="meetings";root.append(grid);
 if(!pluginOff("insta360","Camera","Insta360 Link 2 privacy, tracking and framing.",grid)){
  const card=panel("Camera",grid);card.dataset.part="camera";
  if(!c("insta360.privacy"))card.append(el("p","text-muted","Camera: "+(state.Plugins.insta360||"Starting")));
  else{stateLine("insta360.state",card);control("insta360.privacy",card,"Privacy");control("insta360.tracking",card,"Tracking");control("insta360.framing",card,"Framing");control("insta360.reset",card,"Reset position");}
 }
 if(!pluginOff("discord","Discord","Discord mute, deafen, camera and screen share.",grid)){
  const card=panel("Discord",grid);card.dataset.part="discord";
  if(!c("discord.channel"))card.append(el("p","text-muted","Discord: "+(state.Plugins.discord||"Starting")));
  else{
   stateLine("discord.channel",card);
   const setup=el("p","mb-3 text-attention","Add your Discord application's client_id and client_secret to snoofer.json.");setup.dataset.part="discord-setup";card.append(setup);
   const connect=el("div",ui.actions+" mb-3");connect.dataset.part="discord-connect";command("discord.connect","Connect",connect,ui.primary);card.append(connect);
   updaters.push(()=>{setup.hidden=c("discord.mute")?.Status!=="Setup needed";connect.hidden=!c("discord.connect")?.Available&&c("discord.connect")?.Status!=="Pending";});
   control("discord.mute",card,"Mute");control("discord.deafen",card,"Deafen");control("discord.video",card,"Camera");control("discord.screenshare",card,"Screen share");control("discord.leave",card,"Leave call");
  }
 }
}
// deckRange is the GUI-local key rectangle, or dial span (dials: true, slot
// indexes from 36), for creating regions; selecting it never dispatches
// anything. It resets when the edited page changes.
let deckRange=null;
function rangeKeys(range){
 if(!range)return [];
 if(range.dials){const out=[];for(let i=Math.min(range.anchor,range.end);i<=Math.max(range.anchor,range.end);i++)out.push(i);return out;}
 const top=Math.min(Math.floor(range.anchor/9),Math.floor(range.end/9)),bottom=Math.max(Math.floor(range.anchor/9),Math.floor(range.end/9));
 const left=Math.min(range.anchor%9,range.end%9),right=Math.max(range.anchor%9,range.end%9);
 const out=[];for(let row=top;row<=bottom;row++)for(let col=left;col<=right;col++)out.push(row*9+col);return out;
}
function regionDialsLabel(first,last){
 const a=Math.min(first,last)+1,b=Math.max(first,last)+1;return a===b?"Dial "+a:"Dials "+a+"–"+b;
}
function regionKeysLabel(first,last){
 const keys=rangeKeys({anchor:first,end:last});return keys.length===1?"Key "+(keys[0]+1):"Keys "+(keys[0]+1)+"–"+(keys.at(-1)+1);
}
function buildDeck(){
 const preview=c("streamdeck.preview");
 if(!preview?.ViewData){empty(root,"Stream Deck is disabled or its editor is not ready.");return;}
 const toolbar=el("div",ui.toolbar);
 const selectors=el("div",ui.grid+" col-span-full flex-1");toolbar.append(selectors);
 control("streamdeck.profile",selectors,"Device");control("streamdeck.page",selectors,"Page");root.append(toolbar);
 const workspace=el("div","grid grid-cols-[minmax(0,1fr)_280px] items-start gap-[18px] min-[1500px]:grid-cols-[minmax(0,1fr)_330px] max-[1120px]:grid-cols-1"),left=el("div"),inspector=el("section",ui.panel+" sticky top-5 max-[1120px]:static");
 inspector.dataset.part="inspector";
 workspace.append(left,inspector);root.append(workspace);
 const frame=el("div","mb-5 rounded-[18px] border border-[#2e414f] bg-deck p-[19px]"),keys=el("div","grid grid-cols-9 gap-[7px]"),dials=el("div","mt-[22px] grid grid-cols-6 gap-2");frame.append(keys,dials);left.append(frame);
 keys.setAttribute("aria-label","Deck keys");dials.setAttribute("aria-label","Deck dials");
 const keyClass="relative flex aspect-square min-w-0 flex-col items-center justify-center overflow-hidden rounded-[7px] bg-key px-[3px] py-[5px] text-[9px] data-[empty=true]:border-[#1c2a36] data-[empty=true]:bg-key-empty data-[region=true]:bg-active-bg data-[range=true]:outline-2 data-[range=true]:outline-dashed data-[range=true]:outline-active aria-pressed:border-2 aria-pressed:border-active aria-pressed:shadow-[0_0_0_2px_#183a46] min-[1500px]:text-xs max-[1120px]:text-[11px]";
 const dialClass="relative data-[region=true]:bg-active-bg data-[range=true]:outline-2 data-[range=true]:outline-dashed data-[range=true]:outline-active flex min-h-[62px] min-w-0 flex-col items-center justify-center bg-key px-1 py-[7px] text-[10px] before:mb-[5px] before:size-[18px] before:rounded-full before:border-2 before:border-[#60798a] before:content-[''] aria-pressed:border-2 aria-pressed:border-active aria-pressed:shadow-[0_0_0_2px_#183a46] *:max-w-full *:truncate";
 function slotButton(index,dial){
  const b=button("",e=>{
   // Shift extends the region selection without selecting a binding.
   if(e.shiftKey&&deckRange&&!!deckRange.dials===dial){deckRange.end=index;update();return;}
   deckRange={page:c("streamdeck.page")?.Value,anchor:index,end:index,dials:dial};
   request(c("streamdeck.slot"),"set",(dial?"Dial ":"Key ")+(dial?index-36+1:index+1));
  },dial?dialClass:keyClass);
  b.dataset.part=dial?"dial":"deck-key";
  const number=el("small","absolute top-0.5 left-1 text-[8px] text-muted",String(dial?index-36+1:index+1)),glyph=el("span","leading-tight text-active"),name=el("span","max-w-full truncate"),source=el("span","text-[8px] text-attention"),regionTag=el("small","absolute right-1 bottom-0.5 text-[8px] text-active");
  if(!dial)b.append(number,glyph);b.append(regionTag);b.append(name,source);(dial?dials:keys).append(b);
  b.onkeydown=e=>{
   if(!e.key.startsWith("Arrow"))return;e.preventDefault();
   if(e.shiftKey){
    if(!deckRange||!!deckRange.dials!==dial)deckRange={page:c("streamdeck.page")?.Value,anchor:index,end:index,dials:dial};
    deckRange.end=dial?36+gridMove(deckRange.end-36,e.key,5,5):gridMove(deckRange.end,e.key,9,36);update();
    (dial?dials:keys).children[dial?deckRange.end-36:deckRange.end]?.focus();return;
   }
   const list=[...(dial?dials:keys).querySelectorAll("button:not(:disabled)")];
   const next=gridMove(list.indexOf(b),e.key,dial?5:9,list.length);list[next]?.focus();
  };
  updaters.push(()=>{
   const view=c("streamdeck.preview")?.ViewData;if(!view)return;
   const slot=dial?view.Dials[index-36]:view.Keys[index],item=c(slot.Control);
   b.setAttribute("aria-pressed",String(view.Selected===index));
   const region=slot.Region??-1,inRange=rangeKeys(deckRange).includes(index);
   b.dataset.region=String(region>=0);b.dataset.range=String(inRange);
   regionTag.textContent=region>=0?"R"+(region+1):"";
   b.setAttribute("aria-label",(dial?"Dial "+(index-35):"Key "+(index+1))+": "+(item?.Label||slot.Label||"Empty")+(slot.Source?" · "+slot.Source:"")+(region>=0?" · Region "+(region+1):"")+(inRange?" · In selection":""));
   b.title=b.getAttribute("aria-label");b.disabled=!!pending;
   b.dataset.empty=String(!slot.Control);
   name.textContent=item?.ShortLabel||item?.Label||slot.Label||"";
   source.textContent=slot.Source||"";
   const art=item?.Artwork||"";
   if(glyph.dataset.art!==art||glyph.dataset.control!==slot.Control){
    glyph.dataset.art=art;glyph.dataset.control=slot.Control;glyph.replaceChildren();
    if(art){const image=el("img","size-[30px] object-contain min-[1500px]:size-12");image.src="data:image/png;base64,"+art;image.alt="";glyph.append(image);}
    else if(slot.Control)glyph.append(symbolIcon(slot.Control));
   }
  });
 }
 for(let i=0;i<36;i++)slotButton(i,false);
 for(let i=36;i<41;i++)slotButton(i,true);
 const reserved=el("div",dialClass.replace(/(aria-pressed|data-\[\w+=true\]):\S+ ?/g,"")+" rounded-[7px] border border-line");reserved.append(el("span","","Pages"),el("small","text-muted","Reserved"));dials.append(reserved);
 const tools=panel("Page",left,"grid gap-3");
 control("streamdeck.name",tools,"Name");
 const commands=el("div",ui.actions+" justify-end");tools.append(commands);
 command("streamdeck.earlier","Earlier",commands,"","chevron-left");command("streamdeck.later","Later",commands,"","chevron-right");command("streamdeck.home","Make Home",commands,"","house");
 const more=el("details","mt-4"),summary=el("summary","","Page options");more.append(summary);tools.append(more);
 control("streamdeck.add",more,"New page");
 regionsPanel(left);
 const deletion=el("div",ui.actions+" justify-end");more.append(deletion);
 const del=command("streamdeck.delete","Delete page",deletion,ui.danger);
 if(del)del.onclick=()=>{if(window.confirm("Delete this page from the draft? Save applies the deletion."))press("streamdeck.delete");};
 const selected=preview.ViewData.Selected,dial=selected>=36;
 const eyebrow=el("div","text-[10px] font-semibold tracking-[1.7px] text-muted",dial?"DIAL "+(selected-35):"KEY "+(selected+1));eyebrow.dataset.part="eyebrow";
 inspector.append(eyebrow,el("h2",ui.h2,"Binding"));
 const ownership=el("p",ui.note+" mt-1");inspector.append(ownership);
 updaters.push(()=>{
  const v=c("streamdeck.preview")?.ViewData;if(!v)return;
  const slot=dial?v.Dials[selected-36]:v.Keys[selected];
  ownership.className=slot.Source?ui.note+" mt-1":"mt-1 mb-[15px] text-xs text-muted";
  ownership.textContent=slot.Source==="Auto"?"Auto-filled · assigning a binding pins this position":slot.Source==="Shared"?"Shared across pages":"Page binding";
 });
 control("streamdeck.shared",inspector,"Edit shared binding",{layout:"stack"});
 const search=el("input","w-full");search.type="search";search.placeholder="Find an action";search.setAttribute("aria-label","Find a binding");
 const select=el("select","my-2.5 min-h-[220px] w-full text-xs max-[1120px]:min-h-[150px] *:p-[7px] *:whitespace-normal");select.size=9;select.setAttribute("aria-label","Binding");
 inspector.append(search,select);
 function populate(){
  const value=c("streamdeck.binding")?.Value||"";
  select.replaceChildren();
  const clear=el("option","","Clear binding");clear.value="";select.append(clear);
  for(const item of controls.values()){
   // Go-to and scroll keys are the only Stream Deck controls that can be bound.
   if((item.ID.startsWith("streamdeck.")&&!item.ID.startsWith("streamdeck.goto-")&&!item.ID.startsWith("streamdeck.scroll-")&&item.ID!=="streamdeck.reset-focus") || !compatible(item,dial))continue;
   const label=item.Label+" · "+item.ID.split(".")[0];
   if(!label.toLocaleLowerCase().includes(search.value.toLocaleLowerCase())&&item.ID!==value)continue;
   const option=el("option","",label);option.value=item.ID;option.title=item.ID;select.append(option);
  }
  if(value && ![...select.options].some(o=>o.value===value)){const option=el("option","",value+" · unavailable");option.value=value;select.append(option);}
  select.value=value;
 }
 search.oninput=populate;populate();
 select.onfocus=()=>{select.editRevision=c("streamdeck.binding")?.Revision;};
 select.onchange=()=>{request(c("streamdeck.binding"),"set",select.value,0,select.editRevision);select.blur();};
 updaters.push(()=>{select.disabled=!!pending||!c("streamdeck.binding")?.Available;if(document.activeElement!==select)select.value=c("streamdeck.binding")?.Value||"";});
 const bar=el("div","sticky bottom-3 mt-[18px] flex items-center justify-between gap-3 rounded-[10px] border border-[#345365] bg-[#192a36] px-[18px] py-3.5"),status=el("p","text-xs"),actions=el("div",ui.actions);bar.append(status,actions);root.append(bar);
 const discard=command("streamdeck.cancel","Discard",actions),save=command("streamdeck.save","Save layout",actions,ui.primary);
 updaters.push(()=>{
  const v=c("streamdeck.preview")?.ViewData;status.textContent=c("streamdeck.status")?.Value||"";
  status.className="text-xs "+(v?.Dirty?"text-attention":"text-muted");
  if(discard)discard.disabled=!!pending||!v?.Dirty;if(save)save.disabled=!!pending||!v?.Dirty;
 });
}
function regionsPanel(parent){
 const view=c("streamdeck.preview")?.ViewData;if(!view||!c("streamdeck.region-add"))return;
 const card=panel("Regions",parent,"mt-5 grid gap-3");card.dataset.part="regions";
 const collections=view.Collections||[];
 const sourceSelect=(value,label)=>{
  const select=el("select","min-w-0 flex-1");
  for(const item of collections){const option=el("option","",item.Label);option.value=item.ID;select.append(option);}
  if(value&&!collections.some(item=>item.ID===value)){const option=el("option","",label||value);option.value=value;select.append(option);}
  select.value=value||collections[0]?.ID||"";return select;
 };
 const regions=view.Regions||[];
 if(!regions.length)card.append(el("p",ui.note,"No regions. Select keys or dials (Shift+click or Shift+arrows) and add one."));
 regions.forEach((region,i)=>{
  const row=el("div","flex flex-wrap items-center gap-2.5");row.dataset.part="region";
  const name=el("strong","w-7 text-xs text-active","R"+(i+1)),select=sourceSelect(region.Source,region.Label),keysLabel=el("small","text-muted",region.Legacy?"Whole page · prefix":region.Dials?regionDialsLabel(region.First,region.Last):regionKeysLabel(region.First,region.Last));
  select.setAttribute("aria-label","Region "+(i+1)+" source");
  select.onfocus=()=>{select.editRevision=c("streamdeck.region-source")?.Revision;};
  select.onchange=()=>{request(c("streamdeck.region-source"),"set",i+","+select.value,0,select.editRevision);select.blur();};
  const remove=button("Remove",()=>request(c("streamdeck.region-remove"),"set",String(i)),ui.danger);remove.setAttribute("aria-label","Remove region "+(i+1));
  row.append(name,select,keysLabel,remove);card.append(row);
  updaters.push(()=>{select.disabled=remove.disabled=!!pending;});
 });
 const add=el("div","flex flex-wrap items-center gap-2.5 border-t border-[#25323e] pt-3"),source=sourceSelect("",""),create=button("",()=>{
  if(!deckRange)return;
  request(c("streamdeck.region-add"),"set",deckRange.anchor+","+deckRange.end+","+source.value);
  deckRange=null;
 },ui.primary);
 source.setAttribute("aria-label","New region source");create.dataset.part="add-region";
 add.append(source,create);card.append(add);
 updaters.push(()=>{
  if(deckRange&&deckRange.page!==c("streamdeck.page")?.Value)deckRange=null;
  const keysInRange=rangeKeys(deckRange),current=c("streamdeck.preview")?.ViewData;
  const overlaps=keysInRange.some(k=>((k>=36?current?.Dials[k-36]:current?.Keys[k])?.Region??-1)>=0);
  create.textContent=deckRange?"Add region · "+(deckRange.dials?regionDialsLabel(deckRange.anchor-36,deckRange.end-36):regionKeysLabel(deckRange.anchor,deckRange.end)):"Add region";
  create.disabled=!!pending||!deckRange||overlaps||!collections.length;
  create.title=overlaps?"The selection overlaps a region":!deckRange?"Select keys or dials first":"";
  source.disabled=!collections.length;
 });
}
// Deck key previews map a control's deck icon to the closest Lucide icon.
const deckIcons={"mic-mute":"mic","camera-privacy":"video","camera-privacy-muted":"eye-off","tracking":"scan-face","framing":"frame","camera-reset":"rotate-ccw","call":"phone","call-leave":"phone-off","discord-deafen":"headphones","discord-deafen-muted":"headphone-off","discord-video":"video","discord-video-off":"video-off","screen-share":"screen-share","discord-connect":"link","record-mic":"mic","vr-mic":"mic","speaker-mute":"volume-2","vr-playback":"volume-2","record-computer":"monitor","monitor":"headphones","mic-stack":"power","mode-direct":"audio-lines","mode-element":"audio-lines","tap-pre":"audio-lines","tap-post":"audio-lines","record-toggle":"circle-dot","record-start":"circle-dot","record-stop":"square","soundboard-play":"play","soundboard-stop":"square","soundboard-overlap":"layers","media-prev":"skip-back","media-next":"skip-forward","media-play":"play","open-controls":"sliders-horizontal","defaults":"sliders-horizontal","engine-restart":"refresh-cw","hue-scene":"lightbulb","hue-brightness":"sun","hue-pair":"link","huesync-sync":"monitor","huesync-mode":"layers","huesync-intensity":"waves","hue-motion":"radar","hue-motion-off":"radar","deck-page":"folder","app-audio":"app-window","focus-reset":"crosshair","echo":"audio-lines","tape-play":"play","tape-pause":"pause","tape-stop":"square","tape-rew":"rewind","tape-ff":"fast-forward","deck-up":"chevron-up","deck-down":"chevron-down"};
function symbolIcon(id){
 const mapped=deckIcons[c(id)?.Icon];
 const name=mapped||(/mute/.test(id)?"volume-2":/mic|source|mode/.test(id)?"mic":/play|clip/.test(id)?"play":/stop/.test(id)?"square":/record/.test(id)?"circle-dot":/prev/.test(id)?"skip-back":/next/.test(id)?"skip-forward":/scene/.test(id)?"lightbulb":"circle");
 return icon(name,"size-6 min-[1500px]:size-8");
}
function buildPlugins(){
 for(const id of Object.keys(state.Plugins||{}).sort()){
  const card=el("section",ui.panel+" mb-3 flex items-center justify-between gap-5"),text=el("div"),status=el("p","mt-[5px] text-xs break-words"),toggle=button("",()=>send({Kind:"selection",Plugin:id,Enable:!state.Enabled[id]}),ui.toggle);
  card.dataset.part="plugin";
  text.append(el("h2",ui.h2,id),status);card.append(text,toggle);root.append(card);
  updaters.push(()=>{status.textContent=state.Plugins[id];status.className="mt-[5px] text-xs break-words "+(state.Plugins[id]==="Running"?"text-active":state.Plugins[id]==="Disabled"?"text-muted":"text-critical");toggle.textContent=state.Enabled[id]?"Disable":"Enable";toggle.disabled=!!pending;});
 }
}
async function copyText(text,feedback){
 try{
  if(navigator.clipboard?.writeText)await navigator.clipboard.writeText(text);
  else throw new Error("Clipboard unavailable");
 }catch{
  // WebView clipboard permissions vary; fall back to a temporary selection.
  const area=el("textarea","fixed opacity-0");area.value=text;area.setAttribute("readonly","");document.body.append(area);area.select();
  const ok=document.execCommand("copy");area.remove();
  if(!ok){showError("Copy failed: clipboard unavailable");return;}
 }
 feedback.textContent="Copied";setTimeout(()=>{feedback.textContent="";},1500);
}
function reportCard(id,parent){
 const card=el("section","min-w-0 rounded-[9px] border border-l-[3px] border-seam border-l-line bg-sidebar px-4 py-3.5 data-[tone=active]:border-l-active data-[tone=attention]:border-l-attention data-[tone=critical]:border-l-critical"),head=el("div","mb-2 flex items-center justify-between gap-2.5"),name=el("h3","text-sm font-bold",c(id).Label),state=badge(""),copied=el("small","ml-1.5 text-active"),copy=withIcon(button("Copy details",()=>copyText(connectionText(c(id)),copied),ui.small),"copy");
 card.dataset.part="app-card";
 const endpoint=el("code","mb-1.5 block font-mono text-xs break-words text-muted"),times=el("p","mb-1.5 text-xs text-muted"),error=el("p","mb-2 text-xs break-words"),list=el("dl","my-2 mb-2.5 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-3 gap-y-[3px] text-xs");
 endpoint.dataset.part="endpoint";times.dataset.part="app-times";error.dataset.part="app-error";list.dataset.part="app-details";
 head.append(name,state);card.append(head,endpoint,times,error,list);
 const foot=el("div",ui.actions);foot.append(copy,copied);card.append(foot);parent.append(card);
 updaters.push(()=>{
  const item=c(id);if(!item)return;const conn=item.Connection||{},now=Date.now(),accent=connectionTone(conn);
  state.textContent=item.Value||conn.State||"";state.dataset.tone=accent;card.dataset.tone=accent;
  endpoint.textContent=conn.Endpoint||"";endpoint.hidden=!conn.Endpoint;
  const parts=[];
  if(timeValue(conn.Since)!==null)parts.push("Since "+relativeTime(conn.Since,now));
  if(timeValue(conn.LastActivity)!==null)parts.push("Last activity "+relativeTime(conn.LastActivity,now)+(stale(conn,now)?" · stale":""));
  times.textContent=parts.join(" · ");times.hidden=!parts.length;times.className="mb-1.5 text-xs "+(stale(conn,now)?"text-attention":"text-muted");
  const current=["error","attention","connecting"].includes(conn.State)||(conn.State==="disconnected"&&accent==="critical");
  error.textContent=conn.LastError?(current?"":"Last error: ")+conn.LastError+(timeValue(conn.LastErrorAt)!==null?" · "+relativeTime(conn.LastErrorAt,now):""):"";
  error.hidden=!conn.LastError;error.className="mb-2 text-xs break-words "+(current?"text-critical":"text-muted");
  const rows=(conn.Details||[]).filter(d=>d.Label!=="Required");
  const key=JSON.stringify(rows);
  if(list.dataset.key!==key){list.dataset.key=key;list.replaceChildren();for(const d of rows)list.append(el("dt","text-muted",d.Label),el("dd","break-words",d.Value));}
  list.hidden=!rows.length;
 });
}
function buildApps(){
 const reports=[...controls.values()].filter(v=>v.Kind==="connection").sort((a,b)=>(a.Group||"").localeCompare(b.Group||"")||a.Label.localeCompare(b.Label));
 const bar=el("div",ui.toolbar+" justify-between"),counts=el("div","flex flex-1 flex-wrap gap-2"),copied=el("small","ml-1.5 text-active");
 counts.dataset.part="summary";
 bar.append(counts,withIcon(button("Copy all",()=>copyText(reports.map(r=>connectionText(c(r.ID)||r)).join("\n\n"),copied),ui.small),"copy"),copied);root.append(bar);
 updaters.push(()=>{
  const tally={active:0,attention:0,critical:0,other:0};
  for(const r of reports){const t=connectionTone(c(r.ID)?.Connection);tally[t in tally?t:"other"]++;}
  counts.replaceChildren(...[["active","OK"],["attention","Attention"],["critical","Problems"],["other","Idle"]].map(([k,label])=>badge(tally[k]+" "+label,k==="other"?"":k)));
 });
 if(!reports.length)empty(root,"No third-party apps are reporting. Enable plugins in Plugins.");
 for(const group of [...new Set(reports.map(r=>r.Group||"Other"))]){
  const card=panel(group,root,"mb-[18px]"),grid=el("div","grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-3");card.append(grid);
  for(const r of reports.filter(v=>(v.Group||"Other")===group))reportCard(r.ID,grid);
 }
 const idle=Object.entries(state.Plugins||{}).filter(([,status])=>status!=="Running").sort();
 if(idle.length){
  const card=panel("Not monitored",root,"mb-[18px]"),list=el("ul","list-disc pl-[18px] text-[13px] text-muted");list.dataset.part="not-monitored";card.append(list);
  for(const [id,status] of idle)list.append(el("li","",id+" — "+status));
 }
}
function buildDiagnostics(){
 const toolbar=el("div",ui.toolbar);toolbar.append(withIcon(button("Retry plugins",()=>send({Kind:"retry"})),"refresh-cw"));root.append(toolbar);
 const card=panel("Audio engine",root);control("audio.health",card);control("audio.interface",card);control("audio.asio-unavailable",card);control("audio.auto-recover",card,"Automatic recovery");
 const engine=el("div",ui.actions+" mt-3");card.append(engine);command("audio.engine-restart","Restart audio engine",engine,"","refresh-cw");command("audio.engine-confirm","Confirm restart",engine,ui.danger);
 const details=el("section",ui.panel+" mt-[18px] text-[13px] break-words whitespace-pre-wrap");root.append(details);
 updaters.push(()=>{
  details.replaceChildren(el("h2",ui.h2,"Current notices"));
  const notices=[...controls.values()].filter(v=>v.Status);
  if(!notices.length)details.append(el("p","my-2 text-muted","No control notices."));
  for(const item of notices){details.append(el("h3",ui.h3+" mt-3",item.Label),el("p","my-2 "+(tc(tone(item))||"text-muted"),item.Status));}
 });
}
function layoutKey(){
 // Values and telemetry are updated in place. Only structure/context rebuilds a screen.
 const list=[...controls.values()].map(v=>[v.ID,v.Label,v.Kind,v.Group,v.Options,v.OptionLabels]);
 return JSON.stringify([screen,list,Object.keys(state.Plugins||{}),screen==="lights"?[state.Enabled?.hue,c("hue.group")?.Value,c("hue.rooms")?.ViewData]:null,screen==="meetings"?[state.Enabled?.insta360,state.Enabled?.discord]:null,screen==="media"?[state.Enabled?.nowplaying,(c("nowplaying.status")?.ViewData?.Sessions||[]).map(s=>[s.ID,s.CanSeek,s.CanMute]),(c("nowplaying.status")?.ViewData?.Browsers||[]).map(b=>b.Name)]:null,screen==="appaudio"?[state.Enabled?.appaudio,(c("appaudio.status")?.ViewData?.Apps||[]).map(a=>[a.ID,a.Name,a.Hidden,a.Picked,a.Open,a.Rule,a.Naming]),c("appaudio.status")?.ViewData?.Exclude]:null,screen==="routing"?[c("audio.priorities")?.ViewData,[1,2,3].map(n=>[c("audio.slot-"+n)?.Hidden,c("audio.slot-"+n)?.Status,c("audio.slot-"+n)?.ViewData])]:null,screen==="deck"?[c("streamdeck.preview")?.ViewData?.Selected,c("streamdeck.page")?.Value,c("streamdeck.profile")?.Value,c("streamdeck.shared")?.Value,c("streamdeck.preview")?.ViewData?.Regions,c("streamdeck.preview")?.ViewData?.Collections]:null]);
}
function build(){
 widgets.length=0;updaters.length=0;root.replaceChildren();
 for(const b of document.querySelectorAll("[data-screen]")){if(b.dataset.screen===screen)b.setAttribute("aria-current","page");else b.removeAttribute("aria-current");}
 $("#eyebrow").textContent=titles[screen][0];$("#title").textContent=titles[screen][1];
 ({audio:buildAudio,routing:buildRouting,soundboard:buildSoundboard,lights:buildLights,appaudio:buildAppAudio,meetings:buildMeetings,media:buildMedia,deck:buildDeck,plugins:buildPlugins,apps:buildApps,diagnostics:buildDiagnostics})[screen]();
}
function update(){
 if(pending&&(Date.now()>pending.until || state.Notice!==pending.notice || (pending.id&&c(pending.id)?.Revision!==pending.revision) || (!pending.id&&state.Confirmation)))pending=null;
 for(const w of widgets){
  const item=c(w.id)||w.initial;
  w.title.classList.toggle("text-muted",!!item.Subdued);w.note.textContent=item.Subdued&&item.Status==="VR override"?"":item.Status||"";w.note.className="mt-1 block text-[11px] break-words "+tc(tone(item));
  const input=w.input;
  if(item.Kind==="toggle"){input.textContent=display(item);input.setAttribute("aria-pressed",String(item.Value==="On"));input.dataset.tone=tone(item);}
  else if(item.Kind==="selection"||item.Kind==="text"){if(document.activeElement!==input)input.value=item.Value||"";}
  else if(item.Kind==="status"){input.textContent=display(item);input.className="max-w-[65%] text-right break-words "+tc(tone(item));}
  if(["toggle","selection","text","command"].includes(item.Kind))input.disabled=!!pending||!item.Available;
 }
 for(const fn of updaters)fn();
 // Audio health lives on the Audio and Diagnostics screens, not the shared header.
 $("#summary").replaceChildren();
 if(pending)$("#summary").append(badge("Sending","attention"));
 if(c("audio.normal-source")?.Subdued)$("#summary").append(badge("VR active","active"));
 if(c("audio.pause-devices")?.Value==="On")$("#summary").append(badge("Devices paused","attention"));
 if(c("audio.pause-sends")?.Value==="On")$("#summary").append(badge("Sends paused","attention"));
 root.setAttribute("aria-busy",String(!!pending));
 refreshNotice();
 const dialog=$("#confirm");
 if(state.Confirmation){$("#confirm-text").textContent=state.Confirmation;if(!dialog.open)dialog.showModal();}
 else if(dialog.open)dialog.close();
}
function receive(next){
 state=next||state;controls=controlsByID(state);
 const nextKey=layoutKey();
 if(nextKey!==signature){
  // Preserve edits until blur if a provider changes its available choices mid-edit.
  const focused=document.activeElement;
  if(root.contains(focused)&&["INPUT","SELECT"].includes(focused.tagName)&&!pending){update();return;}
  signature=nextKey;build();
 }
 update();
}
const navIcons={audio:"audio-waveform",routing:"route",soundboard:"music",lights:"lightbulb",appaudio:"app-window",meetings:"video",media:"disc-3",deck:"layout-grid",plugins:"puzzle",apps:"plug",diagnostics:"activity"};
for(const b of document.querySelectorAll("[data-screen]")){
 b.className=ui.nav;
 b.prepend(icon(navIcons[b.dataset.screen],"size-5 shrink-0"));
 b.onclick=()=>{screen=b.dataset.screen;signature="";receive(state);};
 b.onkeydown=e=>{if(!["ArrowUp","ArrowDown","Home","End"].includes(e.key))return;e.preventDefault();const items=[...document.querySelectorAll("[data-screen]")];const i=items.indexOf(b);const next=e.key==="Home"?0:e.key==="End"?items.length-1:gridMove(i,e.key,1,items.length);items[next].focus();items[next].click();};
}
$("#confirm-no").onclick=()=>send({Kind:"cancel"});
$("#confirm-yes").onclick=()=>send({Kind:"confirm"});
$("#confirm").oncancel=e=>{e.preventDefault();send({Kind:"cancel"});};
async function poll(){
 try {
  if(!window.go?.app?.Desktop)throw new Error("Desktop connection unavailable");
  const next=await window.go.app.Desktop.State();receive(next);
 }catch(error){showError(error);update();}
 // Meters need a faster refresh than the rest of the GUI.
 setTimeout(poll,screen==="audio"||screen==="soundboard"||screen==="appaudio"||screen==="media"?100:200);
}
poll();
