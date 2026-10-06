// Native WebView2/IPC smoke; isolated fixture, no plugins or personal settings.
const {spawn,execFileSync}=require("node:child_process");
const path=require("node:path"),fs=require("node:fs"),assert=require("node:assert/strict");
const root=path.resolve(__dirname,".."),delay=ms=>new Promise(r=>setTimeout(r,ms));
async function waitExit(exited,ms,label){
 let timer;
 try{return await Promise.race([exited,new Promise((_,reject)=>{timer=setTimeout(()=>reject(new Error(label)),ms);})]);}
 finally{clearTimeout(timer);}
}
async function closeGUI(child,exited){
 if(child.exitCode!==null||child.signalCode!==null)return;
 child.stdin.end();
 try{await waitExit(exited,7000,"GUI ignored EOF");}
 catch(error){
  // Only this isolated __controls child, which never loads audio plugins.
  child.kill();
  await waitExit(exited,3000,"Forced GUI cleanup did not finish");
  throw new Error("GUI cleanup required forced termination",{cause:error});
 }
}
async function checkTrayShutdown(){
 const config=path.join(root,".local","shutdown-smoke-"+process.pid+".json");
 fs.writeFileSync(config,JSON.stringify({version:1,plugins:{}}),{flag:"wx"});
 const child=spawn(path.join(root,"bin/snoofer.exe"),["--dry-run","--config",config],{cwd:root,windowsHide:true,stdio:"ignore"});
 const exited=new Promise((resolve,reject)=>{child.once("exit",resolve);child.once("error",reject);});
 const stop=pid=>execFileSync("powershell.exe",["-NoProfile","-ExecutionPolicy","Bypass","-File",path.join(root,"scripts/stop.ps1"),"-ProcessId",String(pid)],{cwd:root,windowsHide:true,encoding:"utf8",timeout:25000});
 try{
  await delay(500);
  stop(process.pid); // An unrelated executable must not receive a close request.
  await delay(100); // Deliver exit events after the synchronous helper returns.
  assert.equal(child.exitCode,null,"unrelated PID stopped the fixture");
  console.log(stop(child.pid).trim());
  assert.equal(await waitExit(exited,2000,"tray survived graceful stop"),0);
  stop(child.pid); // Already stopped is harmless.
 }finally{
  if(child.exitCode===null&&child.signalCode===null){stop(child.pid);await waitExit(exited,2000,"tray cleanup did not finish");}
  fs.unlinkSync(config);
 }
 console.log("PASS: scoped graceful tray shutdown and repeated stop");
}
(async()=>{
 const child=spawn(path.join(root,"bin/snoofer.exe"),["__controls"],{cwd:root,windowsHide:false,stdio:["pipe","pipe","pipe"]});
 let output="",errors="";child.stdout.on("data",s=>output+=s);child.stderr.on("data",s=>errors+=s);
 const exited=new Promise(resolve=>child.once("exit",(code,signal)=>resolve({code,signal})));
 child.stdin.on("error",()=>{});
 const state={Controls:[{ID:"audio.health",Label:"Audio health",Kind:"status",Value:"Preview",Available:true,Revision:1},{ID:"hue.status",Label:"Hue",Group:"Hue",Kind:"status",Value:"Connected",Available:true,Revision:1},{ID:"hue.sync",Label:"Hue Sync",Group:"Hue",Kind:"toggle",Value:"Off",Available:true,Operations:["press"],Revision:9}],Plugins:{hue:"Running"},Enabled:{hue:true}};
 child.stdin.write(JSON.stringify({State:state})+"\n");
 try{
  await delay(2500);
  const result=execFileSync("powershell.exe",["-NoProfile","-ExecutionPolicy","Bypass","-File",path.join(root,"scripts/check-desktop-window.ps1"),"-ProcessId",String(child.pid)],{cwd:root,windowsHide:true,encoding:"utf8",timeout:25000});
  console.log(result.trim());
  for(let i=0;i<20&&!output.includes("hue.sync");i++)await delay(100);
  const action=JSON.parse(output.trim().split("\n").at(-1));
  assert.equal(action.Request.ID,"hue.sync");assert.equal(action.Request.Revision,9);
  child.stdin.write(JSON.stringify({State:state,Focus:true})+"\n");
  execFileSync("powershell.exe",["-NoProfile","-File",path.join(root,"scripts/check-desktop-window.ps1"),"-ProcessId",String(child.pid),"-CloseOnly"],{windowsHide:true,timeout:15000});
  const ended=await waitExit(exited,7000,"GUI survived window close");
  assert.equal(ended.code,0);
  const second=spawn(path.join(root,"bin/snoofer.exe"),["__controls"],{cwd:root,windowsHide:false,stdio:["pipe","ignore","ignore"]});
  const secondExit=new Promise(resolve=>second.once("exit",resolve));
  second.stdin.on("error",()=>{});
  try {
   second.stdin.write(JSON.stringify({State:state})+"\n");
   await delay(1500);
   assert.throws(()=>execFileSync("powershell.exe",["-NoProfile","-File",path.join(root,"scripts/stop.ps1"),"-ProcessId",String(second.pid),"-TimeoutSeconds","1"],{windowsHide:true,stdio:"pipe",encoding:"utf8",timeout:10000}),error=>error.status!==0&&String(error.stderr).includes("no process was killed"));
   assert.equal(second.exitCode,null,"timeout killed the controls fixture");
   await closeGUI(second,secondExit);
   const code=await waitExit(secondExit,7000,"GUI survived host EOF");
   assert.equal(code,0);
  }finally{await closeGUI(second,secondExit);}
  console.log("PASS: native WebView2, accessible controls, revisioned actions, window close/reopen and host EOF cleanup");
 }finally{await closeGUI(child,exited);}
 await checkTrayShutdown();
})().catch(error=>{console.error(error);process.exitCode=1;});

