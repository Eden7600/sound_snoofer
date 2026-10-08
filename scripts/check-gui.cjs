// Run with playwright available on NODE_PATH or in node_modules; uses installed Chrome.
const { chromium }=require("playwright");
// The GUI stylesheet is generated; build it so checks exercise the real styling.
require("node:child_process").execSync("npm run --silent css",{cwd:require("node:path").resolve(__dirname,".."),stdio:"inherit"});
const fs=require("node:fs"),path=require("node:path"),http=require("node:http"),assert=require("node:assert/strict");
const root=path.resolve(__dirname,"..");
const controls=[];
function add(ID,Label,Kind,Value,extra={}){controls.push({ID,Label,Kind,Value,Available:true,Revision:1,Operations:Kind==="selection"||Kind==="text"?["set"]:Kind==="numeric"?["adjust","press"]:Kind==="status"?[]:["press"],...extra});}
add("audio.health","Audio health","status","Healthy");
add("audio.interface","Interface","status","Universal Audio Volt");
add("audio.mic-device","Microphone","status","Lavalier",{SurfaceOnly:true});
add("audio.playback-device","Playback device","status","Arena",{SurfaceOnly:true});
for(const [id,label] of [["mic-stack","Mic stack"],["mic-mute","Mic mute"],["speaker-mute","Playback mute"]])add("audio."+id,label,"toggle",id==="mic-stack"?"On":"Off");
for(const id of ["mic","playback"])add("audio.gain-"+id,id+" gain","numeric","-6.0 dB",{Meter:{Present:true,Known:true,DB:-15,At:new Date().toISOString()}});
for(const prefix of ["normal-","vr-profile-"]){
 for(const [id,label,value,opts] of [["source","Microphone","lav",["auto","lav","webcam"]],["mode","Processing","direct",["direct","element"]],["monitor","Monitor","off",["off","pre","post"]],["output","Playback","Arena",["Arena","Headset"]]])
 add("audio."+prefix+id,label,"selection",value,{Options:opts,OptionLabels:{lav:"Lavalier · Volt",auto:"Automatic"},Subdued:prefix==="normal-",Status:prefix==="normal-"?"VR override":""});
}
add("audio.record-mic","Record microphone","toggle","On",{Group:"Recording"});
add("audio.record-toggle","Record","command","Stopped",{Group:"Transport",SurfaceOnly:true,Icon:"record-toggle"});
add("audio.tape-play","Play recording","command","Playing",{Group:"Transport",SurfaceOnly:true,Icon:"tape-pause"});
add("audio.tape-stop","Stop playback","command","",{Group:"Transport",SurfaceOnly:true,Icon:"tape-stop"});
add("audio.tape-rew","Rewind","command","",{Group:"Transport",SurfaceOnly:true,Icon:"tape-rew",Available:false});
add("audio.tape-ff","Fast-forward","command","",{Group:"Transport",SurfaceOnly:true,Icon:"tape-ff"});
add("audio.auto-recover","Automatic recovery","toggle","On",{Group:"Shared audio"});
add("audio.pause-devices","Disable device manipulation","toggle","Off",{Group:"Shared audio"});
add("audio.pause-sends","Disable send manipulation","toggle","On",{Group:"Shared audio",Status:"3 held"});
add("audio.engine-restart","Restart audio engine","command","",{Group:"Bindings",SurfaceOnly:true});
add("aec.mode","Echo cancellation","selection","auto",{Options:["auto","on","off"],OptionLabels:{auto:"Auto",on:"On",off:"Off"},Icon:"echo"});
add("aec.engine","Echo engine","selection","aec3",{Options:["aec3","localvqe-aec","localvqe-voice","localvqe-full"],OptionLabels:{aec3:"WebRTC AEC3","localvqe-aec":"LocalVQE echo-only","localvqe-voice":"LocalVQE voice cleanup","localvqe-full":"LocalVQE full-band"},Icon:"echo"});
add("aec.strength","Echo strength","selection","strong",{Options:["strong","balanced","gentle"],OptionLabels:{strong:"Strong",balanced:"Balanced",gentle:"Gentle"},Icon:"echo"});
add("aec.timing","Neural timing","status","2.5 / 1.2 ms",{Status:"Peak worker / queue · gaps 3 · underruns 0"});
add("aec.retry","Retry echo cancellation","command","",{Available:false});
add("aec.status","Echo cancellation status","status","Active",{Status:"−32 dB echo · 54 ms",Icon:"echo"});
add("soundboard.volume","Volume","numeric","0.0 dB");
add("soundboard.stop","Stop","command","");
add("soundboard.status","Soundboard","status","Ready");
for(const name of ["fah","sadge","instinct","airhorn","ping"])add("soundboard.clip-"+name,name,"command","Ready");
// sadge has animated artwork: two frames served by Desktop.Animation.
const dot="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==";
for(const [id,label,kind,value,options]of [["profile","Device","selection","Default",["Default"]],["page","Page","selection","home",["home","soundboard"]],["slot","Position","selection","Key 1",Array.from({length:36},(_,i)=>"Key "+(i+1))],["shared","Shared","toggle","Off"],["binding","Binding","selection","audio.mic-stack",["","audio.mic-stack","audio.mic-mute"]],["name","Name","text","Home"],["add","New page","text",""],["region-add","Add region","text",""],["region-source","Region source","text",""],["region-remove","Remove region","text",""],["preview","Preview","status",""],["status","Layout status","status","Connected"]])add("streamdeck."+id,label,kind,value,{Options:options});
for(const id of ["save","cancel","delete","earlier","later","home"])add("streamdeck."+id,id,"command","");
add("hue.status","Hue","status","Not paired",{ViewData:{bridge:"172.16.102.3"}});
add("hue.pair","Pair Hue bridge","command","Ready",{ShortLabel:"Pair"});
add("hue.group","Hue room","selection","room-1",{Options:["zone-1","room-1"],OptionLabels:{"zone-1":"Desk (zone)","room-1":"Studio"}});
add("hue.brightness","Hue brightness","numeric","62%",{ShortLabel:"Brightness"});
add("hue.motion","Hue motion sensors","toggle","On",{ShortLabel:"Motion",Icon:"hue-motion"});
add("hue.rooms","Hue rooms","text","",{ViewData:[{ID:"room-1",Name:"Studio",Kind:"room",Chosen:true},{ID:"zone-1",Name:"Desk",Kind:"zone",Chosen:true}]});
const sceneArt=fs.readFileSync(path.join(root,"docs/design/hue-scene-art.png")).toString("base64");
controls.find(c=>c.ID==="soundboard.clip-sadge").Artwork=sceneArt;
const animations={"soundboard.clip-sadge":[{Artwork:sceneArt,Delay:50e6},{Artwork:dot,Delay:50e6}]};
for(const [room,name,value] of [["Studio","Bright","Ready"],["Studio","Relax","Active"],["Studio","Concentrate","Ready"],["Desk","Focus","Ready"],["Kitchen","Cook","Ready"]])add("hue.scene-"+room.toLowerCase()+"-"+name.toLowerCase(),room+" "+name,"command",value,{ShortLabel:name,Group:"Hue scenes",Artwork:name==="Relax"?sceneArt:""});
add("hue.sync-status","Hue Sync","status","Ready");
add("hue.sync","Hue Sync","toggle","Off",{ShortLabel:"Sync"});
add("hue.sync-mode","Hue Sync mode","selection","video",{Options:["video","games","music"],OptionLabels:{video:"Video",games:"Games",music:"Music"},Available:false,Hidden:true});
add("hue.sync-intensity","Hue Sync intensity","selection","moderate",{Options:["subtle","moderate","high","extreme"],OptionLabels:{subtle:"Subtle",moderate:"Moderate",high:"High",extreme:"Extreme"},Available:false,Hidden:true});
for(let n=1;n<=12;n++)add("hue.room-scene-"+n,"","command","",{Available:false,Group:"Hue room scenes"});
const ago=s=>new Date(Date.now()-s*1000).toISOString();
const report=(ID,Label,Group,Value,Connection)=>controls.push({ID,Label,Group,Kind:"connection",Value,Available:true,SurfaceOnly:true,Revision:1,Operations:[],Connection});
report("hue.app-bridge","Hue Bridge","Hue","Connected",{State:"connected",Endpoint:"172.16.102.3",Since:ago(600),LastActivity:ago(1),LastError:"event stream ended: EOF",LastErrorAt:ago(700),Details:[{Label:"Bridge ID",Value:"001788fffe2490e0"},{Label:"Software",Value:"1978293000"}]});
report("hue.app-sync","Hue Sync","Hue","N/A",{State:"disconnected",Endpoint:"ws://127.0.0.1:24851/",Since:ago(90),LastError:"connection refused",LastErrorAt:ago(5),Details:[]});
report("audio.app-voicemeeter","Voicemeeter","Audio","Disconnected",{State:"disconnected",Endpoint:"C:\\Program Files (x86)\\VB\\Voicemeeter\\VoicemeeterRemote64.dll",Since:ago(30),LastActivity:ago(31),LastError:"voicemeeter disconnected",LastErrorAt:ago(30),Details:[{Label:"Required",Value:"Yes"},{Label:"Edition",Value:"Potato"},{Label:"Interval",Value:"1s"}]});
report("audio.app-callback","Audio callback monitor","Audio","Off",{State:"off",Endpoint:"snoofer-audio-monitor.dll",Since:"0001-01-01T00:00:00Z",LastActivity:"0001-01-01T00:00:00Z",Details:[]});
const view={Selected:0,Dirty:false,Home:true,Regions:[],Collections:[{ID:"hue.room-scenes",Label:"Scenes · selected room"},{ID:"soundboard.clips",Label:"Soundboard clips"}],Keys:Array.from({length:36},()=>({Control:"",Label:"",Source:""})),Dials:Array.from({length:5},()=>({Control:"",Label:"",Source:""}))};
["audio.mic-stack","audio.mic-mute","audio.speaker-mute","audio.normal-monitor","audio.normal-mode"].forEach((Control,i)=>view.Keys[i]={Control,Label:"",Source:""});
view.Keys[9]={Control:"soundboard.clip-fah",Label:"fah",Source:"Auto"};
view.Keys[31]={Control:"core.open-controls",Label:"Controls",Source:"Shared"};
["audio.gain-playback","audio.gain-mic",""].forEach((Control,i)=>view.Dials[i]={Control,Label:"",Source:""});
controls.find(c=>c.ID==="streamdeck.preview").ViewData=view;
// App audio: Discord pinned first, Chrome heard, Spotify pinned but closed; Voicemeeter hidden by default.
const appMeter={Present:true,Known:true,DB:-18,At:new Date().toISOString()};
[["discord","Discord","40%"],["chrome","Google Chrome","Mixed"],["spotify","Spotify","Closed"]].forEach(([id,label,value],n)=>add("appaudio.app-"+id,label,"numeric",value,{ShortLabel:label,Group:"App audio",Collection:"appaudio.apps",CollectionLabel:"Apps",Order:n+1,Icon:"app-audio",Operations:["press","adjust","set"],Meter:value==="Closed"?{Present:true}:appMeter}));
add("appaudio.status","App audio","status","3 apps",{ViewData:{RecentMinutes:5,Apps:[
 {ID:"appaudio.app-discord",Name:"Discord",Picked:true,Open:true,Sessions:2,Executables:["C:\\Discord\\Discord.exe"],Devices:["Voicemeeter Input"],PIDs:[11,12],Naming:[{Match:"(^|\\\\)updater\\.exe$",Programs:["updater.exe"]}]},
 {ID:"appaudio.app-chrome",Name:"Google Chrome",Open:true,Sessions:3,Executables:["C:\\Chrome\\chrome.exe"],Devices:["Voicemeeter Input"],PIDs:[21],LastHeard:new Date().toISOString()},
 {ID:"appaudio.app-spotify",Name:"Spotify",Picked:true,Open:false},
 {ID:"appaudio.app-vm",Name:"Voicemeeter",Hidden:true,Open:true,Sessions:1,Rule:"Excluded (voicemeeter*.exe)"},
 {ID:"appaudio.app-hue",Name:"Hue Sync",Hidden:true,Open:true,Sessions:1,Rule:"Excluded (huesync.exe)"},
 {ID:"appaudio.app-rule",Name:"Launcher",Hidden:true,Open:true,Sessions:1,Rule:"Hidden (launcher)"}],
 Exclude:["snoofer.exe","voicemeeter*.exe","audiodg.exe","huesync.exe"]}});
