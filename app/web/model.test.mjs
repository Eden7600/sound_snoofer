import test from "node:test";
import assert from "node:assert/strict";
import {compatible,tone,meterValue,gridMove} from "./model.mjs";
test("binding compatibility, semantic states and meter expiry",()=>{
 assert.equal(compatible({Operations:["press"]},true),false);
 assert.equal(compatible({Operations:["set"]},false),true);
 assert.equal(compatible({Kind:"text",Operations:["set"]},false),false);
 assert.equal(tone({ID:"audio.mic-mute",Available:true,Value:"On"}),"critical");
 assert.equal(tone({ID:"audio.mic-stack",Available:true,Value:"On"}),"active");
 assert.equal(tone({Available:false,Status:"Error"}),"muted");
 const at=Date.now();
 assert.equal(meterValue({Present:true,Known:true,DB:-18,At:new Date(at).toISOString()},at),-18);
 assert.equal(meterValue({Present:true,Known:true,DB:-18,At:new Date(at-501).toISOString()},at),null);
 assert.equal(gridMove(8,"ArrowDown",9,36),17);
 assert.equal(gridMove(0,"ArrowLeft",9,36),0);
});

test("light values and scene rooms",async()=>{
 const {numericValue,sceneRoom}=await import("./model.mjs");
 assert.equal(numericValue("62%"),62);
 assert.equal(numericValue("Sync 62%"),62);
 assert.equal(numericValue("4000K"),4000);
 assert.equal(numericValue("Off"),null);
 assert.equal(sceneRoom({Label:"Living Room Bright",ShortLabel:"Bright"}),"Living Room");
 assert.equal(sceneRoom({Label:"Bright",ShortLabel:"Bright"}),"");
});
test("connection report helpers",async()=>{
 const m=await import("./model.mjs");
 const now=Date.parse("2026-10-05T12:00:00Z");
 assert.equal(m.relativeTime("0001-01-01T00:00:00Z",now),"");
 assert.equal(m.relativeTime("2026-10-05T11:59:59.5Z",now),"just now");
 assert.equal(m.relativeTime("2026-10-05T11:59:30Z",now),"30s ago");
 assert.equal(m.relativeTime("2026-10-05T11:57:00Z",now),"3m ago");
 assert.equal(m.relativeTime("2026-10-05T09:00:00Z",now),"3h ago");
 assert.equal(m.intervalMs("100 ms"),100);assert.equal(m.intervalMs("1s"),1000);assert.equal(m.intervalMs("2 min"),120000);assert.equal(m.intervalMs("often"),null);
 const live={State:"connected",LastActivity:"2026-10-05T11:59:59Z",Details:[{Label:"Interval",Value:"1s"}]};
 assert.equal(m.stale(live,now),false);
 assert.equal(m.stale({...live,LastActivity:"2026-10-05T11:59:50Z"},now),true);
 assert.equal(m.stale({...live,State:"disconnected",LastActivity:"2026-10-05T11:00:00Z"},now),false);
 assert.equal(m.connectionTone({State:"ready"}),"active");
 assert.equal(m.connectionTone({State:"attention"}),"attention");
 assert.equal(m.connectionTone({State:"disconnected"}),"");
 assert.equal(m.connectionTone({State:"disconnected",Details:[{Label:"Required",Value:"Yes"}]}),"critical");
 assert.equal(m.connectionTone({State:"off"}),"muted");
 const text=m.connectionText({Label:"Hue Bridge",Group:"Hue",Value:"Connected",Connection:{State:"connected",Endpoint:"172.16.102.3",Since:"2026-10-05T11:00:00Z",LastError:"timeout",LastErrorAt:"2026-10-05T10:00:00Z",Details:[{Label:"Bridge ID",Value:"abc"}]}});
 assert.equal(text,"Hue Bridge (Hue)\nState: Connected\nEndpoint: 172.16.102.3\nSince: 2026-10-05T11:00:00.000Z\nLast error: timeout (2026-10-05T10:00:00.000Z)\nBridge ID: abc");
});
test("stalled engine health is critical",()=>{
 assert.equal(tone({ID:"audio.health",Available:true,Value:"Audio engine stalled: no callback buffers · restart required"}),"critical");
 assert.equal(tone({ID:"audio.health",Available:true,Value:"No stall detected"}),"");
});
test("meter ballistics and gain position",async()=>{
 const {meterBallistics,gainPosition}=await import("./model.mjs");
 let s=meterBallistics(null,-10,0);
 assert.deepEqual([s.level,s.peak],[-10,-10]);
 s=meterBallistics(s,-40,500);
 assert.ok(Math.abs(s.level+22)<0.01);
 assert.equal(s.peak,-10);
 s=meterBallistics(s,-5,600);
 assert.deepEqual([s.level,s.peak],[-5,-5]);
 s=meterBallistics(s,-50,2600);
 assert.ok(s.peak<-5&&s.peak>=s.level);
 assert.deepEqual(meterBallistics(s,null,2700),{level:null,peak:null});
 assert.deepEqual(gainPosition("-6.0 dB"),{position:54/72,zero:60/72});
 assert.deepEqual(gainPosition("62%"),{position:0.62,zero:null});
 assert.equal(gainPosition("Off"),null);
});
