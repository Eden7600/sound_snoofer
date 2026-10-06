// Run with playwright available on NODE_PATH or in node_modules; uses installed Chrome.
const { chromium }=require("playwright");
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
add("audio.auto-recover","Automatic recovery","toggle","On",{Group:"Shared audio"});
add("soundboard.volume","Volume","numeric","0.0 dB");
add("soundboard.stop","Stop","command","");
add("soundboard.status","Soundboard","status","Ready");
for(const name of ["fah","sadge","instinct","airhorn","ping"])add("soundboard.clip-"+name,name,"command","Ready");
for(const [id,label,kind,value,options]of [["profile","Device","selection","Default",["Default"]],["page","Page","selection","home",["home","soundboard"]],["slot","Position","selection","Key 1",Array.from({length:36},(_,i)=>"Key "+(i+1))],["shared","Shared","toggle","Off"],["binding","Binding","selection","audio.mic-stack",["","audio.mic-stack","audio.mic-mute"]],["name","Name","text","Home"],["add","New page","text",""],["auto-controls","Automatic prefix","text",""],["preview","Preview","status",""],["status","Layout status","status","Connected"]])add("streamdeck."+id,label,kind,value,{Options:options});
for(const id of ["save","cancel","delete","earlier","later","home"])add("streamdeck."+id,id,"command","");
add("hue.status","Hue","status","Not paired",{ViewData:{bridge:"172.16.102.3"}});
add("hue.pair","Pair Hue bridge","command","Ready",{ShortLabel:"Pair"});
add("hue.group","Hue room","selection","room-1",{Options:["zone-1","room-1"],OptionLabels:{"zone-1":"Desk (zone)","room-1":"Studio"}});
add("hue.brightness","Hue brightness","numeric","62%",{ShortLabel:"Brightness"});
const sceneArt=fs.readFileSync(path.join(root,"docs/design/hue-scene-art.png")).toString("base64");
for(const [room,name,value] of [["Studio","Bright","Ready"],["Studio","Relax","Active"],["Studio","Concentrate","Ready"],["Desk","Focus","Ready"],["Kitchen","Cook","Ready"]])add("hue.scene-"+room.toLowerCase()+"-"+name.toLowerCase(),room+" "+name,"command",value,{ShortLabel:name,Group:"Hue scenes",Artwork:name==="Relax"?sceneArt:""});
add("hue.sync-status","Hue Sync","status","Ready");
add("hue.sync","Hue Sync","toggle","Off",{ShortLabel:"Sync"});
add("hue.sync-mode","Hue Sync mode","selection","video",{Options:["video","games","music"],OptionLabels:{video:"Video",games:"Games",music:"Music"},Available:false});
add("hue.sync-intensity","Hue Sync intensity","selection","moderate",{Options:["subtle","moderate","high","extreme"],OptionLabels:{subtle:"Subtle",moderate:"Moderate",high:"High",extreme:"Extreme"},Available:false});
for(let n=1;n<=12;n++)add("hue.room-scene-"+n,"","command","",{Available:false,Group:"Hue room scenes"});
const view={Selected:0,Dirty:false,Home:true,Keys:Array.from({length:36},()=>({Control:"",Label:"",Source:""})),Dials:Array.from({length:5},()=>({Control:"",Label:"",Source:""}))};
["audio.mic-stack","audio.mic-mute","audio.speaker-mute","audio.normal-monitor","audio.normal-mode"].forEach((Control,i)=>view.Keys[i]={Control,Label:"",Source:""});
view.Keys[9]={Control:"soundboard.clip-fah",Label:"fah",Source:"Auto"};
view.Keys[31]={Control:"core.open-controls",Label:"Controls",Source:"Shared"};
["audio.gain-playback","audio.gain-mic",""].forEach((Control,i)=>view.Dials[i]={Control,Label:"",Source:""});
controls.find(c=>c.ID==="streamdeck.preview").ViewData=view;
const fixture={Controls:controls,Plugins:{audio:"Running",soundboard:"Running",streamdeck:"Running",vr:"Running",media:"Disabled",hue:"Running"},Enabled:{audio:true,soundboard:true,streamdeck:true,vr:true,media:false,hue:true}};
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
  await page.addInitScript(fixture=>{
   window.fixture=fixture;window.sent=[];
   window.go={app:{Desktop:{State:async()=>structuredClone(window.fixture),Send:async action=>{
    window.sent.push(action);
    const r=action.Request;if(!r)return;
    const target=window.fixture.Controls.find(c=>c.ID===r.ID);
    if(r.ID==="streamdeck.slot"){window.fixture.Controls.find(c=>c.ID==="streamdeck.preview").ViewData.Selected=Number(r.Value.split(" ")[1])-1;}
    if(r.Operation==="set")target.Value=r.Value;
    target.Revision++;
   }}}};
  },fixture);
  await page.goto("http://127.0.0.1:"+server.address().port);
  await page.getByRole("heading",{name:"Live controls"}).waitFor();
  await page.waitForFunction(()=>document.querySelector(".brandmark").naturalWidth>0);
  assert.equal(await page.locator(".strip").count(),2);
  assert.equal(await page.getByRole("heading",{name:"PLAYBACK",exact:true}).count(),1);
  assert.equal(await page.getByText("A1 OUTPUT",{exact:true}).count(),0);
  await page.screenshot({path:path.join(root,".local/gui-audio.png"),fullPage:true});
  await page.getByRole("button",{name:"Stream Deck",exact:false}).click();
  await page.getByRole("heading",{name:"Binding",exact:true}).waitFor();
  assert.equal(await page.locator(".deck-key").count(),36);
  await page.getByRole("button",{name:"Key 0: Mic stack",exact:true}).focus();
  await page.keyboard.press("ArrowDown");
  assert.match(await page.locator(":focus").getAttribute("aria-label"),/^Key 9:/);
  assert.equal(await page.evaluate(()=>window.sent.length),0,"arrow browsing fired an action");
  await page.screenshot({path:path.join(root,".local/gui-deck.png"),fullPage:true});
  await page.getByRole("button",{name:"Key 7: Empty",exact:true}).click();
  await page.waitForFunction(()=>document.querySelector(".inspector .eyebrow").textContent==="KEY 7");
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"streamdeck.slot");
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
  assert.equal(await page.locator(".clip:visible").count(),1);
  await page.screenshot({path:path.join(root,".local/gui-soundboard.png"),fullPage:true});
  await page.getByRole("button",{name:"Lights",exact:false}).click();
  await page.getByRole("heading",{name:"Bridge found at 172.16.102.3"}).waitFor();
  await page.getByRole("button",{name:"Pair",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"hue.pair");
  await page.evaluate(()=>{const s=window.fixture.Controls.find(c=>c.ID==="hue.status");s.Value="Connected";s.ViewData=null;s.Revision++;});
  await page.waitForFunction(()=>document.querySelector(".setup").hidden);
  assert.deepEqual(await page.locator(".lights-grid > .panel:first-child > .scene-grid .clip strong").allTextContents(),["Bright","Concentrate","Relax"]);
  assert.equal(await page.locator(".clip.scene.playing").count(),1);
  assert.equal(await page.locator(".clip.scene .clip-art img").count(),1,"scene artwork not shown");
  assert.equal(await page.locator("details .room-heading").allTextContents().then(t=>t.join(",")),"Desk,Kitchen");
  await page.screenshot({path:path.join(root,".local/gui-lights.png"),fullPage:true});
  await page.getByRole("button",{name:"BRIGHTNESS up",exact:true}).click();
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Operation,r.Delta];}),["hue.brightness","adjust",1]);
  await page.waitForTimeout(300);
  await page.getByRole("slider",{name:"BRIGHTNESS"}).fill("80");
  assert.deepEqual(await page.evaluate(()=>{const r=window.sent.at(-1).Request;return [r.ID,r.Delta];}),["hue.brightness",9]);
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Relax",exact:false}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"hue.scene-studio-relax");
  assert.equal(await page.getByRole("button",{name:"Games",exact:true}).isDisabled(),true,"mode enabled while not syncing");
  await page.getByText("Start sync to change mode and intensity.").waitFor();
  await page.waitForTimeout(300);
  await page.getByRole("button",{name:"Start sync",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Request.ID),"hue.sync");
  await page.getByRole("button",{name:"Audio",exact:false}).click();
  await page.setViewportSize({width:800,height:600});
  await page.screenshot({path:path.join(root,".local/gui-narrow.png"),fullPage:true});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"horizontal overflow");
  await page.getByRole("button",{name:"Stream Deck",exact:false}).click();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"deck horizontal overflow");
  await page.getByRole("button",{name:"Lights",exact:false}).click();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,"lights horizontal overflow");
  await page.getByRole("button",{name:"Plugins",exact:false}).click();
  assert.equal(await page.locator(".control-row").count(),0,"Plugins page shows configuration");
  assert.equal(await page.getByText("Hue brightness").count(),0,"Plugins page shows plugin controls");
  await page.getByRole("button",{name:"Enable",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Plugin),"media");
  await page.evaluate(()=>window.fixture.Confirmation="Enable media and restart Snoofer?");
  await page.getByRole("dialog").waitFor();
  await page.getByRole("button",{name:"Cancel",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).Kind),"cancel");
  assert.deepEqual(errors,[]);
  console.log("PASS: GUI screens, deck selection, draft text, search, lights, responsive bounds and enable-only plugins");
 }finally{await browser.close();await new Promise(resolve=>server.close(resolve));}
})().catch(e=>{console.error(e);process.exitCode=1;});