add("appaudio.edit","App audio edit","text","abc");
// Routing: one interface, playback with an ambiguous entry, a webcam and the mic priority.
const suggestion=(Name,Driver,Exact,Device,ID="")=>({Name,Driver,Exact,Device,ID});
add("audio.priorities","Device priorities","status","",{Group:"Routing",ViewData:{
 Lists:{interfaces:[{ASIOPattern:"(?i)^Universal Audio Volt$",PresencePattern:"(?i)^INPUT 1/2 \\(Volt 2\\)$",Inputs:{desk:[1],lav:[2]},Matches:["INPUT 1/2 (Volt 2)"],Drivers:["Universal Audio Volt"],InUse:true}],
  playback:[{Driver:"wdm",DeviceID:"{airpods}",DeviceName:"Headphones (AirPods)",Matches:["Headphones (AirPods Pro)"],InUse:true},{Driver:"wdm",Pattern:"(?i)steelseries.*arena",Matches:["Speakers (Arena 7)","Game (Arena 7)"]},{Driver:"asio",Pattern:"(?i)^Universal Audio Volt$",Matches:["Universal Audio Volt"]}],
  "mic-devices:webcam":[{Driver:"wdm",Pattern:"(?i)insta360.*link.*2",Matches:[]}],
  microphones:[{ID:"lav",Option:true,InUse:true},{ID:"webcam",Option:false}]},
 Suggestions:{playback:[suggestion("Speakers (Realtek Audio)","wdm","(?i)^Speakers \\(Realtek Audio\\)$","(?i)Realtek Audio","{realtek}")]},
 Drivers:[suggestion("Focusrite USB ASIO","asio","(?i)^Focusrite USB ASIO$","","{focusrite}")],Inputs:[suggestion("Analogue 1 + 2 (Focusrite USB)","wdm","(?i)^Analogue 1 \\+ 2 \\(Focusrite USB\\)$","(?i)Focusrite USB")],
 Microphones:["desk","lav","webcam","off"],Mics:[{ID:"desk",Name:"Desk"},{ID:"lav",Name:"Lavalier",InUse:true,Option:true},{ID:"webcam",Name:"Webcam",Device:true}]}});
