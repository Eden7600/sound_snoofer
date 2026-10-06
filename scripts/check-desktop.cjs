// Native WebView2/IPC smoke; isolated fixture, no plugins or personal settings.
const {spawn,execFileSync}=require("node:child_process");
const path=require("node:path"),assert=require("node:assert/strict");
const root=path.resolve(__dirname,".."),delay=ms=>new Promise(r=>setTimeout(r,ms));
(async()=>{
 const child=spawn(path.join(root,"bin/snoofer.exe"),["__controls"],{cwd:root,windowsHide:false,stdio:["pipe","pipe","pipe"]});
 let output="",errors="";child.stdout.on("data",s=>output+=s);child.stderr.on("data",s=>errors+=s);
 const exited=new Promise(resolve=>child.once("exit",(code,signal)=>resolve({code,signal})));
 child.stdin.on("error",()=>{});
 const state={Controls:[{ID:"audio.health",Label:"Audio health",Kind:"status",Value:"Preview",Available:true,Revision:1},{ID:"external.light",Label:"Light",Group:"Lighting",Kind:"toggle",Value:"Off",Available:true,Operations:["press"],Revision:9}],Plugins:{external:"Running"},Enabled:{external:true}};
 child.stdin.write(JSON.stringify({State:state})+"\n");
 try{
  await delay(2500);
  const result=execFileSync("powershell.exe",["-NoProfile","-ExecutionPolicy","Bypass","-File",path.join(root,"scripts/check-desktop-window.ps1"),"-ProcessId",String(child.pid)],{cwd:root,windowsHide:true,encoding:"utf8",timeout:25000});
  console.log(result.trim());
  for(let i=0;i<20&&!output.includes("external.light");i++)await delay(100);
  const action=JSON.parse(output.trim().split("\n").at(-1));
  assert.equal(action.Request.ID,"external.light");assert.equal(action.Request.Revision,9);
  child.stdin.write(JSON.stringify({State:state,Focus:true})+"\n");
  execFileSync("powershell.exe",["-NoProfile","-File",path.join(root,"scripts/check-desktop-window.ps1"),"-ProcessId",String(child.pid),"-CloseOnly"],{windowsHide:true,timeout:15000});
  const ended=await Promise.race([exited,delay(7000).then(()=>{throw new Error("GUI survived window close");})]);
  assert.equal(ended.code,0);
  const second=spawn(path.join(root,"bin/snoofer.exe"),["__controls"],{cwd:root,windowsHide:false,stdio:["pipe","ignore","ignore"]});
  const secondExit=new Promise(resolve=>second.once("exit",resolve));
  second.stdin.on("error",()=>{});
  try {
   second.stdin.write(JSON.stringify({State:state})+"\n");
   await delay(1500);
   second.stdin.end();
   const code=await Promise.race([secondExit,delay(7000).then(()=>{throw new Error("GUI survived host EOF");})]);
   assert.equal(code,0);
  }finally{if(second.exitCode===null)second.kill();}
  console.log("PASS: native WebView2, accessible controls, revisioned actions, window close/reopen and host EOF cleanup");
 }finally{if(child.exitCode===null)child.kill();}
})().catch(error=>{console.error(error);process.exitCode=1;});

