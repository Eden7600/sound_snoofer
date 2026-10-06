import test from "node:test";
import assert from "node:assert/strict";
import {compatible,tone,meterValue,gridMove} from "./model.mjs";
test("binding compatibility, semantic states and meter expiry",()=>{
 assert.equal(compatible({Operations:["press"]},true),false);
 assert.equal(compatible({Operations:["set"]},false),true);
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