add("audio.priority-edit","Device priority edit","text","abc",{Group:"Routing"});
// Outputs: Playback plus a Music slot; positions 2 and 3 are hidden.
controls.find(c=>c.ID==="audio.priorities").ViewData.OutputDevices=[suggestion("Speakers (Realtek Audio)","wdm","","","{realtek}"),suggestion("Speakers (Arena 7)","wdm","","")];
add("audio.playback:virtual:1","virtual:1 playback","toggle","On",{Group:"Shared audio"});
add("audio.monitor","Monitor","selection","pre",{Group:"Bindings",Options:["off","pre","post"],OptionLabels:{off:"Off",pre:"Pre",post:"Post"},Operations:["set","press"],SurfaceOnly:true});
add("audio.output-edit","Output edit","text","abc",{Group:"Routing"});
add("audio.slot-1","Music","status","In use",{Group:"Routing",Status:"Speakers (Arena 7)",ViewData:{ID:"music",Bus:"A3"}});
for(const [source,value,available] of [["virtual:1","On",true],["virtual:2","Off",false],["virtual:3","Off",false],["monitor","Off",true],["soundboard","Off",true],["tape","On",true]])add("audio.slot-1:"+source,"Music "+source,"toggle",value,{Group:"Routing",Available:available,Hidden:!available});
for(const n of [2,3]){add("audio.slot-"+n,"Output "+n,"status","",{Group:"Routing",Available:false,Hidden:true});for(const source of ["virtual:1","monitor","soundboard","tape"])add("audio.slot-"+n+":"+source,"Output "+n+" "+source,"toggle","Off",{Group:"Routing",Available:false,Hidden:true});}
// Now playing: a focused YouTube tab and a paused Windows player.
add("nowplaying.s-tab","Video A","command","Playing",{ShortLabel:"Video A",Group:"Now playing",Collection:"nowplaying.sessions",CollectionLabel:"Media sessions",Order:1,Icon:"media-play",Operations:["press","set"]});
add("nowplaying.s-spotify","Song","command","Paused",{ShortLabel:"Song",Group:"Now playing",Collection:"nowplaying.sessions",CollectionLabel:"Media sessions",Order:2,Icon:"media-play",Operations:["press","set"]});
for(const [id,label] of [["prev","Previous track"],["next","Next track"],["toggle","Play or pause"],["mute","Mute tab"]])add("nowplaying."+id,label,"command","");
add("nowplaying.dial","Now playing","numeric","1:05 / 4:45",{Operations:["adjust","press"]});
add("nowplaying.status","Now playing","status","2 sessions",{ViewData:{Windows:"Connected",Bridge:{Port:47815,Error:"",Refused:"Update the Snoofer Media extension in Firefox"},
 Browsers:[{Name:"Brave",Version:"1.0.0",Sessions:1}],
 Sessions:[{ID:"nowplaying.s-tab",Source:"Brave",App:"Brave · youtube.com",Title:"Video A",Artist:"Channel",Status:"Playing",PositionMs:65000,AtMs:Date.now(),Rate:1,DurationMs:285000,Focused:true,CanToggle:true,CanNext:true,CanSeek:true,CanMute:true},
  {ID:"nowplaying.s-spotify",Source:"windows",App:"Spotify",Title:"Song",Artist:"Band",Album:"Album",Status:"Paused",PositionMs:0,DurationMs:200000,CanToggle:true,CanNext:true,CanPrev:true,CanSeek:true}]}});
