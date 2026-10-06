import {controlsByID,compatible,tone,meterValue,display,gridMove} from "./model.mjs";

const $=s=>document.querySelector(s);
const root=$("#content");
let state={Controls:[],Plugins:{},Enabled:{}}, controls=new Map(), screen="audio", signature="", pending=null;
let localError="", dismissedNotice="", connected=false;
const widgets=[], updaters=[];
const titles={audio:["AUDIO","Audio"],soundboard:["LIBRARY","Soundboard"],deck:["CONTROL SURFACE","Stream Deck"],plugins:["SYSTEM","Plugins"],diagnostics:["SYSTEM","Diagnostics"]};
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
function command(id,label,parent,className="") {
 if(!c(id)) return null;
 const b=button(label,()=>press(id),className);parent.append(b);
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
  input.append(button("−",()=>request(c(id),"adjust","",-1)),value,button("+",()=>request(c(id),"adjust","",1)));
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
 node.append(el("h3","",title),value);bar.append(signal);node.append(bar,text,actions);
 actions.append(button("−",()=>request(c(id),"adjust","",-1)),button("0 dB",()=>press(id)),button("+",()=>request(c(id),"adjust","",1)));
 // Only soundboard's press means reset; audio gain press retains its existing mute behavior.
 if(id!=="soundboard.volume"){actions.children[1].textContent="Mute";actions.children[1].title="Toggle mute";}
 parent.append(node);
 updaters.push(()=>{
  const control=c(id);value.textContent=control ? display(control) : "Unavailable";
  const db=meterValue(control?.Meter);
  signal.style.clipPath="inset(0 "+(db===null?100:-(db/60)*100)+"% 0 0)";
  text.textContent=db===null?"LEVEL N/A":db.toFixed(1)+" dBFS";
  const muteID={"audio.gain-mic":"audio.mic-mute","audio.gain-A1":"audio.a1-mute","audio.gain-A2":"audio.a2-mute"}[id];
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
 mixer("audio.gain-mic",strips,"MICROPHONE");mixer("audio.gain-A1",strips,"A1 OUTPUT");mixer("audio.gain-A2",strips,"A2 OUTPUT");
 const used=new Set(["audio.health","audio.mic-stack","audio.mic-mute","audio.speaker-mute","audio.gain-mic","audio.gain-A1","audio.gain-A2"]);
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
 const toolbar=el("div","toolbar"),search=el("input");search.type="search";search.placeholder="Search clips";search.setAttribute("aria-label","Search clips");toolbar.append(search);
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
   if(art.dataset.art!==item.Artwork){art.dataset.art=item.Artwork;art.replaceChildren();if(item.Artwork){const image=el("img");image.src="data:image/png;base64,"+item.Artwork;image.alt="";art.append(image);}else art.textContent="▷";}
   b.hidden=!item.Label.toLocaleLowerCase().includes(search.value.toLocaleLowerCase());
  });
 }
 search.oninput=update;
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
   if(icon.dataset.art!==art){icon.dataset.art=art;icon.replaceChildren();if(art){const image=el("img");image.src="data:image/png;base64,"+art;image.alt="";icon.append(image);}else icon.textContent=slot.Control?symbol(slot.Control):"";}
  });
 }
 for(let i=0;i<36;i++)slotButton(i,false);
 for(let i=36;i<41;i++)slotButton(i,true);
 const reserved=el("div","dial");reserved.append(el("span","","Pages"),el("small","muted","Reserved"));dials.append(reserved);
 const tools=panel("Page",left,"page-tools");
 control("streamdeck.name",tools,"Name");
 const commands=el("div","actions");tools.append(commands);
 command("streamdeck.earlier","← Earlier",commands);command("streamdeck.later","Later →",commands);command("streamdeck.home","Make Home",commands);
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
function symbol(id){if(/mic|source|mode/.test(id))return "♩";if(/mute/.test(id))return "◌";if(/play|clip/.test(id))return "▷";if(/stop/.test(id))return "■";if(/record/.test(id))return "●";if(/prev/.test(id))return "‹";if(/next/.test(id))return "›";return "◇";}
function buildPlugins(){
 for(const id of Object.keys(state.Plugins||{}).sort()){
  const card=el("section","panel plugin-card"),text=el("div"),status=el("p"),toggle=button("",()=>send({Kind:"selection",Plugin:id,Enable:!state.Enabled[id]}),"toggle");
  text.append(el("h2","",id),status);card.append(text,toggle);root.append(card);
  updaters.push(()=>{status.textContent=state.Plugins[id];status.className=state.Plugins[id]==="Running"?"active":state.Plugins[id]==="Disabled"?"muted":"critical";toggle.textContent=state.Enabled[id]?"Disable":"Enable";toggle.disabled=!!pending;});
 }
 commandFallback(root);
}
function commandFallback(parent){
 const other=[...controls.values()].filter(v=>!v.SurfaceOnly&&!/^(audio|soundboard|streamdeck|core)\./.test(v.ID));
 for(const group of [...new Set(other.map(v=>v.Group))]){const card=panel(group||"Controls",parent);card.style.marginTop="18px";for(const item of other.filter(v=>v.Group===group))control(item.ID,card);}
}
function buildDiagnostics(){
 const toolbar=el("div","toolbar");toolbar.append(button("Retry plugins",()=>send({Kind:"retry"})));root.append(toolbar);
 const card=panel("Audio engine",root);control("audio.health",card);
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
 return JSON.stringify([screen,list,Object.keys(state.Plugins||{}),screen==="deck"?[c("streamdeck.preview")?.ViewData?.Selected,c("streamdeck.page")?.Value,c("streamdeck.profile")?.Value,c("streamdeck.shared")?.Value]:null]);
}
function build(){
 widgets.length=0;updaters.length=0;root.replaceChildren();
 for(const b of document.querySelectorAll("[data-screen]")){if(b.dataset.screen===screen)b.setAttribute("aria-current","page");else b.removeAttribute("aria-current");}
 $("#eyebrow").textContent=titles[screen][0];$("#title").textContent=titles[screen][1];
 ({audio:buildAudio,soundboard:buildSoundboard,deck:buildDeck,plugins:buildPlugins,diagnostics:buildDiagnostics})[screen]();
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
for(const b of document.querySelectorAll("[data-screen]")){
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

