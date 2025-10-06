import fetch from 'node-fetch'
const API = process.env.API || 'http://localhost:8080'
const KEY = process.env.KEY


async function loop(){
while(true){
const res = await fetch(`${API}/v1/lease`, {method:'POST', headers:{'Authorization':`Bearer ${KEY}`}})
if(res.status!==200){ await sleep(1000); continue }
const { job } = await res.json()
if(!job){ await sleep(500); continue }
try {
// do work
await sleep(500)
await fetch(`${API}/v1/complete`, {method:'POST', headers:{'Authorization':`Bearer ${KEY}`,'Content-Type':'application/json'}, body:JSON.stringify({jobId:job.id})})
} catch(e){
await fetch(`${API}/v1/fail`, {method:'POST', headers:{'Authorization':`Bearer ${KEY}`,'Content-Type':'application/json'}, body:JSON.stringify({jobId:job.id, error:String(e), retryable:true})})
}
}
}
function sleep(ms){ return new Promise(r=>setTimeout(r,ms)) }
loop()