add("insta360.privacy","Camera privacy","toggle","Off",{Group:"Camera",Icon:"camera-privacy"});
add("insta360.tracking","Camera tracking","selection","group",{Group:"Camera",Options:["off","single","group"],OptionLabels:{off:"Off",single:"Single",group:"Group"}});
add("insta360.framing","Camera framing","selection","half",{Group:"Camera",Options:["head","half","full"],OptionLabels:{head:"Head",half:"Half body",full:"Full body"}});
add("insta360.reset","Reset camera position","command","",{Group:"Camera"});
add("insta360.state","Camera tracking state","status","Working",{Group:"Camera"});
add("discord.channel","Discord channel","status","General",{Group:"Discord"});
for(const [id,label,value] of [["mute","Discord mute","On"],["deafen","Discord deafen","Off"],["video","Discord camera","Off"],["screenshare","Discord screen share","Off"]])add("discord."+id,label,"toggle",value,{Group:"Discord"});
add("discord.leave","Leave Discord call","command","",{Group:"Discord"});
add("discord.connect","Connect Discord","command","",{Group:"Discord",Available:false});
const fixture={Controls:controls,Plugins:{audio:"Running",soundboard:"Running",streamdeck:"Running",vr:"Running",media:"Disabled",hue:"Running",appaudio:"Running",nowplaying:"Running",insta360:"Running",discord:"Running"},Enabled:{insta360:true,discord:true,audio:true,soundboard:true,streamdeck:true,vr:true,media:false,hue:true,appaudio:true,nowplaying:true}};
(async()=>{
 const server=http.createServer((req,res)=>{
  if(req.url==="/logo.ico"){res.setHeader("Content-Type","image/x-icon");res.end(fs.readFileSync(path.join(root,"app/tray.ico")));return;}
  const file=path.join(root,"app/web",req.url==="/"?"index.html":req.url.split("?")[0]);
  if(!file.startsWith(path.join(root,"app/web")+path.sep)){res.writeHead(403);return res.end();}
  fs.readFile(file,(err,data)=>{if(err){res.writeHead(404);res.end();return;}res.setHeader("Content-Type",file.endsWith(".css")?"text/css":/\.(js|mjs)$/.test(file)?"text/javascript":"text/html");res.end(data);});
 });
 await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
 const browser=await chromium.launch({channel:"chrome",headless:true}).catch(error=>{server.close();throw error;});
 try{
  const page=await browser.newPage({viewport:{width:1280,height:820}});
  const errors=[];page.on("pageerror",e=>errors.push(String(e)));
  await page.addInitScript(([fixture,animations])=>{
   window.fixture=fixture;window.animations=animations;window.sent=[];window.copied=[];
   Object.defineProperty(navigator,"clipboard",{value:{writeText:async text=>{window.copied.push(text);}}});
   window.go={app:{Desktop:{State:async()=>{
    // Meter readings expire after 500 ms; keep the fixture's readings live.
    const next=structuredClone(window.fixture);
    for(const c of next.Controls)if(c.Meter)c.Meter.At=new Date().toISOString();
    return next;
   },Animation:async id=>window.animations[id]||null,Send:async action=>{
    window.sent.push(action);
    const r=action.Request;if(!r)return;
    const target=window.fixture.Controls.find(c=>c.ID===r.ID);
    const view=window.fixture.Controls.find(c=>c.ID==="streamdeck.preview").ViewData;
    if(r.ID==="streamdeck.slot"){view.Selected=Number(r.Value.split(" ")[1])-1;}
    if(r.ID.startsWith("streamdeck.region-")){
     if(r.ID==="streamdeck.region-add"){const [first,last,source]=r.Value.split(","),dials=Number(first)>=36;view.Regions.push({Source:source,Label:view.Collections.find(x=>x.ID===source)?.Label||source,First:Number(first)-(dials?36:0),Last:Number(last)-(dials?36:0),Dials:dials});}
     if(r.ID==="streamdeck.region-source"){const [i,source]=r.Value.split(",");Object.assign(view.Regions[Number(i)],{Source:source,Label:view.Collections.find(x=>x.ID===source)?.Label||source});}
     if(r.ID==="streamdeck.region-remove")view.Regions.splice(Number(r.Value),1);
     view.Keys.forEach(k=>{k.Region=-1;});view.Dials.forEach(d=>{d.Region=-1;});
     view.Regions.forEach((g,n)=>{if(g.Dials){for(let d=Math.min(g.First,g.Last);d<=Math.max(g.First,g.Last);d++)view.Dials[d].Region=n;return;}const rows=[Math.floor(g.First/9),Math.floor(g.Last/9)].sort((a,b)=>a-b),cols=[g.First%9,g.Last%9].sort((a,b)=>a-b);for(let row=rows[0];row<=rows[1];row++)for(let col=cols[0];col<=cols[1];col++)view.Keys[row*9+col].Region=n;});
     view.Dirty=true;window.fixture.Controls.find(c=>c.ID==="streamdeck.preview").Revision++;
    }
    if(r.Operation==="set")target.Value=r.Value;
    target.Revision++;
   }}}};
  },[fixture,animations]);
  await page.goto("http://127.0.0.1:"+server.address().port);
  await page.getByRole("heading",{name:"Live controls"}).waitFor();
  await page.waitForFunction(()=>document.querySelector("[data-part=brandmark]").naturalWidth>0);
  assert.equal(await page.locator("[data-part=strip]").count(),2);
  assert.equal(await page.locator("section",{has:page.getByRole("heading",{name:"Live controls",exact:true})}).getByText("Engine",{exact:true}).count(),1,"Audio screen lacks the Engine health row");
  assert.equal(await page.locator("#summary").getByText("Healthy").count(),0,"health badge still in the header");
  assert.equal(await page.getByText("Plugin host").count(),0,"sidebar host footer still shown");
  assert.equal(await page.getByRole("heading",{name:"PLAYBACK",exact:true}).count(),1);
  assert.equal(await page.getByText("A1 OUTPUT",{exact:true}).count(),0);
  const strip=page.locator("[data-part=strip]").first();
  await strip.locator("[data-part=peak]").waitFor();
  assert.equal(await strip.getByText("LEVEL N/A").count(),0,"live meter shown as unavailable");
  assert.equal(await strip.getByText("-15.0 dBFS").count(),1);
  const fillWidth=await strip.locator("[data-part=position] > div").first().evaluate(e=>e.style.width);
  assert.equal(fillWidth,"75%","gain position track");
  for(const mark of ["-60","-30","-12","-3","0"])assert.equal(await strip.getByText(mark,{exact:true}).count(),1,"meter scale "+mark);
  assert.equal(await page.locator("#summary").getByText("Sends paused").count(),1,"paused sends badge");
  assert.equal(await page.locator("#summary").getByText("Devices paused").count(),0,"devices badge while not paused");
  const recovery=page.locator("section",{has:page.getByRole("heading",{name:"Routing & recovery"})});
  assert.equal(await recovery.getByText("Disable send manipulation").count(),1);
  assert.equal(await recovery.getByText("3 held").count(),1);
  const tape=page.locator("[data-part=tape]");
  assert.deepEqual(await tape.getByRole("button").allTextContents(),["Record","Pause","Stop","",""]);
  assert.equal(await tape.getByRole("button",{name:"Fast-forward"}).count(),1);
  assert.equal(await page.getByText("Play recording").count(),0,"tape commands leaked into Routing & recovery");
  assert.equal(await tape.getByRole("button",{name:"Rewind"}).isDisabled(),true,"unavailable tape control enabled");
  await tape.getByRole("button",{name:"Stop"}).click();
  await page.waitForFunction(()=>window.sent.some(a=>a.Request?.ID==="audio.tape-stop"));
  await page.evaluate(()=>{window.sent.length=0;scrollTo(0,0);});
  const echo=page.locator("[data-part=echo]");
  await echo.getByText("2.5 / 1.2 ms",{exact:true}).waitFor();
  await echo.getByText("Peak worker / queue · gaps 3 · underruns 0",{exact:true}).waitFor();
  assert.equal(await echo.getByRole("heading",{name:"Echo cancellation"}).count(),1);
  assert.equal(await echo.getByText("Active",{exact:true}).count(),1,"echo status value");
  assert.equal(await echo.getByText("−32 dB echo · 54 ms").count(),1,"echo status detail");
  assert.equal(await echo.getByLabel("Mode").inputValue(),"auto");
  assert.deepEqual(await echo.getByLabel("Strength").locator("option").allTextContents(),["Strong","Balanced","Gentle"]);
  assert.deepEqual(await echo.getByLabel("Engine").locator("option").allTextContents(),["WebRTC AEC3","LocalVQE echo-only","LocalVQE voice cleanup","LocalVQE full-band"]);
  await echo.getByLabel("Engine").selectOption("localvqe-full");
  await page.waitForFunction(()=>window.fixture.Controls.find(c=>c.ID==="aec.engine").Value==="localvqe-full");
  await page.evaluate(()=>{window.fixture.Controls.find(c=>c.ID==="aec.strength").Available=false;window.fixture.Controls.find(c=>c.ID==="aec.status").Status="48 kHz mono · hybrid · ~63 ms processing latency";});
  await page.waitForFunction(()=>document.querySelector('[data-part=echo] select[aria-label="Strength"]')?.disabled || [...document.querySelectorAll('[data-part=echo] select')].some(x=>x.disabled));
  assert.equal(await echo.getByLabel("Strength").isDisabled(),true);
  await echo.getByText("48 kHz mono · hybrid · ~63 ms processing latency").waitFor();
  assert.equal(await echo.getByRole("button",{name:"Retry",exact:true}).isDisabled(),true);
  await page.evaluate(()=>{window.fixture.Controls.find(c=>c.ID==="aec.retry").Available=true;window.fixture.Controls.find(c=>c.ID==="aec.status").Value="Error";window.fixture.Controls.find(c=>c.ID==="aec.status").Status="Engine error (missing output callback); mic passes through; automatic retry pending";});
  await echo.getByRole("button",{name:"Retry",exact:true}).click();
  await page.waitForFunction(()=>window.sent.some(a=>a.Request?.ID==="aec.retry" && a.Request.Operation==="press"));
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  await page.screenshot({path:path.join(root,".local/gui-aec-retry.png"),fullPage:true});
  await echo.getByLabel("Mode").selectOption("off");
  await page.waitForFunction(()=>window.fixture.Controls.find(c=>c.ID==="aec.mode").Value==="off");
  await page.evaluate(()=>{window.sent.length=0;});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  await page.screenshot({path:path.join(root,".local/gui-audio.png"),fullPage:true});
  assert.equal(await page.getByText("Device priorities").count(),0,"priority controls leaked onto the Audio screen");
  await page.getByRole("button",{name:"Routing",exact:true}).click();
  for(const name of ["Outputs","Microphones","Interfaces","Playback","Mic priority"])await page.getByRole("heading",{name,exact:true}).waitFor();
  // Outputs matrix: sources by destination; hidden slots get no column.
  const outputs=page.locator("[data-part=outputs]");
  assert.equal(await outputs.locator("[data-part=output-slot]").count(),1,"hidden slot positions shown");
  assert.deepEqual(await outputs.locator("tbody th").allTextContents(),["Computer","Monitor","Soundboard","Tape"]);
  assert.deepEqual(await outputs.locator("[data-part=route]").allTextContents(),["On","On","Pre","Off","Off","On"]);
  await outputs.getByRole("button",{name:"Music Monitor"}).focus();
  await page.keyboard.press("Tab");
  assert.equal(await page.evaluate(()=>window.sent.length),0,"focus moves dispatched a route");
  await outputs.getByRole("button",{name:"Music Monitor"}).click();
  await page.waitForFunction(()=>window.sent.some(a=>a.Request?.ID==="audio.slot-1:monitor"&&a.Request.Operation==="press"));
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  // Slots may exist without a device.
  assert.equal(await outputs.getByLabel("New output device").locator("option").first().textContent(),"No device");
  assert.equal(await outputs.getByLabel("Music device").locator("option").first().textContent(),"No device");
  await outputs.getByLabel("New output device").selectOption("Speakers (Realtek Audio)");
  await outputs.getByLabel("New output name").fill("Monitor output");
  await outputs.getByRole("button",{name:"Add output"}).click();
  await page.waitForFunction(()=>window.sent.some(a=>a.Request?.ID==="audio.output-edit"));
  assert.deepEqual(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)),{op:"add",id:"",value:JSON.stringify({name:"Monitor output",device:"Speakers (Realtek Audio)",device_id:"{realtek}"})});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  await outputs.getByRole("button",{name:"Remove Music"}).click();
  await page.waitForFunction(()=>window.sent.filter(a=>a.Request?.ID==="audio.output-edit").length===2);
  assert.deepEqual(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)),{op:"remove",id:"music",value:""});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  await page.evaluate(()=>{window.sent.length=0;});
  const playbackList=page.locator("[data-part=list-playback]");
  assert.deepEqual(await playbackList.locator("[data-part=match]").allTextContents(),["In use","Ambiguous · 2","Ready"]);
  assert.equal(await playbackList.getByRole("button",{name:"Move Playback 1 up"}).isDisabled(),true,"first entry moves up");
  assert.equal(await page.locator("[data-part=list-interfaces]").getByRole("button",{name:"Remove interface 1"}).isDisabled(),true,"last interface removable");
  // Microphones: named, numbered by input; device microphones list their devices.
  const mics=page.locator("[data-part=mics]");
  assert.equal(await mics.locator("[data-part=mic]").count(),3);
  assert.equal(await mics.getByLabel("Microphone 2 name").inputValue(),"Lavalier");
  assert.equal(await mics.locator("[data-part=match]").textContent(),"No match");
  assert.equal(await page.locator("[data-part=list-microphones]").getByText("Lavalier").count(),1,"priority shows configured names");
  assert.deepEqual(await page.locator("[data-part=list-microphones] [data-part=priority] [data-tone]").allTextContents(),["In use","Unavailable"]);
  // Identity entries show the device, not a pattern.
  assert.equal(await playbackList.locator("[data-part=device-entry]").first().textContent(),"Headphones (AirPods Pro)");
  assert.equal(await playbackList.getByLabel("Playback 1 pattern").count(),0);
  // Suggestions add by identity; pattern forms are Snoofer's generated ones.
  assert.deepEqual(await playbackList.locator("[data-part=suggestions] button").allTextContents(),["Add","Exact pattern","Partial pattern"]);
  await playbackList.getByRole("button",{name:"Add Speakers (Realtek Audio)",exact:true}).click();
  await page.waitForFunction(()=>window.sent.some(a=>a.Request?.ID==="audio.priority-edit"));
  assert.deepEqual(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)),{list:"playback",op:"add",index:0,value:JSON.stringify({driver:"wdm",id:"{realtek}",name:"Speakers (Realtek Audio)"}),field:""});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  // A pattern applies only on Enter or Apply, never while typing.
  const pattern=playbackList.getByLabel("Playback 2 pattern");
  await pattern.fill("(?i)arena 7");
  assert.equal(await page.evaluate(()=>window.sent.filter(a=>a.Request?.ID==="audio.priority-edit").length),1,"typing dispatched an edit");
  await pattern.press("Enter");
  await page.waitForFunction(()=>window.sent.filter(a=>a.Request?.ID==="audio.priority-edit").length===2);
  assert.equal(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)).value,"(?i)arena 7");
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  await page.locator("[data-part=interface-add]").getByRole("button",{name:"Add"}).click();
  await page.waitForFunction(()=>window.sent.filter(a=>a.Request?.ID==="audio.priority-edit").length===3);
  assert.deepEqual(JSON.parse(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)).value),{asio_id:"{focusrite}",asio_name:"Focusrite USB ASIO",presence_pattern:"(?i)^Analogue 1 \\+ 2 \\(Focusrite USB\\)$",inputs:[1,2]});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  // Interface channels: one field per interface microphone, "left,right" for stereo.
  const ifaces=page.locator("[data-part=list-interfaces]");
  assert.equal(await ifaces.getByLabel("Desk channels for interface 1").inputValue(),"1");
  assert.equal(await ifaces.getByLabel("Webcam channels for interface 1").count(),0,"device microphone offered interface channels");
  await ifaces.getByLabel("Lavalier channels for interface 1").fill("3,4");
  await ifaces.getByLabel("Lavalier channels for interface 1").press("Tab");
  await page.waitForFunction(()=>window.sent.filter(a=>a.Request?.ID==="audio.priority-edit").length===4);
  assert.deepEqual(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)),{list:"interfaces",op:"set",index:0,value:"3,4",field:"input:lav"});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  // A new microphone: interface channels by default.
  await mics.getByLabel("New microphone name").fill("Guest");
  await mics.getByRole("button",{name:"Add microphone"}).click();
  await page.waitForFunction(()=>window.sent.filter(a=>a.Request?.ID==="audio.priority-edit").length===5);
  assert.deepEqual(JSON.parse(await page.evaluate(()=>window.sent.at(-1).Request.Value)),{list:"mics",op:"add",index:0,value:JSON.stringify({name:"Guest"}),field:""});
  await page.getByText("Sending",{exact:true}).waitFor({state:"hidden"});
  await page.evaluate(()=>{window.fixture.Controls.find(c=>c.ID==="audio.priority-edit").Status="error parsing regexp: missing closing ): `(unclosed`";window.sent.length=0;});
  await page.locator("[data-part=priority-error]").getByText("missing closing").waitFor();
  await page.screenshot({path:path.join(root,".local/gui-routing.png"),fullPage:true});
  await page.evaluate(()=>{window.fixture.Controls.find(c=>c.ID==="audio.priority-edit").Status="";});
  await page.getByRole("button",{name:"Stream Deck",exact:false}).click();
  await page.getByRole("heading",{name:"Binding",exact:true}).waitFor();
  assert.equal(await page.locator("[data-part=deck-key]").count(),36);
  assert.equal(await page.locator("[data-part=deck-key] small").first().textContent(),"1","key numbers start at 1");
  await page.getByRole("button",{name:"Key 1: Mic stack",exact:true}).focus();
  await page.keyboard.press("ArrowDown");
  assert.match(await page.locator(":focus").getAttribute("aria-label"),/^Key 10:/);
  assert.equal(await page.evaluate(()=>window.sent.length),0,"arrow browsing fired an action");
  await page.screenshot({path:path.join(root,".local/gui-deck.png"),fullPage:true});
  await page.getByRole("button",{name:"Key 8: Empty",exact:true}).click();
  await page.waitForFunction(()=>document.querySelector("[data-part=inspector] [data-part=eyebrow]").textContent==="KEY 8");
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"streamdeck.slot");
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:/^Key 19: /}).click();
  await page.waitForFunction(()=>document.querySelector("[data-part=inspector] [data-part=eyebrow]").textContent==="KEY 19");
  const beforeRange=await page.evaluate(()=>window.sent.length);
  await page.getByRole("button",{name:/^Key 35: /}).click({modifiers:["Shift"]});
  await page.keyboard.press("Shift+ArrowRight");
  assert.equal(await page.evaluate(()=>window.sent.length),beforeRange,"range selection dispatched");
  assert.equal(await page.locator("[data-part=deck-key][data-range=true]").count(),18);
  const addRegion=page.locator("[data-part=add-region]");
  assert.equal(await addRegion.textContent(),"Add region · Keys 19–36");
  await page.getByLabel("New region source").selectOption("soundboard.clips");
  await addRegion.click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Value];}),["streamdeck.region-add","18,35,soundboard.clips"]);
  await page.locator("[data-part=region]").waitFor();
  assert.equal(await page.locator("[data-part=deck-key][data-region=true]").count(),18);
  assert.equal(await page.locator("[data-part=deck-key][data-range=true]").count(),0,"selection kept after adding");
  await page.getByRole("button",{name:/^Key 27: /}).click({modifiers:["Shift"]});
  assert.equal(await addRegion.isDisabled(),true,"region offered over an existing region");
  await page.waitForTimeout(300);
  await page.getByLabel("Region 1 source").selectOption("hue.room-scenes");
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Value];}),["streamdeck.region-source","0,hue.room-scenes"]);
  await page.waitForTimeout(300);
  await page.screenshot({path:path.join(root,".local/gui-deck-regions.png"),fullPage:true});
  await page.getByRole("button",{name:"Remove region 1",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Value];}),["streamdeck.region-remove","0"]);
  await page.locator("[data-part=region]").waitFor({state:"detached"});
  // Dial regions: select a span of dials and add a region from it.
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:/^Dial 3: /}).click();
  await page.waitForTimeout(300);
  const beforeDials=await page.evaluate(()=>window.sent.length);
  await page.getByRole("button",{name:/^Dial 5: /}).click({modifiers:["Shift"]});
  assert.equal(await page.evaluate(()=>window.sent.length),beforeDials,"dial range selection dispatched");
  assert.equal(await page.locator("[data-part=dial][data-range=true]").count(),3);
  assert.equal(await addRegion.textContent(),"Add region · Dials 3–5");
  await page.getByLabel("New region source").selectOption("soundboard.clips");
  await addRegion.click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Value];}),["streamdeck.region-add","38,40,soundboard.clips"]);
  await page.locator("[data-part=region]").waitFor();
  assert.equal(await page.locator("[data-part=dial][data-region=true]").count(),3);
  assert.match(await page.locator("[data-part=region]").textContent(),/Dials 3–5/);
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Remove region 1",exact:true}).click();
  await page.locator("[data-part=region]").waitFor({state:"detached"});
  await page.waitForTimeout(300);
  await page.evaluate(()=>{window.fixture.Controls.find(c=>c.ID==="streamdeck.preview").ViewData.Dirty=false;});
  const name=page.getByLabel("Name",{exact:true});
  await name.fill("Unsubmitted name");
  await page.waitForTimeout(600);
  assert.equal(await name.inputValue(),"Unsubmitted name");
  await name.press("Escape");
  await name.fill("Studio");
  await page.getByRole("button",{name:"Apply",exact:true}).first().click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.Value),"Studio");
  await page.waitForTimeout(300);
  await page.evaluate(()=>{window.fixture.Controls.find(c=>c.ID==="streamdeck.preview").ViewData.Dirty=true;});
  await page.getByRole("button",{name:"Save layout",exact:true}).waitFor();
  await page.waitForFunction(()=>!Array.from(document.querySelectorAll("button")).find(b=>b.textContent==="Save layout").disabled);
  await page.getByRole("button",{name:"Save layout",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"streamdeck.save");
  await page.evaluate(()=>{window.fixture.Notice="Control changed; try again";});
  await page.getByRole("alert").filter({hasText:"Control changed; try again"}).waitFor();
  await page.getByRole("button",{name:"Soundboard",exact:false}).click();
  await page.getByRole("searchbox",{name:"Search clips"}).fill("sad");
  await page.waitForTimeout(600);
  assert.equal(await page.locator("[data-part=clip]:visible").count(),1);
  // Animated artwork cycles its frames.
  await page.waitForFunction(dot=>document.querySelector("[data-part=clip]:not([hidden]) img")?.src.endsWith(dot),dot);
  await page.screenshot({path:path.join(root,".local/gui-soundboard.png"),fullPage:true});
  await page.getByRole("button",{name:"Lights",exact:false}).click();
  await page.getByRole("heading",{name:"Bridge found at 172.16.102.3"}).waitFor();
  await page.getByRole("button",{name:"Pair",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"hue.pair");
  await page.evaluate(()=>{const s=window.fixture.Controls.find(c=>c.ID==="hue.status");s.Value="Connected";s.ViewData=null;s.Revision++;});
  await page.waitForFunction(()=>document.querySelector("[data-part=setup]").hidden);
  assert.deepEqual(await page.locator("[data-part=room-scenes] [data-part=scene] strong").allTextContents(),["Bright","Concentrate","Relax"]);
  assert.equal(await page.locator("[data-part=scene][data-state=playing]").count(),1);
  assert.equal(await page.locator("[data-part=scene] img").count(),1,"scene artwork not shown");
  assert.equal(await page.locator("details [data-part=room-heading]").allTextContents().then(t=>t.join(",")),"Desk,Kitchen");
  await page.screenshot({path:path.join(root,".local/gui-lights.png"),fullPage:true});
  await page.getByRole("button",{name:"BRIGHTNESS up",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation,r.Delta];}),["hue.brightness","adjust",1]);
  await page.waitForTimeout(300);
  await page.getByRole("slider",{name:"BRIGHTNESS"}).fill("80");
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Delta];}),["hue.brightness",9]);
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Relax",exact:false}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"hue.scene-studio-relax");
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Motion sensors",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation];}),["hue.motion","press"]);
  await page.evaluate(()=>{const m=window.fixture.Controls.find(c=>c.ID==="hue.motion");m.Hidden=true;m.Available=false;m.Revision++;});
  await page.locator("[data-part=motion]").waitFor({state:"hidden"});
  await page.waitForTimeout(300);
  await page.getByRole("checkbox",{name:"Desk (zone)",exact:true}).click(); // The host's state, not the click, sets the box.
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Value];}),["hue.rooms","room-1"]);
  await page.evaluate(()=>{const r=window.fixture.Controls.find(c=>c.ID==="hue.rooms");r.Value="room-1";r.ViewData[1].Chosen=false;r.Revision++;});
  await page.waitForFunction(()=>!document.querySelector("[data-part=rooms] input[value=zone-1]").checked);
  // The last chosen room cannot be unchecked; the other stays available.
  await page.waitForFunction(()=>{const [room,zone]=document.querySelectorAll("[data-part=rooms] input");return room.disabled&&!zone.disabled;});
  assert.equal(await page.getByRole("button",{name:"Games",exact:true}).isVisible(),false,"mode shown while not syncing");
  await page.evaluate(()=>{for(const id of ["hue.sync-mode","hue.sync-intensity"]){const m=window.fixture.Controls.find(c=>c.ID===id);m.Hidden=false;m.Available=true;m.Revision++;}});
  await page.getByRole("button",{name:"Games",exact:true}).waitFor();
  await page.getByRole("button",{name:"Games",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Value];}),["hue.sync-mode","games"]);
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Start sync",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"hue.sync");
  await page.getByRole("button",{name:"App audio",exact:true}).click();
  await page.locator("[data-part=app]").first().waitFor();
  assert.deepEqual(await page.locator("[data-part=app] h3").allTextContents(),["Discord","Google Chrome","Spotify"]);
  assert.equal(await page.getByRole("spinbutton",{name:"Recent window in minutes"}).inputValue(),"5");
  await page.getByRole("spinbutton",{name:"Recent window in minutes"}).fill("30");
  await page.keyboard.press("Enter");
  assert.deepEqual(await page.evaluate(()=>JSON.parse(window.sent.at(-1).Request.Value)),{op:"recent",app:"recent",value:"30"});
  await page.evaluate(()=>{const s=window.fixture.Controls.find(c=>c.ID==="appaudio.status");s.ViewData.RecentMinutes=30;s.Revision++;const e=window.fixture.Controls.find(c=>c.ID==="appaudio.edit");e.Revision++;});
  await page.waitForFunction(()=>document.querySelector("[data-part=recent]").value==="30");
  const sentBefore=await page.evaluate(()=>window.sent.length);
  await page.getByRole("spinbutton",{name:"Recent window in minutes"}).fill("0");
  await page.keyboard.press("Enter");
  assert.equal(await page.evaluate(()=>window.sent.length),sentBefore,"invalid window sent");
  assert.equal(await page.getByRole("spinbutton",{name:"Recent window in minutes"}).inputValue(),"30");
  await page.waitForTimeout(300);
  assert.equal(await page.getByRole("slider",{name:"Spotify volume"}).isVisible(),false,"closed app adjustable");
  await page.waitForTimeout(300);
  await page.getByRole("slider",{name:"Discord volume"}).fill("60");
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation,r.Value];}),["appaudio.app-discord","set","60"]);
  await page.waitForTimeout(300);
  await page.locator("[data-part=app]").nth(1).locator("[data-part=mute]").click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation];}),["appaudio.app-chrome","press"]);
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Pin Google Chrome",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,JSON.parse(r.Value)];}),["appaudio.edit",{op:"pick",app:"Google Chrome",value:""}]);
  await page.evaluate(()=>{const e=window.fixture.Controls.find(c=>c.ID==="appaudio.edit");e.Revision++;});
  await page.waitForTimeout(300);
  // Separate undoes a rule that named programs into an app.
  await page.locator("[data-part=app]").first().locator("summary").click();
  assert.equal(await page.locator("[data-part=app]").first().locator("[data-part=naming]").getByText("updater.exe").count(),1);
  await page.getByRole("button",{name:"Separate updater.exe",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>JSON.parse(window.sent.at(-1).Request.Value)),{op:"separate",app:"Discord",value:"(^|\\\\)updater\\.exe$"});
  await page.evaluate(()=>{const e=window.fixture.Controls.find(c=>c.ID==="appaudio.edit");e.Revision++;});
  await page.waitForTimeout(300);
  await page.locator("[data-part=app]").nth(1).locator("summary").click();
  await page.locator("[data-part=app]").nth(1).locator("[data-part=hide]").click();
  assert.deepEqual(await page.evaluate(()=>JSON.parse(window.sent.at(-1).Request.Value)),{op:"hide",app:"Google Chrome",value:""});
  await page.evaluate(()=>{const e=window.fixture.Controls.find(c=>c.ID==="appaudio.edit");e.Revision++;});
  await page.waitForTimeout(300);
  const excluded=page.locator("[data-part=excluded]");
  assert.match(await excluded.textContent(),/huesync\.exeHiding Hue Sync/);
  await excluded.getByRole("button",{name:"Remove huesync.exe",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>JSON.parse(window.sent.at(-1).Request.Value)),{op:"include",app:"huesync.exe",value:"huesync.exe"});
  await page.evaluate(()=>{const e=window.fixture.Controls.find(c=>c.ID==="appaudio.edit");e.Revision++;});
  await page.waitForTimeout(300);
  await excluded.getByRole("textbox",{name:"Program to exclude"}).fill("steam.exe");
  await excluded.locator("[data-part=exclude]").click();
  assert.deepEqual(await page.evaluate(()=>JSON.parse(window.sent.at(-1).Request.Value)),{op:"exclude",app:"steam.exe",value:"steam.exe"});
  await page.evaluate(()=>{const e=window.fixture.Controls.find(c=>c.ID==="appaudio.edit");e.Revision++;});
  await page.waitForTimeout(300);
  await excluded.getByRole("button",{name:"Unhide Launcher",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>JSON.parse(window.sent.at(-1).Request.Value)),{op:"unhide",app:"Launcher",value:""});
  await page.screenshot({path:path.join(root,".local/gui-appaudio.png"),fullPage:true});
  await page.getByRole("button",{name:"Media",exact:true}).click();
  await page.locator("[data-part=media-session]").first().waitFor();
  assert.deepEqual(await page.locator("[data-part=media-session] h3").allTextContents(),["Video A","Song"]);
  const [tab,song]=[page.locator("[data-part=media-session]").nth(0),page.locator("[data-part=media-session]").nth(1)];
  assert.equal(await tab.getAttribute("data-focused"),"true");
  assert.equal(await tab.getByRole("button",{name:"Next"}).isVisible(),true,"focused card lacks Next");
  assert.equal(await song.getByRole("button",{name:"Next"}).isVisible(),false,"unfocused card shows Next");
  assert.equal(await page.locator("[data-part=media-session] button",{hasText:"Focus"}).count(),0,"Focus button still offered");
  await song.locator("[data-part=play]").click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation];}),["nowplaying.s-spotify","press"]);
  await page.waitForTimeout(300);
  await song.locator("[data-part=seek]").fill("90000");
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation,r.Value];}),["nowplaying.s-spotify","set","90000"]);
  // After a seek the slider follows the session again, even while focused.
  await page.evaluate(()=>{const s=window.fixture.Controls.find(c=>c.ID==="nowplaying.status").ViewData.Sessions[1];s.PositionMs=120000;});
  await page.waitForFunction(()=>document.querySelectorAll("[data-part=seek]")[1].value==="120000",null,{polling:50});
  // A playing session advances between polls.
  const first=Number(await tab.locator("[data-part=seek]").inputValue());
  await page.waitForTimeout(1200);
  assert.ok(Number(await tab.locator("[data-part=seek]").inputValue())>=first+1000,"playing progress did not advance");
  await page.waitForTimeout(300);
  await tab.locator("[data-part=mute]").click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"nowplaying.mute");
  const extension=page.locator("[data-part=extension]");
  assert.match(await extension.textContent(),/Brave connected · 1 with media · extension 1\.0\.0/);
  assert.match(await extension.textContent(),/Update the Snoofer Media extension in Firefox/);
  assert.equal(await extension.getByRole("button").count(),0,"extension card offers export or token actions");
  await page.screenshot({path:path.join(root,".local/gui-media.png"),fullPage:true});
  await page.getByRole("button",{name:"Meetings",exact:true}).click();
  const camera=page.locator("[data-part=camera]"),discord=page.locator("[data-part=discord]");
  await camera.waitFor();
  assert.equal(await camera.locator("[data-part=state]").textContent(),"Working");
  assert.equal(await discord.locator("[data-part=state]").textContent(),"General");
  assert.equal(await page.locator("[data-part=discord-setup]").isVisible(),false,"setup hint while configured");
  assert.equal(await page.locator("[data-part=discord-connect]").isVisible(),false,"Connect while connected");
  assert.equal(await camera.locator("select").first().inputValue(),"group");
  await page.screenshot({path:path.join(root,".local/gui-meetings.png"),fullPage:true});
  await camera.getByRole("button",{name:"Privacy",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"insta360.privacy");
  await discord.getByRole("button",{name:"Mute",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"discord.mute");
  await discord.getByRole("button",{name:"Leave call",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"discord.leave");
  await page.evaluate(()=>{const s=window.fixture.Controls.find(c=>c.ID==="discord.mute");s.Available=false;s.Status="Setup needed";});
  await page.locator("[data-part=discord-setup]").waitFor();
  await page.evaluate(()=>{const s=window.fixture.Controls.find(c=>c.ID==="discord.mute");s.Available=true;s.Status="";window.fixture.Enabled.discord=false;});
  await page.getByRole("button",{name:"Enable Discord",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const a=window.sent.at(-1);return [a.Kind,a.Plugin,a.Enable];}),["selection","discord",true]);
  await page.evaluate(()=>{window.fixture.Enabled.discord=true;});
  await discord.waitFor();
  await page.getByRole("button",{name:"Third-party apps",exact:false}).click();
  await page.getByRole("heading",{name:"Hue",exact:true}).waitFor();
  assert.equal(await page.locator("[data-part=app-card]").count(),4);
  assert.deepEqual(await page.locator("[data-part=summary] span").allTextContents(),["1 OK","0 Attention","1 Problems","2 Idle"]);
  const vm=page.locator("[data-part=app-card]",{hasText:"Voicemeeter"});
  assert.equal(await vm.getAttribute("data-tone"),"critical","required peer down must be critical");
  assert.equal(await vm.locator("[data-part=app-details] dt").allTextContents().then(t=>t.join(",")),"Edition,Interval","Required stays internal");
  const bridge=page.locator("[data-part=app-card]",{hasText:"Hue Bridge"});
  assert.match(await bridge.locator("[data-part=app-times]").textContent(),/^Since 10m ago · Last activity (just now|\d+s ago)$/);
  assert.match(await bridge.locator("[data-part=app-error]").textContent(),/^Last error: event stream ended: EOF · 11m ago$/);
  assert.equal(await page.locator("[data-part=app-card]",{hasText:"callback"}).locator("[data-part=app-times]").isVisible(),false,"zero times shown");
  await bridge.getByRole("button",{name:"Copy details"}).click();
  await page.waitForFunction(()=>window.copied.length===1);
  const copied=await page.evaluate(()=>window.copied[0]);
  assert.match(copied,/^Hue Bridge \(Hue\)\nState: Connected\nEndpoint: 172\.16\.102\.3\nSince: \d{4}-/);
  assert.equal(copied.includes("Bridge ID: 001788fffe2490e0"),true);
  await page.getByRole("button",{name:"Copy all",exact:true}).click();
  await page.waitForFunction(()=>window.copied.length===2);
  assert.equal((await page.evaluate(()=>window.copied[1])).split("\n\n").length,4);
  assert.deepEqual(await page.locator("[data-part=not-monitored] li").allTextContents(),["media — Disabled"]);
  await page.screenshot({path:path.join(root,".local/gui-apps.png"),fullPage:true});
  await page.getByRole("button",{name:"Audio",exact:true}).click();
  await page.setViewportSize({width:800,height:600});
  await page.screenshot({path:path.join(root,".local/gui-narrow.png"),fullPage:true});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"horizontal overflow");
  await page.getByRole("button",{name:"Stream Deck",exact:false}).click();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"deck horizontal overflow");
  await page.getByRole("button",{name:"Lights",exact:false}).click();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"lights horizontal overflow");
  await page.getByRole("button",{name:"Meetings",exact:true}).click();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"meetings horizontal overflow");
  await page.getByRole("button",{name:"Third-party apps",exact:false}).click();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"apps horizontal overflow");
  await page.getByRole("button",{name:"Diagnostics",exact:false}).click();
  await page.getByRole("button",{name:"Restart audio engine",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"audio.engine-restart");
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Plugins",exact:true}).click();
  assert.equal(await page.locator("[data-part=row]").count(),0,"Plugins page shows configuration");
  assert.equal(await page.getByText("Hue brightness").count(),0,"Plugins page shows plugin controls");
  await page.getByRole("button",{name:"Enable",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Plugin),"media");
  await page.evaluate(()=>window.fixture.Confirmation="Enable media and restart Snoofer?");
  await page.getByRole("dialog").waitFor();
  await page.getByRole("button",{name:"Cancel",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Kind),"cancel");
  assert.deepEqual(errors,[]);
  console.log("PASS: GUI screens, deck selection, draft text, search, lights, meetings, third-party apps, responsive bounds and enable-only plugins");
 }finally{await browser.close();await new Promise(resolve=>server.close(resolve));}
})().catch(e=>{console.error(e);process.exitCode=1;});

