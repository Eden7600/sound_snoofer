import {controlsByID,compatible,tone,meterValue,display,gridMove,numericValue,sceneRoom,relativeTime,timeValue,stale,connectionTone,connectionText} from "./model.mjs";
import {icon} from "./icons.mjs";

const $=s=>document.querySelector(s);
const root=$("#content");
let state={Controls:[],Plugins:{},Enabled:{}}, controls=new Map(), screen="audio", signature="", pending=null;
let localError="", dismissedNotice="", connected=false;
const widgets=[], updaters=[];
const titles={audio:["AUDIO","Audio"],soundboard:["LIBRARY","Soundboard"],lights:["LIGHTING","Lights"],deck:["CONTROL SURFACE","Stream Deck"],plugins:["SYSTEM","Plugins"],apps:["SYSTEM","Third-party apps"],diagnostics:["SYSTEM","Diagnostics"]};
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
 const node=el("section","panel "+className), head=el("div","panel-head");
 head.append(el("h2","",title)); node.append(head); parent.append(node); return node;
}
function withIcon(node,name){node.prepend(icon(name,"size-4 shrink-0"));return node;}
function empty(parent,text){parent.append(el("div","empty",text));}
function c(id){return controls.get(id);}
function showError(error){localError=String(error); refreshNotice();}
function refreshNotice(){
 const message=localError || (state.Notice!==dismissedNotice ? state.Notice : "");
 const node=$("#notice"); node.hidden=!message; node.replaceChildren();
 if(message){node.append(button("Dismiss",()=>{localError="";dismissedNotice=state.Notice;refreshNotice();},"small"),document.createTextNode(message));}
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
 const b=button("",handler);b.append(icon(name,"size-4 shrink-0"));b.setAttribute("aria-label",label);b.title=label;return b;
}
function command(id,label,parent,className="",iconName="") {
 if(!c(id)) return null;
 const b=button(label,()=>press(id),className);if(iconName)b.prepend(icon(iconName,"size-4 shrink-0"));parent.append(b);
 updaters.push(()=>{b.disabled=!!pending || !c(id)?.Available;});
 return b;
}
function control(id,parent,label) {
 const initial=c(id);if(!initial)return;
 const row=el("div","control-row"), caption=el("div","control-label"), title=el("label","",label||initial.Label), note=el("small");
 caption.append(title,note); row.append(caption);
 let input;
 if(initial.Kind==="toggle") {
  input=button("",()=>request(c(id),"press"),"toggle");input.setAttribute("aria-label",label||initial.Label);
 } else if(initial.Kind==="selection") {
  input=el("select"); input.id="field-"+id;title.htmlFor=input.id;
  for(const value of initial.Options||[]) {const option=el("option","",initial.OptionLabels?.[value]??(value||"None"));option.value=value;input.append(option);}
  input.onfocus=()=>{input.editRevision=c(id)?.Revision;};
  input.onchange=()=>{request(c(id),"set",input.value,0,input.editRevision);input.blur();};
 } else if(initial.Kind==="text") {
  input=el("input");input.type="text";input.id="field-"+id;title.htmlFor=input.id;
  input.onfocus=()=>{input.editRevision=c(id)?.Revision;};
  input.onkeydown=e=>{if(e.key==="Enter"){request(c(id),"set",input.value,0,input.editRevision);input.blur();}if(e.key==="Escape"){input.value=c(id)?.Value||"";input.blur();}};
  input.title="Enter to apply";
 } else if(initial.Kind==="command") {
  input=button(label||initial.Label,()=>press(id));title.textContent=initial.Value||"";
 } else if(initial.Kind==="numeric") {
  input=el("div","actions");
  const value=el("span","value");
  input.append(iconButton("minus","Decrease "+(label||initial.Label),()=>request(c(id),"adjust","",-1)),value,iconButton("plus","Increase "+(label||initial.Label),()=>request(c(id),"adjust","",1)));
  updaters.push(()=>{value.textContent=display(c(id)||initial);for(const b of input.querySelectorAll("button"))b.disabled=!!pending||!c(id)?.Available;});
 } else {
  input=el("span","value");
 }
 if(initial.Kind==="text"){
  const editor=el("div","text-edit"),apply=button(id==="streamdeck.add"?"Add":"Apply",()=>{
   request(c(id),"set",input.value,0,input.editRevision??c(id)?.Revision);input.blur();
  },"small");
  editor.append(input,apply);row.append(editor);
  const enable=()=>{apply.disabled=!!pending||!c(id)?.Available||input.value===(c(id)?.Value||"");};
  input.addEventListener("input",enable);updaters.push(enable);
 }else row.append(input);
 parent.append(row);
 widgets.push({id,row,note,input,initial});
}
function mixer(id,parent,title) {
 if(!c(id))return;
 const node=el("div","strip"), value=el("div","gain-readout"),bar=el("div","level"),signal=el("div","level-signal"),text=el("div","meter-label"),actions=el("div","gain-buttons");
 const device=el("div","meter-label");
 node.append(el("h3","",title),device,value);bar.append(signal);node.append(bar,text,actions);
 actions.append(iconButton("minus",title+" down",()=>request(c(id),"adjust","",-1)),button("0 dB",()=>press(id)),iconButton("plus",title+" up",()=>request(c(id),"adjust","",1)));
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
   actions.children[1].className=muted?"toggle critical":"toggle";
   actions.children[1].setAttribute("aria-pressed",String(muted));
  }
  for(const b of actions.children)b.disabled=!!pending||!control?.Available;
 });
}
function buildAudio() {
 if(![...controls.keys()].some(id=>id.startsWith("audio."))){empty(root,"Audio is disabled. Enable it in Plugins.");return;}
 const grid=el("div","grid");root.append(grid);
 const live=panel("Live controls",grid,"wide"), top=el("div","quick-controls"), strips=el("div","strip-grid");
 live.append(top);control("audio.mic-stack",top,"Mic stack");control("audio.mic-mute",top,"Mic mute");control("audio.speaker-mute",top,"Playback mute");
 live.append(strips);
 mixer("audio.gain-mic",strips,"MICROPHONE");mixer("audio.gain-playback",strips,"PLAYBACK");control("audio.interface",live,"Interface");
 const used=new Set(["audio.health","audio.mic-stack","audio.mic-mute","audio.speaker-mute","audio.gain-mic","audio.gain-playback","audio.interface","audio.playback-device"]);
 for(const [title,prefix] of [["Normal","audio.normal-"],["VR","audio.vr-profile-"]]){
  if(!c(prefix+"source"))continue;
  const card=panel(title,grid), note=el("p","section-note");card.append(note);
  updaters.push(()=>{note.textContent=c(prefix+"source")?.Subdued?"VR override · Normal remains editable":"";note.hidden=!note.textContent;});
  for(const [key,label] of [["source","Microphone"],["mode","Processing"],["monitor","Monitor"],["output","Playback"]]){control(prefix+key,card,label);used.add(prefix+key);}
 }
 const recording=[...controls.values()].filter(v=>v.Group==="Recording"&&!v.SurfaceOnly);
 if(recording.length){const card=panel("Recording",grid);for(const item of recording){control(item.ID,card);used.add(item.ID);}}
 const other=[...controls.values()].filter(v=>v.ID.startsWith("audio.")&&!v.SurfaceOnly&&!used.has(v.ID));
 if(other.length){const card=panel("Routing & recovery",grid);for(const item of other)control(item.ID,card);}
}
function buildSoundboard(){
 const clips=[...controls.values()].filter(v=>v.ID.startsWith("soundboard.clip-"));
 const toolbar=el("div","toolbar"),search=el("input");search.type="search";search.placeholder="Search clips";search.setAttribute("aria-label","Search clips");toolbar.append(icon("search","size-5 shrink-0 text-muted"),search);
 command("soundboard.stop","Stop",toolbar);root.append(toolbar);
 if(c("soundboard.volume")){const card=panel("Playback",root);control("soundboard.status",card,"Library");mixer("soundboard.volume",card,"VOLUME");card.style.marginBottom="18px";}
 if(!clips.length){empty(root,"No clips available. Check the soundboard plugin and library folder.");return;}
 const grid=el("div","clip-grid");root.append(grid);
 for(const initial of clips){
  const b=button("",()=>press(initial.ID),"clip"),art=el("div","clip-art"),name=el("strong","",initial.Label),status=el("small");
  b.append(art,name,status);b.title=initial.Label;grid.append(b);
  updaters.push(()=>{
   const item=c(initial.ID);if(!item)return;
   b.disabled=!!pending||!item.Available;b.className="clip"+(item.Value==="Playing"?" playing":item.Value==="Wait"?" wait":tone(item)==="critical"?" error":"");
   status.textContent=item.Status||display(item);status.className=tone(item);
   if(art.dataset.art!==item.Artwork){art.dataset.art=item.Artwork;art.replaceChildren();if(item.Artwork){const image=el("img");image.src="data:image/png;base64,"+item.Artwork;image.alt="";art.append(image);}else art.append(icon("play","size-9"));}
   b.hidden=!item.Label.toLocaleLowerCase().includes(search.value.toLocaleLowerCase());
  });
 }
 search.oninput=update;
}
function lightsSetup(parent){
 const card=el("section","panel setup"),title=el("h2"),text=el("p"),actions=el("div","actions");
 const pair=button("Pair",()=>press("hue.pair"),"primary");actions.append(pair);card.append(title,text,actions);parent.append(card);
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
  card.hidden=!heading;title.textContent=heading;title.className=accent;text.textContent=body;
  pair.hidden=!canPair;pair.textContent=value==="Error"?"Pair again":"Pair";pair.disabled=!!pending||!p?.Available;
 });
}
function lightDial(id,parent,title,pressLabel,step,min,max){
 if(!c(id))return;
 const node=el("div","strip light-strip"),value=el("div","gain-readout"),slider=el("input"),note=el("div","meter-label"),actions=el("div","gain-buttons");
 slider.type="range";slider.min=min;slider.max=max;slider.step="any";slider.setAttribute("aria-label",title);
 node.append(el("h3","",title),value,slider,note,actions);parent.append(node);
 actions.append(iconButton("minus",title+" down",()=>request(c(id),"adjust","",-1)),button(pressLabel,()=>press(id)),iconButton("plus",title+" up",()=>request(c(id),"adjust","",1)));
 // The plugin owns absolute values; the slider sends the equivalent relative ticks on release.
 slider.onchange=()=>{const current=numericValue(c(id)?.Value);if(current===null)return;const ticks=Math.round((Number(slider.value)-current)/step);if(ticks)request(c(id),"adjust","",ticks);};
 updaters.push(()=>{
  const control=c(id),number=numericValue(control?.Value);
  value.textContent=control?display(control):"Unavailable";value.classList.toggle("attention",!!control?.Subdued);
  if(document.activeElement!==slider&&number!==null)slider.value=number;
  slider.disabled=!!pending||!control?.Available||number===null;
  note.textContent=control?.Status||"";note.className="meter-label "+tone(control);
  for(const b of actions.children)b.disabled=!!pending||!control?.Available;
 });
}
function segmented(id,title,parent){
 const initial=c(id);if(!initial)return;
 const row=el("div","segmented-row"),group=el("div","segmented");group.setAttribute("role","group");group.setAttribute("aria-label",title);
 row.append(el("h3","",title),group);parent.append(row);
 updaters.push(()=>{row.hidden=!!c(id)?.Hidden;});
 for(const option of initial.Options||[]){
  const b=button(initial.OptionLabels?.[option]??option,()=>request(c(id),"set",option));group.append(b);
  updaters.push(()=>{const item=c(id);b.setAttribute("aria-pressed",String(item?.Value===option));b.disabled=!!pending||!item?.Available;});
 }
}
function sceneCard(id,parent){
 const b=button("",()=>press(id),"clip scene"),art=el("div","clip-art"),name=el("strong"),status=el("small");
 b.append(art,name,status);parent.append(b);
 updaters.push(()=>{
  const item=c(id);if(!item)return;
  name.textContent=item.ShortLabel||item.Label;b.title=item.Label;
  b.disabled=!!pending||!item.Available;
  b.className="clip scene"+(item.Value==="Active"?" playing":item.Status==="Pending"?" wait":tone(item)==="critical"?" error":"");
  status.textContent=item.Status==="Pending"?"Wait":item.Status||display(item);status.className=tone(item);
  if(art.dataset.art!==item.Artwork){art.dataset.art=item.Artwork||"";art.replaceChildren();if(item.Artwork){const image=el("img");image.src="data:image/png;base64,"+item.Artwork;image.alt="";art.append(image);}else art.append(icon("lightbulb","size-7"));}
 });
}
function buildLights(){
 if(!(state.Plugins||{}).hue){empty(root,"Hue is not part of this build.");return;}
 if(!state.Enabled?.hue){
  const card=panel("Hue is off",root,"setup");card.append(el("p","","Control Hue scenes, room brightness and Hue Sync."));
  const actions=el("div","actions");actions.append(button("Enable Hue",()=>send({Kind:"selection",Plugin:"hue",Enable:true}),"primary"));card.append(actions);return;
 }
 if(!c("hue.status")){empty(root,"Hue: "+(state.Plugins.hue||"Starting"));return;}
 lightsSetup(root);
 const grid=el("div","lights-grid");root.append(grid);
 const room=panel("Room",grid,"room-card"),roomNote=el("p","section-note");room.append(roomNote);
 control("hue.group",room.querySelector(".panel-head"),"Room");
 const dials=el("div","strip-grid single");room.append(dials);
 lightDial("hue.brightness",dials,"BRIGHTNESS","On/Off",2,1,100);
 updaters.push(()=>{roomNote.textContent=c("hue.sync")?.Value==="On"?"Hue Sync is driving the lights · Brightness adjusts the sync":"";roomNote.hidden=!roomNote.textContent;});
 const groupControl=c("hue.group"),roomName=(groupControl?.OptionLabels?.[groupControl.Value]||"").replace(/ \(zone\)$/,"");
 const scenes=[...controls.values()].filter(v=>v.ID.startsWith("hue.scene-"));
 const own=scenes.filter(v=>roomName&&sceneRoom(v)===roomName).sort((a,b)=>a.ShortLabel.localeCompare(b.ShortLabel));
 const sceneGrid=el("div","clip-grid scene-grid");room.append(sceneGrid);
 if(own.length)for(const scene of own)sceneCard(scene.ID,sceneGrid);
 else sceneGrid.append(el("p","muted",roomName?"No scenes in this room.":"Choose a room to see its scenes."));
 const others=scenes.filter(v=>!own.includes(v));
 if(others.length){
  const details=el("details"),summary=el("summary","","Other rooms");details.append(summary);room.append(details);
  for(const name of [...new Set(others.map(sceneRoom))].sort()){
   details.append(el("h3","room-heading",name||"Other"));const g=el("div","clip-grid scene-grid");details.append(g);
   for(const scene of others.filter(v=>sceneRoom(v)===name).sort((a,b)=>a.ShortLabel.localeCompare(b.ShortLabel)))sceneCard(scene.ID,g);
  }
 }
 const sync=panel("Hue Sync",grid),syncState=el("p","sync-state"),toggle=button("",()=>press("hue.sync"),"toggle sync-toggle"),help=el("p","muted");
 sync.append(syncState,toggle);segmented("hue.sync-mode","Mode",sync);segmented("hue.sync-intensity","Intensity",sync);sync.append(help);
 updaters.push(()=>{
  const status=c("hue.sync-status"),item=c("hue.sync"),on=item?.Value==="On";
  syncState.textContent=status?.Value==="N/A"?"Not connected":status?.Value||"";syncState.className="sync-state "+(on?"active":tone(status));
  toggle.textContent=item?.Status==="Pending"?"Wait":on?"Stop sync":"Start sync";toggle.setAttribute("aria-pressed",String(on));
  toggle.className="toggle sync-toggle "+(item?.Status&&item.Status!=="Pending"?"critical":"");toggle.disabled=!!pending||!item?.Available;
  help.textContent=status?.Value==="N/A"?"Open Hue Sync and turn on Settings → Third-party control.":status?.Value==="No bridge"?"Hue Sync has no bridge connection.":item?.Status&&item.Status!=="Pending"?item.Status:"";
  help.className=item?.Status&&item.Status!=="Pending"?"critical":"muted";help.hidden=!help.textContent;
 });
}
function buildDeck(){
 const preview=c("streamdeck.preview");
 if(!preview?.ViewData){empty(root,"Stream Deck is disabled or its editor is not ready.");return;}
 const toolbar=el("div","toolbar");
 const selectors=el("div","grid");selectors.classList.add("wide");selectors.style.flex="1";toolbar.append(selectors);
 control("streamdeck.profile",selectors,"Device");control("streamdeck.page",selectors,"Page");root.append(toolbar);
 const workspace=el("div","deck-workspace"),left=el("div"),inspector=el("section","panel inspector");
 workspace.append(left,inspector);root.append(workspace);
 const frame=el("div","deck-preview"),keys=el("div","deck-keys"),dials=el("div","deck-dials");frame.append(keys,dials);left.append(frame);
 keys.setAttribute("aria-label","Deck keys");dials.setAttribute("aria-label","Deck dials");
 function slotButton(index,dial){
  const b=button("",()=>request(c("streamdeck.slot"),"set",(dial?"Dial ":"Key ")+(dial?index-36+1:index+1)),dial?"dial":"deck-key");
  const number=el("small","",String(dial?index-36+1:index)),icon=el("span","slot-icon"),name=el("span","slot-name"),source=el("span","slot-source");
  if(!dial)b.append(number,icon);b.append(name,source);(dial?dials:keys).append(b);
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
   b.className=(dial?"dial":"deck-key")+(slot.Control?"":" empty-slot");
   name.textContent=item?.ShortLabel||item?.Label||slot.Label||"";
   source.textContent=slot.Source||"";
   const art=item?.Artwork||"";
   if(icon.dataset.art!==art){icon.dataset.art=art;icon.replaceChildren();if(art){const image=el("img");image.src="data:image/png;base64,"+art;image.alt="";icon.append(image);}else if(slot.Control)icon.append(symbolIcon(slot.Control));}
  });
 }
 for(let i=0;i<36;i++)slotButton(i,false);
 for(let i=36;i<41;i++)slotButton(i,true);
 const reserved=el("div","dial");reserved.append(el("span","","Pages"),el("small","muted","Reserved"));dials.append(reserved);
 const tools=panel("Page",left,"page-tools");
 control("streamdeck.name",tools,"Name");
 const commands=el("div","actions");tools.append(commands);
 command("streamdeck.earlier","Earlier",commands,"","chevron-left");command("streamdeck.later","Later",commands,"","chevron-right");command("streamdeck.home","Make Home",commands,"","house");
 const more=el("details"),summary=el("summary","","Page options");more.append(summary);tools.append(more);
 control("streamdeck.add",more,"New page");
 control("streamdeck.auto-controls",more,"Automatic prefix");
 const deletion=el("div","actions");more.append(deletion);
 const del=command("streamdeck.delete","Delete page",deletion,"danger");
 if(del)del.onclick=()=>{if(window.confirm("Delete this page from the draft? Save applies the deletion."))press("streamdeck.delete");};
 const selected=preview.ViewData.Selected,dial=selected>=36;
 inspector.append(el("div","eyebrow",dial?"DIAL "+(selected-35):"KEY "+selected),el("h2","","Binding"));
 const ownership=el("p","section-note");inspector.append(ownership);
 updaters.push(()=>{
  const v=c("streamdeck.preview")?.ViewData;if(!v)return;
  const slot=dial?v.Dials[selected-36]:v.Keys[selected];
  ownership.className=slot.Source?"section-note":"muted";
  ownership.textContent=slot.Source==="Auto"?"Auto-filled · assigning a binding pins this position":slot.Source==="Shared"?"Shared across pages":"Page binding";
 });
 control("streamdeck.shared",inspector,"Edit shared binding");
 const search=el("input");search.type="search";search.placeholder="Find an action";search.setAttribute("aria-label","Find a binding");
 const select=el("select");select.size=9;select.setAttribute("aria-label","Binding");
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
 const bar=el("div","savebar"),status=el("p"),actions=el("div","actions");bar.append(status,actions);root.append(bar);
 const discard=command("streamdeck.cancel","Discard",actions),save=command("streamdeck.save","Save layout",actions,"primary");
 updaters.push(()=>{
  const v=c("streamdeck.preview")?.ViewData;status.textContent=c("streamdeck.status")?.Value||"";
  status.className=v?.Dirty?"attention":"muted";
  if(discard)discard.disabled=!!pending||!v?.Dirty;if(save)save.disabled=!!pending||!v?.Dirty;
 });
}
// Deck key previews map a control's deck icon to the closest Lucide icon.
const deckIcons={"mic-mute":"mic","record-mic":"mic","vr-mic":"mic","speaker-mute":"volume-2","vr-playback":"volume-2","record-computer":"monitor","monitor":"headphones","mic-stack":"power","mode-direct":"audio-lines","mode-element":"audio-lines","tap-pre":"audio-lines","tap-post":"audio-lines","record-toggle":"circle-dot","record-start":"circle-dot","record-stop":"square","soundboard-play":"play","soundboard-stop":"square","media-prev":"skip-back","media-next":"skip-forward","media-play":"play","open-controls":"sliders-horizontal","defaults":"sliders-horizontal","engine-restart":"refresh-cw","hue-scene":"lightbulb","hue-brightness":"sun","hue-pair":"link","huesync-sync":"monitor","huesync-mode":"layers","huesync-intensity":"waves"};
function symbolIcon(id){
 const mapped=deckIcons[c(id)?.Icon];
 const name=mapped||(/mute/.test(id)?"volume-2":/mic|source|mode/.test(id)?"mic":/play|clip/.test(id)?"play":/stop/.test(id)?"square":/record/.test(id)?"circle-dot":/prev/.test(id)?"skip-back":/next/.test(id)?"skip-forward":/scene/.test(id)?"lightbulb":"circle");
 return icon(name,"size-6");
}
function buildPlugins(){
 for(const id of Object.keys(state.Plugins||{}).sort()){
  const card=el("section","panel plugin-card"),text=el("div"),status=el("p"),toggle=button("",()=>send({Kind:"selection",Plugin:id,Enable:!state.Enabled[id]}),"toggle");
  text.append(el("h2","",id),status);card.append(text,toggle);root.append(card);
  updaters.push(()=>{status.textContent=state.Plugins[id];status.className=state.Plugins[id]==="Running"?"active":state.Plugins[id]==="Disabled"?"muted":"critical";toggle.textContent=state.Enabled[id]?"Disable":"Enable";toggle.disabled=!!pending;});
 }
}
async function copyText(text,feedback){
 try{
  if(navigator.clipboard?.writeText)await navigator.clipboard.writeText(text);
  else throw new Error("Clipboard unavailable");
 }catch{
  // WebView clipboard permissions vary; fall back to a temporary selection.
  const area=el("textarea");area.value=text;area.setAttribute("readonly","");area.style.position="fixed";area.style.opacity="0";document.body.append(area);area.select();
  const ok=document.execCommand("copy");area.remove();
  if(!ok){showError("Copy failed: clipboard unavailable");return;}
 }
 feedback.textContent="Copied";setTimeout(()=>{feedback.textContent="";},1500);
}
function reportCard(id,parent){
 const card=el("section","app-card"),head=el("div","app-head"),name=el("h3","",c(id).Label),badge=el("span","badge"),copy=withIcon(button("Copy details",()=>copyText(connectionText(c(id)),copied),"small"),"copy"),copied=el("small","copied");
 const endpoint=el("code","endpoint"),times=el("p","app-times"),error=el("p","app-error"),list=el("dl","app-details");
 head.append(name,badge);card.append(head,endpoint,times,error,list);
 const foot=el("div","actions");foot.append(copy,copied);card.append(foot);parent.append(card);
 updaters.push(()=>{
  const item=c(id);if(!item)return;const conn=item.Connection||{},now=Date.now(),accent=connectionTone(conn);
  badge.textContent=item.Value||conn.State||"";badge.className="badge "+accent;card.dataset.tone=accent;
  endpoint.textContent=conn.Endpoint||"";endpoint.hidden=!conn.Endpoint;
  const parts=[];
  if(timeValue(conn.Since)!==null)parts.push("Since "+relativeTime(conn.Since,now));
  if(timeValue(conn.LastActivity)!==null)parts.push("Last activity "+relativeTime(conn.LastActivity,now)+(stale(conn,now)?" · stale":""));
  times.textContent=parts.join(" · ");times.hidden=!parts.length;times.className="app-times"+(stale(conn,now)?" attention":"");
  const current=["error","attention","connecting"].includes(conn.State)||(conn.State==="disconnected"&&accent==="critical");
  error.textContent=conn.LastError?(current?"":"Last error: ")+conn.LastError+(timeValue(conn.LastErrorAt)!==null?" · "+relativeTime(conn.LastErrorAt,now):""):"";
  error.hidden=!conn.LastError;error.className="app-error "+(current?"critical":"muted");
  const rows=(conn.Details||[]).filter(d=>d.Label!=="Required");
  const key=JSON.stringify(rows);
  if(list.dataset.key!==key){list.dataset.key=key;list.replaceChildren();for(const d of rows)list.append(el("dt","",d.Label),el("dd","",d.Value));}
  list.hidden=!rows.length;
 });
}
function buildApps(){
 const reports=[...controls.values()].filter(v=>v.Kind==="connection").sort((a,b)=>(a.Group||"").localeCompare(b.Group||"")||a.Label.localeCompare(b.Label));
 const bar=el("div","toolbar apps-summary"),counts=el("div","summary-counts"),copied=el("small","copied");
 bar.append(counts,withIcon(button("Copy all",()=>copyText(reports.map(r=>connectionText(c(r.ID)||r)).join("\n\n"),copied),"small"),"copy"),copied);root.append(bar);
 updaters.push(()=>{
  const tally={active:0,attention:0,critical:0,other:0};
  for(const r of reports){const t=connectionTone(c(r.ID)?.Connection);tally[t in tally?t:"other"]++;}
  counts.replaceChildren(...[["active","OK"],["attention","Attention"],["critical","Problems"],["other","Idle"]].map(([k,label])=>el("span","badge "+(k==="other"?"":k),tally[k]+" "+label)));
 });
 if(!reports.length)empty(root,"No third-party apps are reporting. Enable plugins in Plugins.");
 for(const group of [...new Set(reports.map(r=>r.Group||"Other"))]){
  const card=panel(group,root,"apps-group"),grid=el("div","app-grid");card.append(grid);
  for(const r of reports.filter(v=>(v.Group||"Other")===group))reportCard(r.ID,grid);
 }
 const idle=Object.entries(state.Plugins||{}).filter(([,status])=>status!=="Running").sort();
 if(idle.length){
  const card=panel("Not monitored",root,"apps-group"),list=el("ul","not-monitored");card.append(list);
  for(const [id,status] of idle)list.append(el("li","",id+" — "+status));
 }
}
function buildDiagnostics(){
 const toolbar=el("div","toolbar");toolbar.append(withIcon(button("Retry plugins",()=>send({Kind:"retry"})),"refresh-cw"));root.append(toolbar);
 const card=panel("Audio engine",root);control("audio.health",card);control("audio.interface",card);control("audio.asio-unavailable",card);
 const details=el("section","panel diagnostic");details.style.marginTop="18px";root.append(details);
 updaters.push(()=>{
  details.replaceChildren(el("h2","","Current notices"));
  const notices=[...controls.values()].filter(v=>v.Status);
  if(!notices.length)details.append(el("p","","No control notices."));
  for(const item of notices){details.append(el("h3","",item.Label),el("p",tone(item),item.Status));}
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
  w.row.classList.toggle("subdued",!!item.Subdued);w.note.textContent=item.Subdued&&item.Status==="VR override"?"":item.Status||"";w.note.className=tone(item);
  const input=w.input;
  if(item.Kind==="toggle"){input.textContent=display(item);input.setAttribute("aria-pressed",String(item.Value==="On"));input.className="toggle "+tone(item);}
  else if(item.Kind==="selection"||item.Kind==="text"){if(document.activeElement!==input)input.value=item.Value||"";}
  else if(item.Kind==="status"){input.textContent=display(item);input.className="value "+tone(item);}
  if(["toggle","selection","text","command"].includes(item.Kind))input.disabled=!!pending||!item.Available;
 }
 for(const fn of updaters)fn();
 const health=c("audio.health");$("#summary").replaceChildren();
 if(pending)$("#summary").append(el("span","badge attention","Sending"));
 if(health)$("#summary").append(el("span","badge "+tone(health),health.Value||"Unknown"));
 if(c("audio.normal-source")?.Subdued)$("#summary").append(el("span","badge active","VR active"));
 $("#connection").textContent=connected?"Connected":"Disconnected";$("#connection").className="badge "+(connected?"active":"critical");
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

