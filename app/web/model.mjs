export function controlsByID(state) {
 return new Map((state.Controls || []).map(c => [c.ID,c]));
}
export function compatible(control, dial) {
 const ops=control.Operations || [];
 return dial ? ops.includes("adjust") : ops.includes("press") || ops.includes("set");
}
export function tone(control) {
 if (!control || !control.Available) return "muted";
 const status=control.Status || "";
 if (/failed|error/i.test(status)) return "critical";
 if (/pending|wait|fallback|override/i.test(status) || control.Value==="Wait") return "attention";
 if (control.Value==="On" && /mute/.test(control.ID)) return "critical";
 if (control.Value==="On" || control.Value==="Playing") return "active";
 return "";
}
export function meterValue(meter, now=Date.now()) {
 if (!meter?.Present || !meter.Known || !Number.isFinite(meter.DB) || now-Date.parse(meter.At)>500 || !Number.isFinite(Date.parse(meter.At))) return null;
 return Math.min(0,Math.max(-60,meter.DB));
}
export function display(control) {
 if (!control.Available) return "Unavailable";
 return control.OptionLabels?.[control.Value] ?? control.Value ?? "";
}
export function gridMove(index,key,columns,count) {
 const step={ArrowLeft:-1,ArrowRight:1,ArrowUp:-columns,ArrowDown:columns}[key];
 if (step===undefined) return index;
 return Math.max(0,Math.min(count-1,index+step));
}

