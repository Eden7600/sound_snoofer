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
 if (/stalled/i.test(control.Value||"")) return "critical";
 if (/failed|error/i.test(status)) return "critical";
 if (/pending|wait|fallback|override/i.test(status) || control.Value==="Wait") return "attention";
 if (control.Value==="On" && /mute/.test(control.ID)) return "critical";
 if (control.Value==="On" || control.Value==="Playing" || control.Value==="Active") return "active";
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

// numericValue reads the number in a dial value such as "62%", "Sync 62%" or "4000K".
export function numericValue(text) {
 const match=/(-?\d+(?:\.\d+)?)/.exec(text||"");
 return match ? Number(match[1]) : null;
}
// sceneRoom recovers the room from a scene Label of the form "<Room> <Scene>".
export function sceneRoom(control) {
 const label=control?.Label||"", name=control?.ShortLabel||"";
 return label.endsWith(name) ? label.slice(0,label.length-name.length).trim() : "";
}
// Connection reports (Kind "connection"). Go zero times marshal as year 0001.
export function timeValue(iso) {
 const t=Date.parse(iso||"");
 return Number.isFinite(t)&&t>0 ? t : null;
}
export function relativeTime(iso, now=Date.now()) {
 const t=timeValue(iso);
 if (t===null) return "";
 const s=Math.max(0,Math.round((now-t)/1000));
 if (s<2) return "just now";
 if (s<60) return s+"s ago";
 if (s<3600) return Math.floor(s/60)+"m ago";
 if (s<86400) return Math.floor(s/3600)+"h ago";
 return Math.floor(s/86400)+"d ago";
}
export function detail(conn, label) {
 return (conn?.Details||[]).find(d=>d.Label===label)?.Value ?? "";
}
// intervalMs parses a provider "Interval" detail such as "1s", "100 ms" or "2 min".
export function intervalMs(text) {
 const m=/^\s*([\d.]+)\s*(ms|s|sec|m|min)\s*$/i.exec(text||"");
 if (!m) return null;
 const n=Number(m[1]), unit=m[2].toLowerCase();
 return unit==="ms"?n:unit==="s"||unit==="sec"?n*1000:n*60000;
}
export function stale(conn, now=Date.now()) {
 const interval=intervalMs(detail(conn,"Interval")), last=timeValue(conn?.LastActivity);
 return interval!==null && last!==null && ["connected","ready","attention"].includes(conn.State) && now-last>3*interval;
}
export function connectionTone(conn) {
 switch (conn?.State) {
 case "connected": case "ready": return "active";
 case "connecting": case "attention": return "attention";
 case "error": return "critical";
 case "disconnected": return detail(conn,"Required")==="Yes" ? "critical" : "";
 default: return "muted";
 }
}
export function connectionText(control) {
 const c=control.Connection||{}, lines=[control.Label+" ("+(control.Group||"")+")","State: "+(control.Value||c.State||"")];
 if (c.Endpoint) lines.push("Endpoint: "+c.Endpoint);
 for (const [label,iso] of [["Since",c.Since],["Last activity",c.LastActivity]]) if (timeValue(iso)!==null) lines.push(label+": "+new Date(timeValue(iso)).toISOString());
 if (c.LastError) lines.push("Last error: "+c.LastError+(timeValue(c.LastErrorAt)!==null?" ("+new Date(timeValue(c.LastErrorAt)).toISOString()+")":""));
 for (const d of c.Details||[]) lines.push(d.Label+": "+d.Value);
 return lines.join("\n");
}
