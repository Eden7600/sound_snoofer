import {controlsByID,compatible,tone,meterValue,display,gridMove,numericValue,sceneRoom,relativeTime,timeValue,stale,connectionTone,connectionText} from "./model.mjs";
import {icon} from "./icons.mjs";

const $=s=>document.querySelector(s);
const root=$("#content");
let state={Controls:[],Plugins:{},Enabled:{}}, controls=new Map(), screen="audio", signature="", pending=null;
let localError="", dismissedNotice="", connected=false;
const widgets=[], updaters=[];
const titles={audio:["AUDIO","Audio"],soundboard:["LIBRARY","Soundboard"],lights:["LIGHTING","Lights"],deck:["CONTROL SURFACE","Stream Deck"],plugins:["SYSTEM","Plugins"],apps:["SYSTEM","Third-party apps"],diagnostics:["SYSTEM","Diagnostics"]};

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
function mixer(id,parent,title) {
 if(!c(id))return;
 const node=el("div",ui.strip), value=el("div",ui.readout),bar=el("div","relative mt-3 h-2 overflow-hidden rounded-[3px] bg-[#283542]"),signal=el("div","h-full w-full bg-[linear-gradient(90deg,var(--color-meter-green)_0%,var(--color-meter-green)_76%,var(--color-meter-amber)_80%,var(--color-meter-red)_98%)] [clip-path:inset(0_100%_0_0)]"),text=el("div",ui.meterLabel),actions=el("div",ui.gainButtons);
 const device=el("div",ui.meterLabel);
 node.dataset.part="strip";
 node.append(el("h3",ui.stripTitle,title),device,value);bar.append(signal);node.append(bar,text,actions);
 actions.append(iconButton("minus",title+" down",()=>request(c(id),"adjust","",-1)),button("0 dB",()=>press(id),ui.toggle),iconButton("plus",title+" up",()=>request(c(id),"adjust","",1)));
 // Only soundboard's press means reset; audio gain press retains its existing mute behavior.
 if(id!=="soundboard.volume"){actions.children[1].textContent="Mute";actions.children[1].title="Toggle mute";}
 parent.append(node);
 updaters.push(()=>{
  const control=c(id);device.textContent=id==="audio.gain-playback"?(c("audio.playback-device")?.Value||""):id==="audio.gain-mic"?(c("audio.mic-device")?.Value||""):"";device.hidden=!device.textContent;value.textContent=control ? display(control) : "Unavailable";
  const db=meterValue(control?.Meter);
  signal.style.clipPath="inset(0 "+(db===null?100:-(db/60)*100)+"% 0 0)";
  text.textContent=db===null?"LEVEL N/A":db.toFixed(1)+" dBFS";
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
 if(recording.length){const card=panel("Recording",grid);for(const item of recording){control(item.ID,card);used.add(item.ID);}}
 const other=[...controls.values()].filter(v=>v.ID.startsWith("audio.")&&!v.SurfaceOnly&&!used.has(v.ID));
 if(other.length){const card=panel("Routing & recovery",grid);for(const item of other)control(item.ID,card);}
}
function setArt(art,artwork,fallback,size){
 if(art.dataset.art===(artwork||""))return;
 art.dataset.art=artwork||"";art.replaceChildren();
 if(artwork){const image=el("img",size+" object-contain");image.src="data:image/png;base64,"+artwork;image.alt="";art.append(image);}
 else art.append(fallback());
}
function clipState(item,active){return item.Value===active?"playing":item.Value==="Wait"||item.Status==="Pending"?"wait":tone(item)==="critical"?"error":"";}
function buildSoundboard(){
 const clips=[...controls.values()].filter(v=>v.ID.startsWith("soundboard.clip-"));
 const toolbar=el("div",ui.toolbar),search=el("input","min-w-[170px] flex-1");search.type="search";search.placeholder="Search clips";search.setAttribute("aria-label","Search clips");toolbar.append(icon("search","size-5 shrink-0 text-muted"),search);
 command("soundboard.stop","Stop",toolbar,"","square");root.append(toolbar);
 if(c("soundboard.volume")){const card=panel("Playback",root,"mb-[18px]");control("soundboard.status",card,"Library");mixer("soundboard.volume",card,"VOLUME");}
 if(!clips.length){empty(root,"No clips available. Check the soundboard plugin and library folder.");return;}
 const grid=el("div",ui.clipGrid);root.append(grid);
 for(const initial of clips){
  const b=button("",()=>press(initial.ID),ui.clip),art=el("div",ui.clipArt),name=el("strong","text-[13px] break-words",initial.Label),status=el("small","mt-[5px] text-[11px] text-muted");
  b.dataset.part="clip";b.append(art,name,status);b.title=initial.Label;grid.append(b);
  updaters.push(()=>{
   const item=c(initial.ID);if(!item)return;
   b.disabled=!!pending||!item.Available;b.dataset.state=clipState(item,"Playing");
   status.textContent=item.Status||display(item);status.className="mt-[5px] text-[11px] "+(tc(tone(item))||"text-muted");
   setArt(art,item.Artwork,()=>icon("play","size-9"),"size-16");
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
 const keyClass="relative flex aspect-square min-w-0 flex-col items-center justify-center overflow-hidden rounded-[7px] bg-key px-[3px] py-[5px] text-[9px] data-[empty=true]:border-[#1c2a36] data-[empty=true]:bg-key-empty aria-pressed:border-2 aria-pressed:border-active aria-pressed:shadow-[0_0_0_2px_#183a46] min-[1500px]:text-xs max-[1120px]:text-[11px]";
 const dialClass="flex min-h-[62px] min-w-0 flex-col items-center justify-center bg-key px-1 py-[7px] text-[10px] before:mb-[5px] before:size-[18px] before:rounded-full before:border-2 before:border-[#60798a] before:content-[''] aria-pressed:border-2 aria-pressed:border-active aria-pressed:shadow-[0_0_0_2px_#183a46] *:max-w-full *:truncate";
 function slotButton(index,dial){
  const b=button("",()=>request(c("streamdeck.slot"),"set",(dial?"Dial ":"Key ")+(dial?index-36+1:index+1)),dial?dialClass:keyClass);
  b.dataset.part=dial?"dial":"deck-key";
  const number=el("small","absolute top-0.5 left-1 text-[8px] text-muted",String(dial?index-36+1:index)),glyph=el("span","leading-tight text-active"),name=el("span","max-w-full truncate"),source=el("span","text-[8px] text-attention");
  if(!dial)b.append(number,glyph);b.append(name,source);(dial?dials:keys).append(b);
  b.onkeydown=e=>{
   if(!e.key.startsWith("Arrow"))return;e.preventDefault();
   const list=[...(dial?dials:keys).querySelectorAll("button:not(:disabled)")];
   const next=gridMove(list.indexOf(b),e.key,dial?5:9,list.length);list[next]?.focus();
  };
  updaters.push(()=>{
   const view=c("streamdeck.preview")?.ViewData;if(!view)return;
   const slot=dial?view.Dials[index-36]:view.Keys[index],item=c(slot.Control);
   b.setAttribute("aria-pressed",String(view.Selected===index));
   b.setAttribute("aria-label",(dial?"Dial "+(index-35):"Key "+index)+": "+(item?.Label||slot.Label||"Empty")+(slot.Source?" · "+slot.Source:""));
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
 const reserved=el("div",dialClass.replace(/aria-pressed:\S+ ?/g,"")+" rounded-[7px] border border-line");reserved.append(el("span","","Pages"),el("small","text-muted","Reserved"));dials.append(reserved);
 const tools=panel("Page",left,"grid gap-3");
 control("streamdeck.name",tools,"Name");
 const commands=el("div",ui.actions+" justify-end");tools.append(commands);
 command("streamdeck.earlier","Earlier",commands,"","chevron-left");command("streamdeck.later","Later",commands,"","chevron-right");command("streamdeck.home","Make Home",commands,"","house");
 const more=el("details","mt-4"),summary=el("summary","","Page options");more.append(summary);tools.append(more);
 control("streamdeck.add",more,"New page");
 control("streamdeck.auto-controls",more,"Automatic prefix");
 const deletion=el("div",ui.actions+" justify-end");more.append(deletion);
 const del=command("streamdeck.delete","Delete page",deletion,ui.danger);
 if(del)del.onclick=()=>{if(window.confirm("Delete this page from the draft? Save applies the deletion."))press("streamdeck.delete");};
 const selected=preview.ViewData.Selected,dial=selected>=36;
 const eyebrow=el("div","text-[10px] font-semibold tracking-[1.7px] text-muted",dial?"DIAL "+(selected-35):"KEY "+selected);eyebrow.dataset.part="eyebrow";
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
   if(item.ID.startsWith("streamdeck.") || !compatible(item,dial))continue;
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
// Deck key previews map a control's deck icon to the closest Lucide icon.
const deckIcons={"mic-mute":"mic","record-mic":"mic","vr-mic":"mic","speaker-mute":"volume-2","vr-playback":"volume-2","record-computer":"monitor","monitor":"headphones","mic-stack":"power","mode-direct":"audio-lines","mode-element":"audio-lines","tap-pre":"audio-lines","tap-post":"audio-lines","record-toggle":"circle-dot","record-start":"circle-dot","record-stop":"square","soundboard-play":"play","soundboard-stop":"square","media-prev":"skip-back","media-next":"skip-forward","media-play":"play","open-controls":"sliders-horizontal","defaults":"sliders-horizontal","engine-restart":"refresh-cw","hue-scene":"lightbulb","hue-brightness":"sun","hue-pair":"link","huesync-sync":"monitor","huesync-mode":"layers","huesync-intensity":"waves"};
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
 const card=panel("Audio engine",root);control("audio.health",card);control("audio.interface",card);control("audio.asio-unavailable",card);
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
 return JSON.stringify([screen,list,Object.keys(state.Plugins||{}),screen==="lights"?[state.Enabled?.hue,c("hue.group")?.Value]:null,screen==="deck"?[c("streamdeck.preview")?.ViewData?.Selected,c("streamdeck.page")?.Value,c("streamdeck.profile")?.Value,c("streamdeck.shared")?.Value]:null]);
}
function build(){
 widgets.length=0;updaters.length=0;root.replaceChildren();
 for(const b of document.querySelectorAll("[data-screen]")){if(b.dataset.screen===screen)b.setAttribute("aria-current","page");else b.removeAttribute("aria-current");}
 $("#eyebrow").textContent=titles[screen][0];$("#title").textContent=titles[screen][1];
 ({audio:buildAudio,soundboard:buildSoundboard,lights:buildLights,deck:buildDeck,plugins:buildPlugins,apps:buildApps,diagnostics:buildDiagnostics})[screen]();
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
 const health=c("audio.health");$("#summary").replaceChildren();
 if(pending)$("#summary").append(badge("Sending","attention"));
 if(health)$("#summary").append(badge(health.Value||"Unknown",tone(health)));
 if(c("audio.normal-source")?.Subdued)$("#summary").append(badge("VR active","active"));
 const link=$("#connection");link.textContent=connected?"Connected":"Disconnected";link.className=ui.badge;link.dataset.tone=connected?"active":"critical";
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
const navIcons={audio:"audio-waveform",soundboard:"music",lights:"lightbulb",deck:"layout-grid",plugins:"puzzle",apps:"plug",diagnostics:"activity"};
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
  const next=await window.go.app.Desktop.State();connected=true;receive(next);
 }catch(error){connected=false;showError(error);update();}
 setTimeout(poll,200);
}
poll();
