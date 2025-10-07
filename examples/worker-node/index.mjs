const API = process.env.API || "http://localhost:8080";
const KEY = process.env.KEY || "dev-key";

async function sleep(ms){ return new Promise(r=>setTimeout(r,ms)); }

async function safeJSON(res) {
  const txt = await res.text();
  try { return JSON.parse(txt); } catch { return { _raw: txt }; }
}

async function loop() {
  for(;;) {
    try {
      const leaseRes = await fetch(`${API}/v1/lease`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${KEY}`, "Content-Type": "application/json" },
        body: JSON.stringify({ workerId: "node-worker-1", capabilities: ["email.send"], maxBatch: 1 })
      });

      if (!leaseRes.ok) {
        const body = await leaseRes.text();
        console.error("Lease HTTP", leaseRes.status, body);
        await sleep(1000);
        continue;
      }

      const data = await safeJSON(leaseRes);
      if (!data || !data.job) { await sleep(300); continue; }

      const job = data.job;
      console.log("Leased:", job.id, job.type);

      // simulate work; sometimes fail to exercise retries
      await sleep(200);
      if (Math.random() < 0.2) {
        const failRes = await fetch(`${API}/v1/fail`, {
          method: "POST",
          headers: { "Authorization": `Bearer ${KEY}`, "Content-Type": "application/json" },
          body: JSON.stringify({ workerId: "node-worker-1", jobId: job.id, error: "random fail", retryable: true })
        });
        console.log("Failed (retry scheduled):", job.id, failRes.status);
        continue;
      }

      const done = await fetch(`${API}/v1/complete`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${KEY}`, "Content-Type": "application/json" },
        body: JSON.stringify({ workerId: "node-worker-1", jobId: job.id })
      });
      console.log(done.ok ? "Completed:" : "Complete HTTP "+done.status, job.id);
    } catch (e) {
      console.error("Worker network/error:", e?.message || e);
      await sleep(1000);
    }
  }
}
loop();